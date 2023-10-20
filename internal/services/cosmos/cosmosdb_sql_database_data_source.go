// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cosmos

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/cosmosdb"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/common"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func dataSourceCosmosDbSQLDatabase() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Read: dataSourceCosmosDbSQLDatabaseRead,

		Timeouts: &pluginsdk.ResourceTimeout{
			Read: pluginsdk.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validate.CosmosEntityName,
			},

			"resource_group_name": commonschema.ResourceGroupName(),

			"account_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"throughput": {
				Type:     pluginsdk.TypeInt,
				Computed: true,
			},

			"autoscale_settings": {
				Type:     pluginsdk.TypeList,
				Computed: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"max_throughput": {
							Type:     pluginsdk.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceCosmosDbSQLDatabaseRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.CosmosDBClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id := cosmosdb.NewSqlDatabaseID(subscriptionId, d.Get("resource_group_name").(string), d.Get("account_name").(string), d.Get("name").(string))

	resp, err := client.SqlResourcesGetSqlDatabase(ctx, id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			return fmt.Errorf("%s was not found", id)
		}

		return fmt.Errorf("reading Cosmos SQL Database %q (Account: %q): %+v", id.SqlDatabaseName, id.DatabaseAccountName, err)
	}

	d.SetId(id.ID())
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
		throughputResp, err := client.SqlResourcesGetSqlDatabaseThroughput(ctx, id)
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
