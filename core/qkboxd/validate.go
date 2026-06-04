package qkboxd

import (
	"encoding/json"

	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

func validateContent(content string) model.Diagnostics {
	var entries []model.ValidationDiagnostic

	var raw interface{}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		entries = append(entries, model.ValidationDiagnostic{
			Severity: model.SeverityError,
			Field:    "",
			Message:  "Invalid JSON: " + err.Error(),
		})
		return model.Diagnostics{
			Status:  model.ValidationStatusInvalid,
			Entries: entries,
		}
	}

	obj, ok := raw.(map[string]interface{})
	if !ok {
		entries = append(entries, model.ValidationDiagnostic{
			Severity: model.SeverityError,
			Field:    "",
			Message:  "Top-level value must be a JSON object.",
		})
		return model.Diagnostics{
			Status:  model.ValidationStatusInvalid,
			Entries: entries,
		}
	}

	validateArrayField(obj, "inbounds", &entries)
	validateArrayField(obj, "outbounds", &entries)

	status := model.ValidationStatusValid
	for _, e := range entries {
		if e.Severity == model.SeverityError {
			status = model.ValidationStatusInvalid
			break
		}
	}

	return model.Diagnostics{
		Status:  status,
		Entries: entries,
	}
}

func validateArrayField(obj map[string]interface{}, field string, entries *[]model.ValidationDiagnostic) {
	val, exists := obj[field]
	if !exists {
		*entries = append(*entries, model.ValidationDiagnostic{
			Severity: model.SeverityError,
			Field:    field,
			Message:  "Missing required '" + field + "' field.",
		})
		return
	}
	if _, ok := val.([]interface{}); !ok {
		*entries = append(*entries, model.ValidationDiagnostic{
			Severity: model.SeverityError,
			Field:    field,
			Message:  "'" + field + "' must be an array.",
		})
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
