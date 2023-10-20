// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cosmos

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/cosmosdb"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/common"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/migration"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func resourceCosmosDbSQLDatabase() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceCosmosDbSQLDatabaseCreate,
		Read:   resourceCosmosDbSQLDatabaseRead,
		Update: resourceCosmosDbSQLDatabaseUpdate,
		Delete: resourceCosmosDbSQLDatabaseDelete,

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := cosmosdb.ParseSqlDatabaseID(id)
			return err
		}),

		SchemaVersion: 1,
		StateUpgraders: pluginsdk.StateUpgrades(map[int]pluginsdk.StateUpgrade{
			0: migration.SqlDatabaseV0ToV1{},
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
				ValidateFunc: validate.CosmosEntityName,
			},

			"resource_group_name": commonschema.ResourceGroupName(),

			"account_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"throughput": {
				Type:         pluginsdk.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validate.CosmosThroughput,
			},

			"autoscale_settings": common.DatabaseAutoscaleSettingsSchema(),
		},
	}
}

func resourceCosmosDbSQLDatabaseCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id := cosmosdb.NewSqlDatabaseID(subscriptionId, d.Get("resource_group_name").(string), d.Get("account_name").(string), d.Get("name").(string))

	existing, err := client.SqlResourcesGetSqlDatabase(ctx, id)
	if err != nil {
		if !response.WasNotFound(existing.HttpResponse) {
			return fmt.Errorf("checking for presence of %s: %+v", id, err)
		}
	} else {
		if existing.Model.Id == nil && *existing.Model.Id == "" {
			return fmt.Errorf("generating import ID for %s", id)
		}

		return tf.ImportAsExistsError("azurerm_cosmosdb_sql_database", *existing.Model.Id)
	}

	db := cosmosdb.SqlDatabaseCreateUpdateParameters{
		Properties: cosmosdb.SqlDatabaseCreateUpdateProperties{
			Resource: cosmosdb.SqlDatabaseResource{
				Id: id.SqlDatabaseName,
			},
			Options: &cosmosdb.CreateUpdateOptions{},
		},
	}

	if throughput, hasThroughput := d.GetOk("throughput"); hasThroughput {
		if throughput != 0 {
			db.Properties.Options.Throughput = common.ConvertThroughputFromResourceData(throughput)
		}
	}

	if _, hasAutoscaleSettings := d.GetOk("autoscale_settings"); hasAutoscaleSettings {
		db.Properties.Options.AutoScaleSettings = common.ExpandCosmosDbAutoscaleSettings(d)
	}

	future, err := client.SqlResourcesCreateUpdateSqlDatabase(ctx, id, db)
	if err != nil {
		return fmt.Errorf("issuing create/update request for %s: %+v", id, err)
	}

	if err = future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting on create/update future for %s: %+v", id, err)
	}

	d.SetId(id.ID())

	return resourceCosmosDbSQLDatabaseRead(d, meta)
}

func resourceCosmosDbSQLDatabaseUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseSqlDatabaseID(d.Id())
	if err != nil {
		return err
	}

	err = common.CheckForChangeFromAutoscaleAndManualThroughput(d)
	if err != nil {
		return fmt.Errorf("updating Cosmos SQL Database %q (Account: %q) - %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	db := cosmosdb.SqlDatabaseCreateUpdateParameters{
		Properties: cosmosdb.SqlDatabaseCreateUpdateProperties{
			Resource: cosmosdb.SqlDatabaseResource{
				Id: id.SqlDatabaseName,
			},
			Options: &cosmosdb.CreateUpdateOptions{},
		},
	}

	future, err := client.SqlResourcesCreateUpdateSqlDatabase(ctx, *id, db)
	if err != nil {
		return fmt.Errorf("issuing create/update request for Cosmos SQL Database %q (Account: %q): %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	if err = future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting on create/update future for Cosmos SQL Database %q (Account: %q): %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	if common.HasThroughputChange(d) {
		throughputParameters := common.ExpandCosmosDBThroughputSettingsUpdateParameters(d)
		throughputFuture, err := client.SqlResourcesUpdateSqlDatabaseThroughput(ctx, *id, *throughputParameters)
		if err != nil {
			if response.WasNotFound(throughputFuture.HttpResponse) {
				return fmt.Errorf("setting Throughput for Cosmos SQL Database %q (Account: %q) %+v - "+
					"If the collection has not been created with an initial throughput, you cannot configure it later", id.SqlDatabaseName, id.DatabaseAccountName, err)
			}
		}

		if err = throughputFuture.Poller.PollUntilDone(); err != nil {
			return fmt.Errorf("waiting on ThroughputUpdate future for Cosmos SQL Database %q (Account: %q): %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
		}
	}

	return resourceCosmosDbSQLDatabaseRead(d, meta)
}

func resourceCosmosDbSQLDatabaseRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseSqlDatabaseID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.SqlResourcesGetSqlDatabase(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[INFO] Error reading Cosmos SQL Database %q (Account: %q) - removing from state", id.SqlDatabaseName, id.DatabaseAccountName)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("reading Cosmos SQL Database %q (Account: %q): %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	d.Set("resource_group_name", id.ResourceGroupName)
	d.Set("account_name", id.DatabaseAccountName)
	if props := resp.Model.Properties; props != nil {
		if res := props.Resource; res != nil {
			d.Set("name", res.Id)
		}
	}

	accountId := cosmosdb.NewDatabaseAccountID(subscriptionId, id.ResourceGroupName, id.DatabaseAccountName)

	accResp, err := client.DatabaseAccountsGet(ctx, accountId)
	if err != nil {
		return fmt.Errorf("reading CosmosDB Account %q (Resource Group %q): %+v", id.DatabaseAccountName, id.ResourceGroupName, err)
	}

	if accResp.Model.Id == nil || *accResp.Model.Id == "" {
		return fmt.Errorf("cosmosDB Account %q (Resource Group %q) ID is empty or nil", id.DatabaseAccountName, id.ResourceGroupName)
	}

	// if the cosmos account is serverless calling the get throughput api would yield an error
	if !common.IsServerlessCapacityMode(*accResp.Model) {
		throughputResp, err := client.SqlResourcesGetSqlDatabaseThroughput(ctx, *id)
		if err != nil {
			if !response.WasNotFound(throughputResp.HttpResponse) {
				return fmt.Errorf("reading Throughput on Cosmos SQL Database %q (Account: %q) ID: %v", id.SqlDatabaseName, id.DatabaseAccountName, err)
			} else {
				d.Set("throughput", nil)
				d.Set("autoscale_settings", nil)
			}
		} else {
			common.SetResourceDataThroughputFromResponse(*throughputResp.Model, d)
		}
	}

	return nil
}

func resourceCosmosDbSQLDatabaseDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := cosmosdb.ParseSqlDatabaseID(d.Id())
	if err != nil {
		return err
	}

	future, err := client.SqlResourcesDeleteSqlDatabase(ctx, *id)
	if err != nil {
		if !response.WasNotFound(future.HttpResponse) {
			return fmt.Errorf("deleting Cosmos SQL Database %q (Account: %q): %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
		}
	}

	err = future.Poller.PollUntilDone()
	if err != nil {
		return fmt.Errorf("waiting on delete future for Cosmos SQL Database %q (Account: %q): %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	return nil
}
