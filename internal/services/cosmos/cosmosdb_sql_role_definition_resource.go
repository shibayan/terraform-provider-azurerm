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
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/rbacs"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/locks"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/cosmos/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceCosmosDbSQLRoleDefinition() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceCosmosDbSQLRoleDefinitionCreate,
		Read:   resourceCosmosDbSQLRoleDefinitionRead,
		Update: resourceCosmosDbSQLRoleDefinitionUpdate,
		Delete: resourceCosmosDbSQLRoleDefinitionDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := rbacs.ParseSqlRoleDefinitionID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{
			"role_definition_id": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"resource_group_name": commonschema.ResourceGroupName(),

			"account_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.CosmosAccountName,
			},

			"type": {
				Type:         pluginsdk.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      string(rbacs.RoleDefinitionTypeCustomRole),
				ValidateFunc: validation.StringInSlice(rbacs.PossibleValuesForRoleDefinitionType(), false),
			},

			"assignable_scopes": {
				Type:     pluginsdk.TypeSet,
				Required: true,
				Elem: &pluginsdk.Schema{
					Type:         pluginsdk.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},

			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"permissions": {
				Type:     pluginsdk.TypeSet,
				Required: true,
				Elem: &pluginsdk.Resource{
					Schema: map[string]*pluginsdk.Schema{
						"data_actions": {
							Type:     pluginsdk.TypeSet,
							Required: true,
							Elem: &pluginsdk.Schema{
								Type:         pluginsdk.TypeString,
								ValidateFunc: validation.StringIsNotEmpty,
							},
						},
					},
				},
			},
		},
	}
}

func resourceCosmosDbSQLRoleDefinitionCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	roleDefinitionId := d.Get("role_definition_id").(string)
	if roleDefinitionId == "" {
		uuid, err := uuid.GenerateUUID()
		if err != nil {
			return fmt.Errorf("generating UUID for Cosmos DB SQL Role Definition: %+v", err)
		}

		roleDefinitionId = uuid
	}

	resourceGroup := d.Get("resource_group_name").(string)
	accountName := d.Get("account_name").(string)

	id := rbacs.NewSqlRoleDefinitionID(subscriptionId, resourceGroup, accountName, roleDefinitionId)

	locks.ByName(id.DatabaseAccountName, CosmosDbAccountResourceName)
	defer locks.UnlockByName(id.DatabaseAccountName, CosmosDbAccountResourceName)

	existing, err := client.SqlResourcesGetSqlRoleDefinition(ctx, id)
	if err != nil {
		if !response.WasNotFound(existing.HttpResponse) {
			return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
		}
	}
	if !response.WasNotFound(existing.HttpResponse) {
		return tf.ImportAsExistsError("azurerm_cosmosdb_sql_role_definition", id.ID())
	}

	parameters := rbacs.SqlRoleDefinitionCreateUpdateParameters{
		Properties: &rbacs.SqlRoleDefinitionResource{
			RoleName:         pointer.FromString(d.Get("name").(string)),
			AssignableScopes: utils.ExpandStringSlice(d.Get("assignable_scopes").(*pluginsdk.Set).List()),
			Permissions:      expandSqlRoleDefinitionPermissions(d.Get("permissions").(*pluginsdk.Set).List()),
			Type:             pointer.To(rbacs.RoleDefinitionType(d.Get("type").(string))),
		},
	}

	future, err := client.SqlResourcesCreateUpdateSqlRoleDefinition(ctx, id, parameters)
	if err != nil {
		return fmt.Errorf("creating %s: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for creation of %s: %+v", id, err)
	}

	d.SetId(id.ID())

	return resourceCosmosDbSQLRoleDefinitionRead(d, meta)
}

func resourceCosmosDbSQLRoleDefinitionRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := rbacs.ParseSqlRoleDefinitionID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.SqlResourcesGetSqlRoleDefinition(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[DEBUG] %s was not found - removing from state", *id)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	d.Set("role_definition_id", id.RoleDefinitionId)
	d.Set("resource_group_name", id.ResourceGroupName)
	d.Set("account_name", id.DatabaseAccountName)

	if props := resp.Model.Properties; props != nil {
		d.Set("assignable_scopes", utils.FlattenStringSlice(props.AssignableScopes))
		d.Set("name", props.RoleName)
		d.Set("type", props.Type)

		if err := d.Set("permissions", flattenSqlRoleDefinitionPermissions(props.Permissions)); err != nil {
			return fmt.Errorf("setting `permissions`: %+v", err)
		}
	}

	return nil
}

func resourceCosmosDbSQLRoleDefinitionUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := rbacs.ParseSqlRoleDefinitionID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.DatabaseAccountName, CosmosDbAccountResourceName)
	defer locks.UnlockByName(id.DatabaseAccountName, CosmosDbAccountResourceName)

	parameters := rbacs.SqlRoleDefinitionCreateUpdateParameters{
		Properties: &rbacs.SqlRoleDefinitionResource{
			RoleName:         pointer.FromString(d.Get("name").(string)),
			AssignableScopes: utils.ExpandStringSlice(d.Get("assignable_scopes").(*pluginsdk.Set).List()),
			Permissions:      expandSqlRoleDefinitionPermissions(d.Get("permissions").(*pluginsdk.Set).List()),
			Type:             pointer.To(rbacs.RoleDefinitionType(d.Get("type").(string))),
		},
	}

	future, err := client.SqlResourcesCreateUpdateSqlRoleDefinition(ctx, *id, parameters)
	if err != nil {
		return fmt.Errorf("updating %s: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for update of %s: %+v", id, err)
	}

	d.SetId(id.ID())

	return resourceCosmosDbSQLRoleDefinitionRead(d, meta)
}

func resourceCosmosDbSQLRoleDefinitionDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := rbacs.ParseSqlRoleDefinitionID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.DatabaseAccountName, CosmosDbAccountResourceName)
	defer locks.UnlockByName(id.DatabaseAccountName, CosmosDbAccountResourceName)

	future, err := client.SqlResourcesDeleteSqlRoleDefinition(ctx, *id)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for deletion of %s: %+v", *id, err)
	}

	return nil
}

func expandSqlRoleDefinitionPermissions(input []interface{}) *[]rbacs.Permission {
	results := make([]rbacs.Permission, 0)

	for _, item := range input {
		v := item.(map[string]interface{})

		results = append(results, rbacs.Permission{
			DataActions: utils.ExpandStringSlice(v["data_actions"].(*pluginsdk.Set).List()),
		})
	}

	return &results
}

func flattenSqlRoleDefinitionPermissions(input *[]rbacs.Permission) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	for _, item := range *input {
		results = append(results, map[string]interface{}{
			"data_actions": utils.FlattenStringSlice(item.DataActions),
		})
	}

	return results
}
