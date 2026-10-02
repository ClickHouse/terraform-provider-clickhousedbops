package dbops

import "strings"

// splitSystemAccessObject separates the representation used by
// system.grants.access_object for source filters (for example
// URL(`https://example.com/.*`)) into the source name and regexp. Other
// parameterized targets are returned unchanged.
func splitSystemAccessObject(accessType string, accessObject *string) (*string, *string) {
	if accessObject == nil || accessType != "READ" && accessType != "WRITE" {
		return accessObject, nil
	}

	value := *accessObject
	open := strings.IndexByte(value, '(')
	if open <= 0 || !strings.HasSuffix(value, ")") {
		return accessObject, nil
	}

	name := value[:open]
	if !sourcesFamily[strings.ToUpper(name)] {
		return accessObject, nil
	}

	formattedFilter := value[open+1 : len(value)-1]
	filter, ok := parseSystemAccessObjectFilter(formattedFilter)
	if !ok {
		// Preserve an unfamiliar representation as an opaque access object. It
		// must never be mistaken for an unfiltered (and therefore broader) grant.
		return accessObject, nil
	}

	return new(name), new(filter)
}

func parseSystemAccessObjectFilter(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	if value[0] != '`' {
		return value, true
	}
	if len(value) < 2 || value[len(value)-1] != '`' {
		return "", false
	}

	value = value[1 : len(value)-1]
	var result strings.Builder
	result.Grow(len(value))

	for i := 0; i < len(value); i++ {
		if value[i] != '\\' {
			result.WriteByte(value[i])
			continue
		}
		if i+1 == len(value) {
			return "", false
		}
		i++
		switch value[i] {
		case '0':
			result.WriteByte(0)
		case 'a':
			result.WriteByte('\a')
		case 'b':
			result.WriteByte('\b')
		case 't':
			result.WriteByte('\t')
		case 'n':
			result.WriteByte('\n')
		case 'v':
			result.WriteByte('\v')
		case 'f':
			result.WriteByte('\f')
		case 'r':
			result.WriteByte('\r')
		case 'e':
			result.WriteByte(0x1b)
		default:
			// ClickHouse backquoted strings escape both '\\' and '`' with a
			// backslash. Preserve the escaped byte for any future escape form.
			result.WriteByte(value[i])
		}
	}

	return result.String(), true
}

func accessObjectMatches(
	accessType string,
	expectedObject, expectedFilter, actualObject *string,
) bool {
	actualObject, actualFilter := splitSystemAccessObject(accessType, actualObject)
	if !accessObjectNameMatches(accessType, expectedObject, actualObject) {
		return false
	}
	return equalStringPointers(expectedFilter, actualFilter)
}

func accessObjectNameMatches(accessType string, expected, actual *string) bool {
	if expected == nil || actual == nil {
		return expected == nil && actual == nil
	}

	expectedValue := strings.TrimSuffix(*expected, "*")
	if accessType == "READ" || accessType == "WRITE" {
		return strings.EqualFold(expectedValue, *actual)
	}
	return expectedValue == *actual
}

func equalStringPointers(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
