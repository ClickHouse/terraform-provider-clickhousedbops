package resourcebuilder

import (
	"strings"
	"testing"
)

func TestResourcebuilder_Build(t *testing.T) {
	tests := []struct {
		name                   string
		resourceType           string
		resourceName           string
		stringAttributes       map[string]string
		resourceFieldReference map[string]struct {
			resourceType string
			resourceName string
			fieldName    string
		}
		functionCalls map[string]struct {
			function string
			arg      string
		}
		want string
	}{
		{
			name:         "Empty resource",
			resourceType: "test",
			resourceName: "foo",
			want: `resource "test" "foo" {
}`,
		},
		{
			name:         "Resource with string attribute",
			resourceType: "test",
			resourceName: "foo",
			stringAttributes: map[string]string{
				"name": "john",
			},
			want: `resource "test" "foo" {
  name = "john"
}`,
		},
		{
			name:         "Resource with resource field reference attribute",
			resourceType: "test",
			resourceName: "foo",
			resourceFieldReference: map[string]struct {
				resourceType string
				resourceName string
				fieldName    string
			}{
				"vpc": {
					resourceType: "aws_vpc",
					resourceName: "vpc1",
					fieldName:    "id",
				},
			},
			want: `resource "test" "foo" {
  vpc = aws_vpc.vpc1.id
}`,
		},
		{
			name:         "Resource with resource field reference attribute",
			resourceType: "test",
			resourceName: "foo",
			functionCalls: map[string]struct {
				function string
				arg      string
			}{
				"hash": {
					function: "sha256",
					arg:      "test",
				},
			},
			want: `resource "test" "foo" {
  hash = sha256("test")
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.resourceType, tt.resourceName)

			for n, v := range tt.stringAttributes {
				r.WithStringAttribute(n, v)
			}

			for n, v := range tt.resourceFieldReference {
				r.WithResourceFieldReference(n, v.resourceType, v.resourceName, v.fieldName)
			}

			for n, v := range tt.functionCalls {
				r.WithFunction(n, v.function, v.arg)
			}

			if got := strings.TrimRight(r.Build(), "\n"); got != tt.want {
				t.Errorf("Build() = %q, want %q", got, tt.want)
			}
		})
	}
}
