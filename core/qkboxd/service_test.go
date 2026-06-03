package qkboxd

import (
	"context"
	"testing"

	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/shared/api"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	db, err := persistence.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	key, err := qkboxcrypto.RandomBytes(qkboxcrypto.KeySize)
	if err != nil {
		t.Fatal(err)
	}
	return NewService(db, key)
}

func TestHelloReturnsCapabilityShells(t *testing.T) {
	svc := newTestService(t)
	reply, err := svc.Hello(context.Background(), api.DefaultHelloRequest())
	if err != nil {
		t.Fatal(err)
	}
	if reply.APIVersion != api.APIVersion {
		t.Fatalf("api version = %s", reply.APIVersion)
	}
	if len(reply.RuntimeCapabilities) == 0 || len(reply.PlatformCapabilities) == 0 {
		t.Fatal("expected capability shells")
	}
	for _, capability := range append(reply.RuntimeCapabilities, reply.PlatformCapabilities...) {
		if capability.State == "" {
			t.Fatalf("capability %s has empty state", capability.Name)
		}
	}
}

func TestHelloRejectsUnsupportedAPIVersion(t *testing.T) {
	svc := newTestService(t)
	req := api.DefaultHelloRequest()
	req.ClientAPIVersion = "0"
	_, err := svc.Hello(context.Background(), req)
	if err == nil {
		t.Fatal("expected structured error")
	}
	if err.Code != api.ErrorIPCVersionUnsupported {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestProfileCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// create
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "test-profile",
		Content: `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	// get
	getReply, err := svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getReply.Profile.Name != "test-profile" {
		t.Fatalf("name = %s", getReply.Profile.Name)
	}
	if getReply.Content == "" {
		t.Fatal("expected content")
	}

	// update
	_, err = svc.UpdateProfileDraft(ctx, api.UpdateProfileDraftRequest{
		ProfileID: pid,
		Content:   `{"inbounds":[],"outbounds":[{"type":"block"}]}`,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// list
	listReply, err := svc.ListProfiles(ctx, api.ListProfilesRequest{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listReply.Profiles) != 1 {
		t.Fatalf("count = %d", len(listReply.Profiles))
	}

	// delete
	_, err = svc.DeleteProfile(ctx, api.DeleteProfileRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// get after delete
	_, err = svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected error after delete")
	}
	if err.Code != api.ErrorProfileNotFound {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestSnapshotLifecycle(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// create profile
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "snap-test",
		Content: `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	pid := createReply.Profile.ID

	// validate
	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "valid" {
		t.Fatalf("status = %s", validReply.Diagnostics.Status)
	}

	// create snapshot
	snapReply, err := svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	sid := snapReply.Snapshot.ID

	// list snapshots
	listReply, err := svc.ListSnapshots(ctx, api.ListSnapshotsRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(listReply.Snapshots) != 1 {
		t.Fatalf("snapshot count = %d", len(listReply.Snapshots))
	}

	// activate
	_, err = svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: sid})
	if err != nil {
		t.Fatalf("activate: %v", err)
	}

	// get active
	activeReply, err := svc.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if err != nil {
		t.Fatalf("get active profile: %v", err)
	}
	if activeReply.Profile == nil {
		t.Fatal("expected active profile")
	}
	if activeReply.Profile.ID != pid {
		t.Fatalf("active profile id = %s", activeReply.Profile.ID)
	}

	activeSnapReply, err := svc.GetActiveSnapshot(ctx, api.GetActiveSnapshotRequest{})
	if err != nil {
		t.Fatalf("get active snapshot: %v", err)
	}
	if activeSnapReply.Snapshot == nil {
		t.Fatal("expected active snapshot")
	}
	if activeSnapReply.Snapshot.ID != sid {
		t.Fatalf("active snapshot id = %s", activeSnapReply.Snapshot.ID)
	}

	// rollback (same snapshot)
	_, err = svc.RollbackToSnapshot(ctx, api.RollbackToSnapshotRequest{SnapshotID: sid})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
}

func TestValidationBlocksInvalidSnapshot(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "bad-profile",
		Content: `not json`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	// validate shows invalid
	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid, got %s", validReply.Diagnostics.Status)
	}

	// snapshot blocked
	_, err = svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestValidationBlocksEmptyObject(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "empty-obj",
		Content: `{}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid for empty object, got %s", validReply.Diagnostics.Status)
	}

	_, err = svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected snapshot blocked for empty object")
	}
	if err.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestValidationBlocksNonArrayFields(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "non-array",
		Content: `{"inbounds":"not-an-array","outbounds":123}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid, got %s", validReply.Diagnostics.Status)
	}

	_, err = svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected snapshot blocked")
	}
	if err.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestContentIsEncrypted(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	content := `{"secret":"my-password-123"}`
	_, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "encrypted-test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// retrieve encrypted content via repo and verify it's not plaintext
	contents, listErr := svc.db.ListAllContent()
	if listErr != nil {
		t.Fatalf("list content: %v", listErr)
	}
	if len(contents) == 0 {
		t.Fatal("no content stored")
	}
	for _, c := range contents {
		raw := string(c.Ciphertext)
		if raw == content {
			t.Fatal("content stored in plaintext")
		}
		if len(c.Ciphertext) == 0 {
			t.Fatal("empty ciphertext")
		}
		if len(c.IV) == 0 {
			t.Fatal("empty IV")
		}
	}

	// verify we can still decrypt and get the original content
	getReply, err := svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: contents[0].SourceID})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getReply.Content != content {
		t.Fatalf("decrypted content mismatch: got %q", getReply.Content)
	}
}
