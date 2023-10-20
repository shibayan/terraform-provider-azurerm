// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cosmos

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/cosmosdb"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func resourceCosmosDbSQLStoredProcedure() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceCosmosDbSQLStoredProcedureCreate,
		Read:   resourceCosmosDbSQLStoredProcedureRead,
		Update: resourceCosmosDbSQLStoredProcedureUpdate,
		Delete: resourceCosmosDbSQLStoredProcedureDelete,

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := cosmosdb.ParseStoredProcedureID(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"resource_group_name": commonschema.ResourceGroupName(),

			"account_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"body": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"container_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},

			"database_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosEntityName,
			},
		},
	}
}

func resourceCosmosDbSQLStoredProcedureCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	storedProcBody := d.Get("body").(string)
	id := cosmosdb.NewStoredProcedureID(subscriptionId, d.Get("resource_group_name").(string), d.Get("account_name").(string), d.Get("database_name").(string), d.Get("container_name").(string), d.Get("name").(string))

	existing, err := client.SqlResourcesGetSqlStoredProcedure(ctx, id)
	if err != nil {
		if !response.WasNotFound(existing.HttpResponse) {
			return fmt.Errorf("checking for presence of %s: %+v", id, err)
		}
	} else {
		if existing.Model.Id == nil && *existing.Model.Id == "" {
			return fmt.Errorf("generating import ID for %s", id)
		}

		return tf.ImportAsExistsError("azurerm_cosmosdb_sql_stored_procedure", *existing.Model.Id)
	}

	storedProcParams := cosmosdb.SqlStoredProcedureCreateUpdateParameters{
		Properties: cosmosdb.SqlStoredProcedureCreateUpdateProperties{
			Resource: cosmosdb.SqlStoredProcedureResource{
				Id:   id.StoredProcedureName,
				Body: &storedProcBody,
			},
			Options: &cosmosdb.CreateUpdateOptions{},
		},
	}

	future, err := client.SqlResourcesCreateUpdateSqlStoredProcedure(ctx, id, storedProcParams)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", id, err)
	}

	if err = future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for creation of %s: %+v", id, err)
	}

	d.SetId(id.ID())

	return resourceCosmosDbSQLStoredProcedureRead(d, meta)
}

func resourceCosmosDbSQLStoredProcedureUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseStoredProcedureID(d.Id())
	if err != nil {
		return err
	}

	containerName := id.ContainerName
	databaseName := id.SqlDatabaseName
	accountName := id.DatabaseAccountName
	name := id.StoredProcedureName

	storedProcParams := cosmosdb.SqlStoredProcedureCreateUpdateParameters{
		Properties: cosmosdb.SqlStoredProcedureCreateUpdateProperties{
			Resource: cosmosdb.SqlStoredProcedureResource{
				Id:   name,
				Body: pointer.FromString(d.Get("body").(string)),
			},
			Options: &cosmosdb.CreateUpdateOptions{},
		},
	}

	future, err := client.SqlResourcesCreateUpdateSqlStoredProcedure(ctx, *id, storedProcParams)
	if err != nil {
		return fmt.Errorf("updating SQL Stored Procedure %q (Container %q / Database %q / Account %q): %+v", name, containerName, databaseName, accountName, err)
	}

	if err = future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for update of SQL Stored Procedure %q (Container %q / Database %q / Account %q): %+v", name, containerName, databaseName, accountName, err)
	}

	return resourceCosmosDbSQLStoredProcedureRead(d, meta)
}

func resourceCosmosDbSQLStoredProcedureRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseStoredProcedureID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.SqlResourcesGetSqlStoredProcedure(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[INFO] SQL Stored Procedure %q (Container %q / Database %q / Account %q) was not found - removing from state", id.StoredProcedureName, id.ContainerName, id.SqlDatabaseName, id.DatabaseAccountName)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving SQL Stored Procedure %q (Container %q / Database %q / Account %q): %+v", id.StoredProcedureName, id.ContainerName, id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	d.Set("resource_group_name", id.ResourceGroupName)
	d.Set("account_name", id.DatabaseAccountName)
	d.Set("database_name", id.SqlDatabaseName)
	d.Set("container_name", id.ContainerName)
	d.Set("name", id.StoredProcedureName)

	if props := resp.Model.Properties; props != nil {
		if props.Resource != nil {
			d.Set("body", props.Resource.Body)
		}
	}

	return nil
}

func resourceCosmosDbSQLStoredProcedureDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseStoredProcedureID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.SqlResourcesDeleteSqlStoredProcedure(ctx, *id)
	if err != nil {
		if !response.WasNotFound(future.HttpResponse) {
			return fmt.Errorf("deleting SQL Stored Procedure %q (Container %q / Database %q / Account %q): %+v", id.StoredProcedureName, id.ContainerName, id.SqlDatabaseName, id.DatabaseAccountName, err)
		}
	}

	err = future.Poller.PollUntilDone()
	if err != nil {
		return fmt.Errorf("waiting for deletion of SQL Stored Procedure %q (Container %q / Database %q / Account %q): %+v", id.StoredProcedureName, id.ContainerName, id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	return nil
}
