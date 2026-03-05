package aiclientv2

import (
	"encoding/json"
	"fmt"
	"strings"
)

func extractField(jsonStr, field string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return ""
	}
	if v, ok := data[field]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func simpleText(content, fallbackKey string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return content
	}
	candidates := []string{
		fallbackKey, "message", "summary", "result", "content", "error", "reason", "prompt",
	}
	for _, key := range candidates {
		if v, ok := obj[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return content
}
