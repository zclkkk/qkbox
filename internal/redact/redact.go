package redact

import (
	"encoding/json"
	"strings"
)

var sensitiveFieldNames = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"auth",
	"key",
	"private_key",
	"privatekey",
	"private-key",
	"apikey",
	"api_key",
	"api-key",
	"credential",
	"credentials",
}

const redactedValue = "***REDACTED***"

func Content(content string) string {
	var raw interface{}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return "[invalid JSON]"
	}
	redacted := scrubValue(raw)
	b, err := json.Marshal(redacted)
	if err != nil {
		return "[redaction error]"
	}
	return string(b)
}

func scrubValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v2 := range val {
			if isSensitiveField(k) {
				out[k] = redactedValue
			} else {
				out[k] = scrubValue(v2)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, v2 := range val {
			out[i] = scrubValue(v2)
		}
		return out
	default:
		return val
	}
}

func isSensitiveField(name string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range sensitiveFieldNames {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
