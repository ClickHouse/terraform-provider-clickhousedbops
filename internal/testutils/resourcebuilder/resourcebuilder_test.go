package resourcebuilder

import (
	"testing"
)

func TestResourcebuilder_Build(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		resourceName string
		attributes   map[string]attribute
		want         string
	}{
		{
			name:         "Empty resource",
			resourceType: "test",
			resourceName: "foo",
			attributes:   nil,
			want: `resource "test" "foo" {

}`,
		},
		{
			name:         "Resource with string attribute",
			resourceType: "test",
			resourceName: "foo",
			attributes: map[string]attribute{
				"name": {
					value:   "john",
					literal: false,
				},
			},
			want: `resource "test" "foo" {
  name = "john"
}`,
		},
		{
			name:         "Resource with literal string attribute",
			resourceType: "test",
			resourceName: "foo",
			attributes: map[string]attribute{
				"name": {
					value:   `sha256("test")`,
					literal: true,
				},
			},
			want: `resource "test" "foo" {
  name = sha256("test")
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Resourcebuilder{
				resourceType: tt.resourceType,
				resourceName: tt.resourceName,
				attributes:   tt.attributes,
			}
			if got := r.Build(); got != tt.want {
				t.Errorf("Build() = %s, want %s", got, tt.want)
			}
		})
	}
}
