// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cosmos

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/restorables"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func dataSourceCosmosDbRestorableDatabaseAccounts() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Read: dataSourceCosmosDbRestorableDatabaseAccountsRead,

		Timeouts: &pluginsdk.ResourceTimeout{
			Read: pluginsdk.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"location": commonschema.LocationWithoutForceNew(),

			"accounts": {
				Type:     pluginsdk.TypeList,
				Computed: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"id": {
							Type:     pluginsdk.TypeString,
							Computed: true,
						},

						"api_type": {
							Type:     pluginsdk.TypeString,
							Computed: true,
						},

						"creation_time": {
							Type:     pluginsdk.TypeString,
							Computed: true,
						},

						"deletion_time": {
							Type:     pluginsdk.TypeString,
							Computed: true,
						},

						"restorable_locations": {
							Type:     pluginsdk.TypeList,
							Computed: true,
							Elem: &pluginsdk.Resource{
								Schema: map[string]*pluginsdk.Schema{
									"creation_time": {
										Type:     pluginsdk.TypeString,
										Computed: true,
									},

									"deletion_time": {
										Type:     pluginsdk.TypeString,
										Computed: true,
									},

									"location": {
										Type:     pluginsdk.TypeString,
										Computed: true,
									},

									"regional_database_account_instance_id": {
										Type:     pluginsdk.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceCosmosDbRestorableDatabaseAccountsRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.RestorablesClient
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id := restorables.NewRestorableDatabaseAccountID(subscriptionId, d.Get("location").(string), "read")

	name := d.Get("name").(string)
	location := d.Get("location").(string)

	locationId := restorables.NewLocationID(subscriptionId, location)

	resp, err := client.RestorableDatabaseAccountsListByLocation(ctx, locationId)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			return fmt.Errorf("%s was not found", id)
		}
		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	d.Set("location", location)

	if props := resp.Model.Value; props != nil {
		if err := d.Set("accounts", flattenCosmosDbRestorableDatabaseAccounts(props, name)); err != nil {
			return fmt.Errorf("flattening `accounts`: %+v", err)
		}
	}

	d.SetId(id.ID())

	return nil
}

func flattenCosmosDbRestorableDatabaseAccounts(input *[]restorables.RestorableDatabaseAccountGetResult, accountName string) []interface{} {
	result := make([]interface{}, 0)

	if len(*input) == 0 {
		return result
	}

	for _, item := range *input {
		if props := item.Properties; props != nil && props.AccountName != nil && *props.AccountName == accountName {
			var id, creationTime, deletionTime string
			var apiType restorables.ApiType

			if item.Id != nil {
				id = *item.Id
			}

			if props.ApiType != nil {
				apiType = *props.ApiType
			}

			if props.CreationTime != nil {
				creationTime = *props.CreationTime
			}

			if props.DeletionTime != nil {
				deletionTime = *props.DeletionTime
			}

			result = append(result, map[string]interface{}{
				"id":                   id,
				"api_type":             string(apiType),
				"creation_time":        creationTime,
				"deletion_time":        deletionTime,
				"restorable_locations": flattenCosmosDbRestorableDatabaseAccountsRestorableLocations(props.RestorableLocations),
			})
		}
	}

	return result
}

func flattenCosmosDbRestorableDatabaseAccountsRestorableLocations(input *[]restorables.RestorableLocationResource) []interface{} {
	result := make([]interface{}, 0)

	if len(*input) == 0 {
		return result
	}

	for _, item := range *input {
		var location, regionalDatabaseAccountInstanceId, creationTime, deletionTime string

		if item.LocationName != nil {
			location = *item.LocationName
		}

		if item.RegionalDatabaseAccountInstanceId != nil {
			regionalDatabaseAccountInstanceId = *item.RegionalDatabaseAccountInstanceId
		}

		if item.CreationTime != nil {
			creationTime = *item.CreationTime
		}

		if item.DeletionTime != nil {
			deletionTime = *item.DeletionTime
		}

		result = append(result, map[string]interface{}{
			"creation_time":                         creationTime,
			"deletion_time":                         deletionTime,
			"location":                              location,
			"regional_database_account_instance_id": regionalDatabaseAccountInstanceId,
		})
	}

	return result
}
