// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cosmos

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/cosmosdb"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func resourceCosmosDbSQLTrigger() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceCosmosDbSQLTriggerCreateUpdate,
		Read:   resourceCosmosDbSQLTriggerRead,
		Update: resourceCosmosDbSQLTriggerCreateUpdate,
		Delete: resourceCosmosDbSQLTriggerDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := cosmosdb.ParseTriggerID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},

			"container_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.SqlContainerID,
			},

			"body": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"operation": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(cosmosdb.PossibleValuesForTriggerOperation(), false),
			},

			"type": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(cosmosdb.PossibleValuesForTriggerType(), false),
			},
		},
	}
}

func resourceCosmosDbSQLTriggerCreateUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	containerId, _ := cosmosdb.ParseContainerID(d.Get("container_id").(string))
	body := d.Get("body").(string)
	triggerOperation := d.Get("operation").(string)
	triggerType := d.Get("type").(string)

	id := cosmosdb.NewTriggerID(subscriptionId, containerId.ResourceGroupName, containerId.DatabaseAccountName, containerId.SqlDatabaseName, containerId.ContainerName, name)

	if d.IsNewResource() {
		existing, err := client.SqlResourcesGetSqlTrigger(ctx, id)
		if err != nil {
			if !response.WasNotFound(existing.HttpResponse) {
				return fmt.Errorf("checking for existing CosmosDb SQLTrigger %q: %+v", id, err)
			}
		}
		if !response.WasNotFound(existing.HttpResponse) {
			return tf.ImportAsExistsError("azurerm_cosmosdb_sql_trigger", id.ID())
		}
	}

	createUpdateSqlTriggerParameters := cosmosdb.SqlTriggerCreateUpdateParameters{
		Properties: cosmosdb.SqlTriggerCreateUpdateProperties{
			Resource: cosmosdb.SqlTriggerResource{
				Id:               name,
				Body:             &body,
				TriggerType:      pointer.To(cosmosdb.TriggerType(triggerType)),
				TriggerOperation: pointer.To(cosmosdb.TriggerOperation(triggerOperation)),
			},
			Options: &cosmosdb.CreateUpdateOptions{},
		},
	}
	future, err := client.SqlResourcesCreateUpdateSqlTrigger(ctx, id, createUpdateSqlTriggerParameters)
	if err != nil {
		return fmt.Errorf("creating/updating CosmosDb SQLTrigger %q: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for creation/update of the CosmosDb SQLTrigger %q: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceCosmosDbSQLTriggerRead(d, meta)
}

func resourceCosmosDbSQLTriggerRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseTriggerID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.SqlResourcesGetSqlTrigger(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[INFO] CosmosDb SQLTrigger %q does not exist - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving CosmosDb SQLTrigger %q: %+v", id, err)
	}
	containerId := cosmosdb.NewContainerID(id.SubscriptionId, id.ResourceGroupName, id.DatabaseAccountName, id.SqlDatabaseName, id.ContainerName)
	d.Set("name", id.TriggerName)
	d.Set("container_id", containerId.ID())
	if props := resp.Model.Properties; props != nil {
		if props.Resource != nil {
			d.Set("body", props.Resource.Body)
			d.Set("operation", props.Resource.TriggerOperation)
			d.Set("type", props.Resource.TriggerType)
		}
	}
	return nil
}

func resourceCosmosDbSQLTriggerDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseTriggerID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.SqlResourcesDeleteSqlTrigger(ctx, *id)
	if err != nil {
		return fmt.Errorf("deleting CosmosDb SQLResourcesSQLTrigger %q: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for deletion of the CosmosDb SQLResourcesSQLTrigger %q: %+v", id, err)
	}
	return nil
}
