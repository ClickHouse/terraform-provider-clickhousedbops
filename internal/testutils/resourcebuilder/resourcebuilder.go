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

func (r *ResourceBuilder) WithFunction(attrName string, function string, args ...string) *ResourceBuilder {
	r.getRootResourceBody().SetAttributeRaw(attrName, functionTokens(function, args))

	return r
}

func (r *ResourceBuilder) WithBlock(name string, fn func(*BlockBuilder)) *ResourceBuilder {
	block := r.getRootResourceBody().AppendNewBlock(name, nil)
	fn(&BlockBuilder{body: block.Body()})

	return r
}

func (r *ResourceBuilder) WithListAttribute(attrName string, data []cty.Value) *ResourceBuilder {
	r.getRootResourceBody().SetAttributeValue(attrName, cty.ListVal(data))

	return r
}

func (r *ResourceBuilder) WithEmptyListAttribute(attrName string) *ResourceBuilder {
	r.getRootResourceBody().SetAttributeValue(attrName, cty.ListValEmpty(cty.String))

	return r
}

func (r *ResourceBuilder) WithListResourceFieldReference(attrName string, resourceType string, resourceName string, fieldName string) *ResourceBuilder {
	// Create a list with a single resource field reference: [resource.name.field]
	r.getRootResourceBody().SetAttributeRaw(attrName, hclwrite.Tokens{
		{Type: hclsyntax.TokenOBrack, Bytes: []byte("[")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(resourceType)},
		{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(resourceName)},
		{Type: hclsyntax.TokenDot, Bytes: []byte(".")},
		{Type: hclsyntax.TokenIdent, Bytes: []byte(fieldName)},
		{Type: hclsyntax.TokenCBrack, Bytes: []byte("]")},
	})

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

// BlockBuilder is a helper
type BlockBuilder struct {
	body *hclwrite.Body
}

func (b *BlockBuilder) WithStringAttribute(attrName string, attrVal string) *BlockBuilder {
	b.body.SetAttributeValue(attrName, cty.StringVal(attrVal))

	return b
}

func (b *BlockBuilder) WithIntAttribute(attrName string, attrVal int64) *BlockBuilder {
	b.body.SetAttributeValue(attrName, cty.NumberIntVal(attrVal))

	return b
}

func (b *BlockBuilder) WithBoolAttribute(attrName string, attrVal bool) *BlockBuilder {
	b.body.SetAttributeValue(attrName, cty.BoolVal(attrVal))

	return b
}

func (b *BlockBuilder) WithFunction(attrName string, function string, args ...string) *BlockBuilder {
	b.body.SetAttributeRaw(attrName, functionTokens(function, args))

	return b
}

// WithBlock appends a nested block inside this block. Call it more than once with the same name to emit repeated blocks.
func (b *BlockBuilder) WithBlock(name string, fn func(*BlockBuilder)) *BlockBuilder {
	nested := b.body.AppendNewBlock(name, nil)
	fn(&BlockBuilder{body: nested.Body()})

	return b
}

func functionTokens(function string, args []string) hclwrite.Tokens {
	tokens := hclwrite.Tokens{
		{Type: hclsyntax.TokenIdent, Bytes: []byte(function)},
		{Type: hclsyntax.TokenOParen, Bytes: []byte("(")},
	}

	for i, arg := range args {
		if i != 0 {
			tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenComma, Bytes: []byte(",")})
		}

		tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(fmt.Sprintf("%q", arg))})
	}

	tokens = append(tokens, &hclwrite.Token{Type: hclsyntax.TokenCParen, Bytes: []byte(")")})
	return tokens
}
