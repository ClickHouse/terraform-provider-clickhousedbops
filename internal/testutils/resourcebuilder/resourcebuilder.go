package resourcebuilder

import (
	"fmt"
	"strings"
)

type attribute struct {
	value   string
	literal bool
}

func (a *attribute) String() string {
	if a.literal {
		return a.value
	}

	return fmt.Sprintf("%q", a.value)
}

type Resourcebuilder struct {
	resourceType string
	resourceName string

	attributes map[string]attribute
}

func New(resourceType string, resourceName string) *Resourcebuilder {
	return &Resourcebuilder{
		resourceType: resourceType,
		resourceName: resourceName,

		attributes: make(map[string]attribute),
	}
}

func (r *Resourcebuilder) WithStringAttribute(attrName string, attrVal string) *Resourcebuilder {
	r.attributes[attrName] = attribute{
		value:   attrVal,
		literal: false,
	}

	return r
}

func (r *Resourcebuilder) WithLiteralAttribute(attrName string, attrVal interface{}) *Resourcebuilder {
	r.attributes[attrName] = attribute{
		value:   fmt.Sprintf("%v", attrVal),
		literal: true,
	}

	return r
}

func (r *Resourcebuilder) Build() string {
	attributes := make([]string, 0)
	for k, v := range r.attributes {
		attributes = append(attributes, fmt.Sprintf("  %s = %s", k, v.String()))
	}

	return fmt.Sprintf(`resource "%s" "%s" {
%s
}`, r.resourceType, r.resourceName, strings.Join(attributes, "\n"))
}
