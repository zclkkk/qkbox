package api

const (
	MethodHello = "hello"

	// Profile CRUD
	MethodCreateProfile     = "createProfile"
	MethodUpdateProfileDraft = "updateProfileDraft"
	MethodDeleteProfile     = "deleteProfile"
	MethodListProfiles      = "listProfiles"
	MethodGetProfile        = "getProfile"

	// Snapshot lifecycle
	MethodValidateProfileDraft    = "validateProfileDraft"
	MethodGetProfileDiagnostics   = "getProfileDiagnostics"
	MethodCreateProfileSnapshot   = "createProfileSnapshot"
	MethodActivateProfileSnapshot = "activateProfileSnapshot"
	MethodGetActiveProfile        = "getActiveProfile"
	MethodGetActiveSnapshot       = "getActiveSnapshot"
	MethodListSnapshots           = "listSnapshots"
	MethodRollbackToSnapshot      = "rollbackToSnapshot"
)

var MethodRegistry = map[string]struct{}{
	MethodHello:                   {},
	MethodCreateProfile:           {},
	MethodUpdateProfileDraft:      {},
	MethodDeleteProfile:           {},
	MethodListProfiles:            {},
	MethodGetProfile:              {},
	MethodValidateProfileDraft:    {},
	MethodGetProfileDiagnostics:   {},
	MethodCreateProfileSnapshot:   {},
	MethodActivateProfileSnapshot: {},
	MethodGetActiveProfile:        {},
	MethodGetActiveSnapshot:       {},
	MethodListSnapshots:           {},
	MethodRollbackToSnapshot:      {},
}
