package qkboxd

import (
	"encoding/json"

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

	if _, exists := obj["inbounds"]; !exists {
		entries = append(entries, model.ValidationDiagnostic{
			Severity: model.SeverityWarning,
			Field:    "inbounds",
			Message:  "Missing 'inbounds' field.",
		})
	}

	if _, exists := obj["outbounds"]; !exists {
		entries = append(entries, model.ValidationDiagnostic{
			Severity: model.SeverityWarning,
			Field:    "outbounds",
			Message:  "Missing 'outbounds' field.",
		})
	}

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
