package contentful

import (
	"context"

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
							Type:     schema.TypeString,
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
		content := field["content"]

		// Check the type of content and convert to integer if it's an integer.
		var fieldValue interface{}
		switch content.(type) {
		case int:
			fieldValue = content.(int)
		case float64:
			// Handle float64 as well, in case content is a floating-point number.
			fieldValue = int(content.(float64))
		default:
			// If it's not an integer, assume it's a string.
			fieldValue = content.(string)
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
		fieldProperties[field["id"].(string)] = map[string]interface{}{}
		fieldProperties[field["id"].(string)].(map[string]interface{})[field["locale"].(string)] = field["content"].(string)
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
