package clickhouseclient

import (
	"sort"
	"strings"
)

func redactSensitiveValues(value string, sensitiveValues []string) string {
	values := append([]string(nil), sensitiveValues...)
	sort.Slice(values, func(i, j int) bool {
		return len(values[i]) > len(values[j])
	})
	for _, sensitive := range values {
		if sensitive != "" {
			value = strings.ReplaceAll(value, sensitive, "[REDACTED]")
		}
	}
	return value
}
