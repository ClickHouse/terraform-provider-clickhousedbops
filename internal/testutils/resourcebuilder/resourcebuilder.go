package resourcebuilder

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// ResourceBuilder is an helper to build terraform definitions like:
//
//	resource "resource_type" "name" {
//	  attribute = "value"
//	  attribute1 = 3
//	}
//
// and return them as strings.
// Used in acceptance tests to build test resource definitions.
type ResourceBuilder struct {
	resourceType string
	resourceName string

	dependencies []string

	file *hclwrite.File
}

func New(resourceType string, resourceName string) *ResourceBuilder {
	file := hclwrite.NewEmptyFile()

	rootBody := file.Body()
	rootBody.AppendNewBlock("resource", []string{resourceType, resourceName})

	return &ResourceBuilder{
		resourceType: resourceType,
		resourceName: resourceName,

		file: file,
	}
}

func (r *ResourceBuilder) WithStringAttribute(attrName string, attrVal string) *ResourceBuilder {
	r.getRootResourceBody().SetAttributeValue(attrName, cty.StringVal(attrVal))

	return r
}

func (r *ResourceBuilder) WithIntAttribute(attrName string, attrVal int64) *ResourceBuilder {
	r.getRootResourceBody().SetAttributeValue(attrName, cty.NumberIntVal(attrVal))

	return r
}

func (r *ResourceBuilder) WithBoolAttribute(attrName string, attrVal bool) *ResourceBuilder {
	r.getRootResourceBody().SetAttributeValue(attrName, cty.BoolVal(attrVal))

	return r
}

func (r *ResourceBuilder) WithResourceFieldReference(attrName string, resourceType string, resourceName string, fieldName string) *ResourceBuilder {
	// Reference to another resource
	r.getRootResourceBody().SetAttributeTraversal(attrName, hcl.Traversal{
		hcl.TraverseRoot{Name: resourceType},
		hcl.TraverseAttr{Name: resourceName},
		hcl.TraverseAttr{Name: fieldName},
	})

	return r
}

func (r *ResourceBuilder) WithFunction(attrName string, function string, arg string) *ResourceBuilder {
	// function call
	r.getRootResourceBody().SetAttributeRaw(attrName, hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(function)},
		{Type: hclsyntax.TokenOParen, Bytes: []byte("(")},
		{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(fmt.Sprintf("%q", arg))},
		{Type: hclsyntax.TokenCParen, Bytes: []byte(")")},
	})

	return r
}

func (r *ResourceBuilder) WithListAttribute(attrName string, data []cty.Value) *ResourceBuilder {
	r.getRootResourceBody().SetAttributeValue(attrName, cty.ListVal(data))

	return r
}

func (r *ResourceBuilder) AddDependency(resource string) *ResourceBuilder {
	r.dependencies = append(r.dependencies, resource)
	return r
}

func (r *ResourceBuilder) Build() string {
	tokens := make([]string, 0)
	tokens = append(tokens, r.dependencies...)
	tokens = append(tokens, string(r.file.Bytes()))

	return strings.Join(tokens, "\n")
}

func (r *ResourceBuilder) getRootResourceBody() *hclwrite.Body {
	return r.file.Body().FirstMatchingBlock("resource", []string{r.resourceType, r.resourceName}).Body()
}
