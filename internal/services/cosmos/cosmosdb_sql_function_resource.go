// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cosmos

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/cosmosdb"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func resourceCosmosDbSQLFunction() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceCosmosDbSQLFunctionCreateUpdate,
		Read:   resourceCosmosDbSQLFunctionRead,
		Update: resourceCosmosDbSQLFunctionCreateUpdate,
		Delete: resourceCosmosDbSQLFunctionDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := cosmosdb.ParseUserDefinedFunctionID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ForceNew: true,
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
		},
	}
}

func resourceCosmosDbSQLFunctionCreateUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	containerId, _ := cosmosdb.ParseContainerID(d.Get("container_id").(string))
	body := d.Get("body").(string)

	id := cosmosdb.NewUserDefinedFunctionID(subscriptionId, containerId.ResourceGroupName, containerId.DatabaseAccountName, containerId.SqlDatabaseName, containerId.ContainerName, name)

	if d.IsNewResource() {
		existing, err := client.SqlResourcesGetSqlUserDefinedFunction(ctx, id)
		if err != nil {
			if !response.WasNotFound(existing.HttpResponse) {
				return fmt.Errorf("checking for existing CosmosDb SqlFunction %q: %+v", id, err)
			}
		}
		if !response.WasNotFound(existing.HttpResponse) {
			return tf.ImportAsExistsError("azurerm_cosmosdb_sql_function", id.ID())
		}
	}

	createUpdateSqlUserDefinedFunctionParameters := cosmosdb.SqlUserDefinedFunctionCreateUpdateParameters{
		Properties: cosmosdb.SqlUserDefinedFunctionCreateUpdateProperties{
			Resource: cosmosdb.SqlUserDefinedFunctionResource{
				Id:   name,
				Body: &body,
			},
			Options: &cosmosdb.CreateUpdateOptions{},
		},
	}
	future, err := client.SqlResourcesCreateUpdateSqlUserDefinedFunction(ctx, id, createUpdateSqlUserDefinedFunctionParameters)
	if err != nil {
		return fmt.Errorf("creating/updating CosmosDb SqlFunction %q: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for creation/update of the CosmosDb SqlFunction %q: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceCosmosDbSQLFunctionRead(d, meta)
}

func resourceCosmosDbSQLFunctionRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseUserDefinedFunctionID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.SqlResourcesGetSqlUserDefinedFunction(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[INFO] CosmosDb SqlFunction %q does not exist - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving CosmosDb SqlFunction %q: %+v", id, err)
	}
	containerId := cosmosdb.NewContainerID(id.SubscriptionId, id.ResourceGroupName, id.DatabaseAccountName, id.SqlDatabaseName, id.ContainerName)
	d.Set("name", id.UserDefinedFunctionName)
	d.Set("container_id", containerId.ID())
	if props := resp.Model.Properties; props != nil {
		if props.Resource != nil {
			d.Set("body", props.Resource.Body)
		}
	}
	return nil
}

func resourceCosmosDbSQLFunctionDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseUserDefinedFunctionID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.SqlResourcesDeleteSqlUserDefinedFunction(ctx, *id)
	if err != nil {
		return fmt.Errorf("deleting CosmosDb SqlFunction %q: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for deletion of the CosmosDb SqlFunction %q: %+v", id, err)
	}
	return nil
}
