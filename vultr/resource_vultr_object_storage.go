package vultr

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVultrObjectStorage() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVultrObjectStorageCreate,
		ReadContext:   resourceVultrObjectStorageRead,
		UpdateContext: resourceVultrObjectStorageUpdate,
		DeleteContext: resourceVultrObjectStorageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Required: true,
			},
			"label": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"date_created": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"location": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"s3_hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"s3_access_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"s3_secret_key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func resourceVultrObjectStorageCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Client).govultrClient()

	objStoreCluster := d.Get("cluster_id").(int)
	label := d.Get("label").(string)

	obj, err := client.ObjectStorage.Create(ctx, objStoreCluster, label)
	if err != nil {
		return diag.Errorf("error creating object storage: %v", err)
	}

	d.SetId(obj.ID)

	if _, err = waitForObjAvailable(ctx, d, "active", []string{"pending"}, "status", meta); err != nil {
		return diag.Errorf("error while waiting for Object Storage %s to be in a active state : %s", d.Id(), err)
	}

	return resourceVultrObjectStorageRead(ctx, d, meta)
}

func resourceVultrObjectStorageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Client).govultrClient()

	obj, err := client.ObjectStorage.Get(ctx, d.Id())
	if err != nil {
		return diag.Errorf("error getting object storage account: %v", err)
	}

	d.Set("date_created", obj.DateCreated)
	d.Set("cluster_id", obj.ObjectStoreClusterID)
	d.Set("label", obj.Label)
	d.Set("location", obj.Location)
	d.Set("region", obj.Region)
	d.Set("status", obj.Status)
	d.Set("s3_hostname", obj.S3Hostname)
	d.Set("s3_access_key", obj.S3AccessKey)
	d.Set("s3_secret_key", obj.S3SecretKey)

	return nil
}

func resourceVultrObjectStorageUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Client).govultrClient()

	label := d.Get("label").(string)

	if err := client.ObjectStorage.Update(ctx, d.Id(), label); err != nil {
		return diag.Errorf("error updating object storage %s label : %v", d.Id(), err)
	}

	return resourceVultrObjectStorageRead(ctx, d, meta)
}

func resourceVultrObjectStorageDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*Client).govultrClient()

	log.Printf("[INFO] Deleting Object storage subscription %s", d.Id())

	if err := client.ObjectStorage.Delete(ctx, d.Id()); err != nil {
		return diag.Errorf("error deleting object storage subscription %s : %v", d.Id(), err)
	}

	return nil
}

func waitForObjAvailable(ctx context.Context, d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}) (interface{}, error) {
	log.Printf(
		"[INFO] Waiting for Object Storage (%s) to have %s of %s",
		d.Id(), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:        pending,
		Target:         []string{target},
		Refresh:        newServerObjRefresh(ctx, d, meta, attribute),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	return stateConf.WaitForStateContext(ctx)
}

func newServerObjRefresh(ctx context.Context, d *schema.ResourceData, meta interface{}, attr string) resource.StateRefreshFunc {
	client := meta.(*Client).govultrClient()
	return func() (interface{}, string, error) {
		log.Printf("[INFO] Creating Object Storage")

		obj, err := client.ObjectStorage.Get(ctx, d.Id())
		if err != nil {
			return nil, "", fmt.Errorf("error retrieving Object Store %s : %s", d.Id(), err)
		}

		log.Print(obj)
		if attr == "status" {
			log.Printf("[INFO] The Object Storage Status is %s", obj.Status)
			return obj, obj.Status, nil
		}
		return nil, "", nil
	}
}
