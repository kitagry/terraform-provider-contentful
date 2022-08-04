package contentful

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	contentful "github.com/kitagry/contentful-go"
)

func TestAccContentfulContentType_Basic(t *testing.T) {
	var contentType contentful.ContentType

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContentfulContentTypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccContentfulContentTypeConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contentful_contenttype.mycontenttype", "name", "tf_test1"),
					testAccCheckContentfulContentTypeExists("contentful_contenttype.mycontenttype", &contentType),
				),
			},
			{
				Config: testAccContentfulContentTypeUpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contentful_contenttype.mycontenttype", "name", "tf_test1"),
					testAccCheckContentfulContentTypeExists("contentful_contenttype.mycontenttype", &contentType),
				),
			},
			{
				Config: testAccContentfulContentTypeLinkConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contentful_contenttype.mycontenttype", "name", "tf_test1"),
					resource.TestCheckResourceAttr("contentful_contenttype.mylinked_contenttype", "name", "tf_linked"),
					testAccCheckContentfulContentTypeExists("contentful_contenttype.mycontenttype", &contentType),
					testAccCheckContentfulContentTypeExists("contentful_contenttype.mylinked_contenttype", &contentType),
				),
			},
			{
				Config: testAccContentfulContentTypeWithID,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contentful_contenttype.content_type_with_id", "name", "tf_test_with_id"),
					testAccCheckContentfulContentTypeExists("contentful_contenttype.content_type_with_id", &contentType),
				),
			},
		},
	})
}

func TestAccContentfulContentType_TypeChanged(t *testing.T) {
	var contentType contentful.ContentType

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContentfulContentTypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccContentfulContentTypeConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contentful_contenttype.mycontenttype", "name", "tf_test1"),
					testAccCheckContentfulContentTypeExists("contentful_contenttype.mycontenttype", &contentType),
				),
			},
			{
				Config: testAccContentfulContentTypeUpdateFieldType,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("contentful_contenttype.mycontenttype", "name", "tf_test1"),
					testAccCheckContentfulContentTypeExists("contentful_contenttype.mycontenttype", &contentType),
				),
			},
		},
	})
}

// noinspection GoUnusedFunction
func testAccCheckContentfulContentTypeExists(n string, contentType *contentful.ContentType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no content type ID is set")
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("no space_id is set")
		}

		envID := rs.Primary.Attributes["env_id"]
		if envID == "" {
			return fmt.Errorf("no env_id is set")
		}

		client := testAccProvider.Meta().(*contentful.Client)

		env := &contentful.Environment{
			Sys: &contentful.Sys{
				ID: envID,
				Space: &contentful.Space{
					Sys: &contentful.Sys{
						ID: spaceID,
					},
				},
			},
		}

		ct, err := client.ContentTypes.Get(context.Background(), env, rs.Primary.ID)
		if err != nil {
			return err
		}

		*contentType = *ct

		return nil
	}
}

func testAccCheckContentfulContentTypeDestroy(s *terraform.State) (err error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contentful_contenttype" {
			continue
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("no space_id is set")
		}

		envID := rs.Primary.Attributes["env_id"]
		if envID == "" {
			return fmt.Errorf("no env_id is set")
		}

		client := testAccProvider.Meta().(*contentful.Client)

		env := &contentful.Environment{
			Sys: &contentful.Sys{
				ID: envID,
				Space: &contentful.Space{
					Sys: &contentful.Sys{
						ID: spaceID,
					},
				},
			},
		}

		_, err := client.ContentTypes.Get(context.Background(), env, rs.Primary.ID)
		if _, ok := err.(contentful.NotFoundError); ok {
			return nil
		}

		return fmt.Errorf("content type still exists with id: %s", rs.Primary.ID)
	}

	return nil
}

var testAccContentfulContentTypeConfig = `
resource "contentful_contenttype" "mycontenttype" {
	space_id = "` + spaceID + `"
	env_id = "` + envID + `"
	name = "tf_test1"
	description = "Terraform Acc Test Content Type"
	display_field = "field1"
	field {
		disabled  = false
		id        = "field1"
		localized = false
		name      = "Field 1"
		omitted   = false
		required  = true
		type      = "Text"
	}
	field {
		disabled  = false
		id        = "field2"
		localized = false
		name      = "Field 2"
		omitted   = false
		required  = false
		type      = "Integer"
	}
}
`

var testAccContentfulContentTypeUpdateConfig = `
resource "contentful_contenttype" "mycontenttype" {
	space_id = "` + spaceID + `"
	env_id = "` + envID + `"
	name = "tf_test1"
	description = "Terraform Acc Test Content Type description change"
	display_field = "field1"
	field {
		disabled  = false
		id        = "field1"
		localized = false
		name      = "Field 1 name change"
		omitted   = false
		required  = true
		type      = "Text"
	}
	field {
		disabled  = false
		id        = "field3"
		localized = false
		name      = "Field 3 new field"
		omitted   = false
		required  = true
		type      = "Integer"
	}
}
`

var testAccContentfulContentTypeLinkConfig = `
resource "contentful_contenttype" "mycontenttype" {
	space_id = "` + spaceID + `"
	env_id = "` + envID + `"
	name = "tf_test1"
	description = "Terraform Acc Test Content Type description change"
	display_field = "field1"
	content_type_id = "tf_test1"
	field {
		disabled  = false
		id        = "field1"
		localized = false
		name      = "Field 1 name change"
		omitted   = false
		required  = true
		type      = "Text"
	}
	field {
		disabled  = false
		id        = "field3"
		localized = false
		name      = "Field 3 new field"
		omitted   = false
		required  = true
		type      = "Integer"
	}
}

resource "contentful_contenttype" "mylinked_contenttype" {
	space_id = "` + spaceID + `"
	env_id = "` + envID + `"
	name          = "tf_linked"
	description   = "Terraform Acc Test Content Type with links"
	display_field = "entry_link_field"
	field {
	  id   = "asset_field"
	  name = "Asset Field"
	  type = "Array"
	  items {
	    type      = "Link"
	    link_type = "Asset"
	  }
	  required = true
	}
	field {
	  id        = "entry_link_field"
	  name      = "Entry Link Field"
	  type      = "Link"
	  link_type = "Entry"
	  validations = [
			jsonencode({
				linkContentType = [
					"tf_test1"
				]
			})
		]
	  required = false
	}
}
`

var testAccContentfulContentTypeWithID = `
resource "contentful_contenttype" "content_type_with_id" {
	space_id = "` + spaceID + `"
	env_id = "` + envID + `"
	name = "tf_test_with_id"
	description = "Content Type with ID"
	content_type_id = "contentTypeWithID"
	display_field = "field1"
	field {
		disabled  = false
		id        = "field1"
		localized = false
		name      = "Field 1 name"
		omitted   = false
		required  = true
		type      = "Text"
	}
	field {
		disabled  = false
		id        = "field2"
		localized = false
		name      = "Field 2 name"
		omitted   = false
		required  = true
		type      = "Symbol"
	}
}
`

var testAccContentfulContentTypeUpdateFieldType = `
resource "contentful_contenttype" "mycontenttype" {
	space_id = "` + spaceID + `"
	env_id = "` + envID + `"
	name = "tf_test1"
	description = "Terraform Acc Test Content Type"
	display_field = "field1"
	field {
		disabled  = false
		id        = "field1"
		localized = false
		name      = "Field 1"
		omitted   = false
		required  = true
		type      = "Text"
	}
	field {
		disabled  = false
		id        = "field2"
		localized = false
		name      = "Field 2"
		omitted   = false
		required  = false
		type      = "Integer" // This field is changed

		validations = [
			jsonencode({
				range = {
					min = 1
				}
			})
		]
	}
}
`
