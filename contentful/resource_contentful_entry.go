package contentful

import (
	"context"
	"fmt"
    "strconv"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulEntry() *schema.Resource {
	return &schema.Resource{
		CreateContext: wrapEntry(resourceCreateEntry),
		ReadContext:   wrapEntry(resourceReadEntry),
		UpdateContext: wrapEntry(resourceUpdateEntry),
		DeleteContext: wrapEntry(resourceDeleteEntry),

		Schema: map[string]*schema.Schema{
			"entry_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"space_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"env_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"contenttype_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"locale": {
				Type:     schema.TypeString,
				Required: true,
			},
			"field": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"content": {
							Type: schema.TypeString,
							Required: true,
						},
						"locale": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"published": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"archived": {
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

func wrapEntry(f func(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, entryClient ContentfulEntryClient) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
		client := m.(*contentful.Client)
		spaceID := d.Get("space_id").(string)
		envID := d.Get("env_id").(string)
		env, err := client.Environments.Get(ctx, spaceID, envID)
		if err != nil {
			diags = append(diags, contentfulErrorToDiagnostic(err)...)
			return
		}
		return f(ctx, d, env, client.Entries)
	}
}

func resourceCreateEntry(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulEntryClient) (diags diag.Diagnostics) {
	fieldProperties := map[string]interface{}{}
	rawField := d.Get("field").([]interface{})
	for i := 0; i < len(rawField); i++ {
		field := rawField[i].(map[string]interface{})
		content, ok := field["content"].(string)

		if !ok {
			fmt.Println("content is not a string")
			continue
		}

		var fieldValue interface{}
		if floatValue, err := strconv.ParseFloat(content, 64); err == nil {
			// fmt.Printf("%s is a float: %f\n", content, floatValue)
			fieldValue = floatValue
		} else if intValue, err := strconv.Atoi(content); err == nil {
			// fmt.Printf("%s is an integer: %d\n", content, intValue)
			fieldValue = intValue
		} else {
			// fmt.Printf("%s is neither an integer nor a float\n", content)
			fieldValue = content
		}
			
		fieldProperties[field["id"].(string)] = map[string]interface{}{}
		fieldProperties[field["id"].(string)].(map[string]interface{})[field["locale"].(string)] = fieldValue
	}

	entry := &contentful.Entry{
		Locale: d.Get("locale").(string),
		Fields: fieldProperties,
		Sys: &contentful.Sys{
			ID: d.Get("entry_id").(string),
		},
	}

	err := client.Upsert(ctx, env, d.Get("contenttype_id").(string), entry)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err := setEntryProperties(d, entry); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(entry.Sys.ID)

	if err := setEntryState(ctx, d, env, client); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func resourceUpdateEntry(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulEntryClient) (diags diag.Diagnostics) {
	entryID := d.Id()
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	// lookup the entry
	entry, err := client.Get(ctx, env, entryID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	fieldProperties := map[string]interface{}{}
	rawField := d.Get("field").([]interface{})
	for i := 0; i < len(rawField); i++ {
		field := rawField[i].(map[string]interface{})
		content, ok := field["content"].(string)

		if !ok {
			fmt.Println("Content is not a string")
			continue
		}
		
		var fieldValue interface{}
		if floatValue, err := strconv.ParseFloat(content, 64); err == nil {
			// fmt.Printf("%s is a float: %f\n", content, floatValue)
			fieldValue = floatValue
		} else if intValue, err := strconv.Atoi(content); err == nil {
			// fmt.Printf("%s is an integer: %d\n", content, intValue)
			fieldValue = intValue
		} else {
			// fmt.Printf("%s is neither an integer nor a float\n", content)
			fieldValue = content
		}
			
		fieldProperties[field["id"].(string)] = map[string]interface{}{}
		fieldProperties[field["id"].(string)].(map[string]interface{})[field["locale"].(string)] = fieldValue
	}

	entry.Fields = fieldProperties
	entry.Locale = d.Get("locale").(string)

	err = client.Upsert(ctx, env, d.Get("contenttype_id").(string), entry)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(entry.Sys.ID)

	if err := setEntryProperties(d, entry); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	if err := setEntryState(ctx, d, env, client); err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func setEntryState(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulEntryClient) (err error) {
	entryID := d.Id()

	entry, _ := client.Get(ctx, env, entryID)

	if d.Get("published").(bool) && entry.Sys.PublishedAt == "" {
		err = client.Publish(ctx, env, entry)
	} else if !d.Get("published").(bool) && entry.Sys.PublishedAt != "" {
		err = client.Unpublish(ctx, env, entry)
	}

	if d.Get("archived").(bool) && entry.Sys.ArchivedAt == "" {
		err = client.Archive(ctx, env, entry)
	} else if !d.Get("archived").(bool) && entry.Sys.ArchivedAt != "" {
		err = client.Unarchive(ctx, env, entry)
	}

	return err
}

func resourceReadEntry(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulEntryClient) (diags diag.Diagnostics) {
	entryID := d.Id()

	entry, err := client.Get(ctx, env, entryID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setEntryProperties(d, entry)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func resourceDeleteEntry(ctx context.Context, d *schema.ResourceData, env *contentful.Environment, client ContentfulEntryClient) (diags diag.Diagnostics) {
	entryID := d.Id()

	_, err := client.Get(ctx, env, entryID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Delete(ctx, env, entryID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func setEntryProperties(d *schema.ResourceData, entry *contentful.Entry) (err error) {
	if err = d.Set("space_id", entry.Sys.Space.Sys.ID); err != nil {
		return err
	}

	if err = d.Set("version", entry.Sys.Version); err != nil {
		return err
	}

	if err = d.Set("contenttype_id", entry.Sys.ContentType.Sys.ID); err != nil {
		return err
	}

	return err
}
