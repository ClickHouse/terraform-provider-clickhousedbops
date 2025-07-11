package nilcompare

import (
	"reflect"
)

// NilCompare compares a and b's values and considers pointers and nil values as well.
func NilCompare(a interface{}, b interface{}) bool {
	// Use reflection to handle the different cases
	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	if aVal.Kind() == reflect.Ptr && aVal.IsNil() {
		a = nil
	}

	if bVal.Kind() == reflect.Ptr && bVal.IsNil() {
		b = nil
	}

	// Both nil or nil pointers
	if a == nil && b == nil {
		return true
	}

	// One is nil, one is not.
	if (a == nil) != (b == nil) {
		return false
	}

	// Check if both are pointers
	if aVal.Kind() == reflect.Ptr && bVal.Kind() == reflect.Ptr {
		// Both are pointers - compare if both are nil or both are non-nil
		if aVal.IsNil() && bVal.IsNil() {
			return true
		}
		if aVal.IsNil() || bVal.IsNil() {
			return false
		}
		// Both are non-nil pointers, compare the values they point to
		return reflect.DeepEqual(aVal.Elem().Interface(), bVal.Elem().Interface())
	}

	// Check if one is pointer, one is scalar
	if aVal.Kind() == reflect.Ptr && bVal.Kind() != reflect.Ptr {
		// Compare dereferenced pointer value with scalar
		return reflect.DeepEqual(aVal.Elem().Interface(), b)
	}

	if aVal.Kind() != reflect.Ptr && bVal.Kind() == reflect.Ptr {
		// Compare scalar with dereferenced pointer value
		return reflect.DeepEqual(a, bVal.Elem().Interface())
	}

	// Both are scalar values
	return reflect.DeepEqual(a, b)
}
