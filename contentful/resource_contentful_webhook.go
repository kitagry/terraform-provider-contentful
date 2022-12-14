package contentful

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/kitagry/contentful-go"
)

func resourceContentfulWebhook() *schema.Resource {
	return &schema.Resource{
		CreateContext: wrapWebhook(resourceCreateWebhook),
		ReadContext:   wrapWebhook(resourceReadWebhook),
		UpdateContext: wrapWebhook(resourceUpdateWebhook),
		DeleteContext: wrapWebhook(resourceDeleteWebhook),

		Schema: map[string]*schema.Schema{
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"space_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			// Webhook specific props
			"url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"http_basic_auth_username": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"http_basic_auth_password": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"headers": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"topics": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems: 1,
				Required: true,
			},
		},
	}
}

func wrapWebhook(f func(ctx context.Context, d *schema.ResourceData, client ContentfulWebhookClient) diag.Diagnostics) func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) (diags diag.Diagnostics) {
		client := m.(*contentful.Client)
		return f(ctx, d, client.Webhooks)
	}
}

func resourceCreateWebhook(ctx context.Context, d *schema.ResourceData, client ContentfulWebhookClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)

	webhook := &contentful.Webhook{
		Name:              d.Get("name").(string),
		URL:               d.Get("url").(string),
		Topics:            transformTopicsToContentfulFormat(d.Get("topics").([]interface{})),
		Headers:           transformHeadersToContentfulFormat(d.Get("headers")),
		HTTPBasicUsername: d.Get("http_basic_auth_username").(string),
		HTTPBasicPassword: d.Get("http_basic_auth_password").(string),
	}

	err := client.Upsert(ctx, spaceID, webhook)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setWebhookProperties(d, webhook)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(webhook.Sys.ID)

	return nil
}

func resourceUpdateWebhook(ctx context.Context, d *schema.ResourceData, client ContentfulWebhookClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	webhookID := d.Id()
	defer func() {
		if diags.HasError() {
			d.Partial(true)
		}
	}()

	webhook, err := client.Get(ctx, spaceID, webhookID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	webhook.Name = d.Get("name").(string)
	webhook.URL = d.Get("url").(string)
	webhook.Topics = transformTopicsToContentfulFormat(d.Get("topics").([]interface{}))
	webhook.Headers = transformHeadersToContentfulFormat(d.Get("headers"))
	webhook.HTTPBasicUsername = d.Get("http_basic_auth_username").(string)
	webhook.HTTPBasicPassword = d.Get("http_basic_auth_password").(string)

	err = client.Upsert(ctx, spaceID, webhook)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setWebhookProperties(d, webhook)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	d.SetId(webhook.Sys.ID)

	return nil
}

func resourceReadWebhook(ctx context.Context, d *schema.ResourceData, client ContentfulWebhookClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	webhookID := d.Id()

	webhook, err := client.Get(ctx, spaceID, webhookID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}

	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = setWebhookProperties(d, webhook)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}
	return
}

func resourceDeleteWebhook(ctx context.Context, d *schema.ResourceData, client ContentfulWebhookClient) (diags diag.Diagnostics) {
	spaceID := d.Get("space_id").(string)
	webhookID := d.Id()

	webhook, err := client.Get(ctx, spaceID, webhookID)
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	err = client.Delete(ctx, spaceID, webhook)
	if _, ok := err.(contentful.NotFoundError); ok {
		return nil
	}
	if err != nil {
		diags = append(diags, contentfulErrorToDiagnostic(err)...)
		return
	}

	return
}

func setWebhookProperties(d *schema.ResourceData, webhook *contentful.Webhook) (err error) {
	headers := make(map[string]string)
	for _, entry := range webhook.Headers {
		headers[entry.Key] = entry.Value
	}

	err = d.Set("headers", headers)
	if err != nil {
		return err
	}

	err = d.Set("space_id", webhook.Sys.Space.Sys.ID)
	if err != nil {
		return err
	}

	err = d.Set("version", webhook.Sys.Version)
	if err != nil {
		return err
	}

	err = d.Set("name", webhook.Name)
	if err != nil {
		return err
	}

	err = d.Set("url", webhook.URL)
	if err != nil {
		return err
	}

	err = d.Set("http_basic_auth_username", webhook.HTTPBasicUsername)
	if err != nil {
		return err
	}

	err = d.Set("topics", webhook.Topics)
	if err != nil {
		return err
	}

	return nil
}

func transformHeadersToContentfulFormat(headersTerraform interface{}) []*contentful.WebhookHeader {
	var headers []*contentful.WebhookHeader

	for k, v := range headersTerraform.(map[string]interface{}) {
		headers = append(headers, &contentful.WebhookHeader{
			Key:   k,
			Value: v.(string),
		})
	}

	return headers
}

func transformTopicsToContentfulFormat(topicsTerraform []interface{}) []string {
	var topics []string

	for _, v := range topicsTerraform {
		topics = append(topics, v.(string))
	}

	return topics
}
