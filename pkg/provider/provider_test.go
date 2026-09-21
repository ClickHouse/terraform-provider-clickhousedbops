package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderSchema_HasTimeoutAttribute(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	attr, ok := resp.Schema.Attributes["dial_timeout"]
	if !ok {
		t.Fatal("expected 'dial_timeout' attribute in provider schema, not found")
	}

	if attr.IsRequired() {
		t.Error("'dial_timeout' attribute should be optional, not required")
	}
}

func TestProviderSchema_HasReadAfterWriteTimeoutAttribute(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	if _, ok := resp.Schema.Attributes["read_after_write_timeout"]; !ok {
		t.Fatal("expected 'read_after_write_timeout' attribute in provider schema, not found")
	}
}

func TestProviderSchema_RequiredAttributes(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	required := []string{"protocol", "host", "port", "auth_config"}
	for _, name := range required {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected required attribute %q not found", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %q should be required", name)
		}
	}
}

func TestProviderResources_IncludesRevokePrivilege(t *testing.T) {
	p := &Provider{}
	for _, factory := range p.Resources(context.Background()) {
		r := factory()
		var resp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "clickhousedbops"}, &resp)
		if resp.TypeName == "clickhousedbops_revoke_privilege" {
			return
		}
	}
	t.Fatal("clickhousedbops_revoke_privilege resource is not registered")
}
