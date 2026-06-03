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
