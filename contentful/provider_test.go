package contentful

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var (
	testAccProviders map[string]terraform.ResourceProvider
	testAccProvider  *schema.Provider
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"contentful": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	_ = Provider()
}

func testAccPreCheck(t *testing.T) {
	var cmaToken, organizationID string
	if cmaToken = CMAToken; cmaToken == "" {
		t.Fatal("CONTENTFUL_MANAGEMENT_TOKEN must set with a valid Contentful Content Management API Token for acceptance tests")
	}
	if organizationID = orgID; organizationID == "" {
		t.Fatal("CONTENTFUL_ORGANIZATION_ID must set with a valid Contentful Organization ID for acceptance tests")
	}
}
