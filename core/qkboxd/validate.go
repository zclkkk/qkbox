package qkboxd

import (
	"encoding/json"

	"github.com/zclkkk/qkbox/internal/singboxadapter"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

func validateProfileConfig(profileID string, content string) model.Diagnostics {
	diag := singboxadapter.Validate(content)
	diag.ProfileID = profileID
	return diag
}

func profileConfigValidationError(message string, diag model.Diagnostics, userAction string) *api.StructuredError {
	return &api.StructuredError{
		Code:        api.ErrorConfigValidationFailed,
		Message:     message,
		Detail:      diag.Entries,
		Source:      "qkboxd",
		Recoverable: true,
		UserAction:  userAction,
	}
}

func extractRequiredCapabilities(content string) []string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	for _, inbound := range arrayObjects(obj["inbounds"]) {
		switch inbound["type"] {
		case "tun":
			seen[api.CapabilityTunMode] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	caps := make([]string, 0, len(seen))
	for cap := range seen {
		caps = append(caps, cap)
	}
	return caps
}

func arrayObjects(value interface{}) []map[string]interface{} {
	raw, ok := value.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		obj, ok := item.(map[string]interface{})
		if ok {
			out = append(out, obj)
		}
	}
	return out
}
