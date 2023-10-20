// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cosmos

import (
	"fmt"
	"log"
	"time"

	// nolint: staticcheck
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
)

func resourceCosmosDbSQLRoleAssignment() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceCosmosDbSQLRoleAssignmentCreate,
		Read:   resourceCosmosDbSQLRoleAssignmentRead,
		Update: resourceCosmosDbSQLRoleAssignmentUpdate,
		Delete: resourceCosmosDbSQLRoleAssignmentDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := rbacs.ParseAccountID(id)
			return err
		}),

		Schema: map[string]*pluginsdk.Schema{
			"name": {
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

			"principal_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"scope": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"role_definition_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ValidateFunc: validate.SqlRoleDefinitionID,
			},
		},
	}
}

func resourceCosmosDbSQLRoleAssignmentCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	name := d.Get("name").(string)
	if name == "" {
		uuid, err := uuid.GenerateUUID()
		if err != nil {
			return fmt.Errorf("generating UUID for Cosmos DB SQL Role Assignment: %+v", err)
		}

		name = uuid
	}

	resourceGroup := d.Get("resource_group_name").(string)
	accountName := d.Get("account_name").(string)

	id := rbacs.NewAccountID(subscriptionId, resourceGroup, accountName, name)

	locks.ByName(id.DatabaseAccountName, CosmosDbAccountResourceName)
	defer locks.UnlockByName(id.DatabaseAccountName, CosmosDbAccountResourceName)

	existing, err := client.SqlResourcesGetSqlRoleAssignment(ctx, id)
	if err != nil {
		if !response.WasNotFound(existing.HttpResponse) {
			return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
		}
	}
	if !response.WasNotFound(existing.HttpResponse) {
		return tf.ImportAsExistsError("azurerm_cosmosdb_sql_role_assignment", id.ID())
	}

	parameters := rbacs.SqlRoleAssignmentCreateUpdateParameters{
		Properties: &rbacs.SqlRoleAssignmentResource{
			PrincipalId:      pointer.FromString(d.Get("principal_id").(string)),
			RoleDefinitionId: pointer.FromString(d.Get("role_definition_id").(string)),
			Scope:            pointer.FromString(d.Get("scope").(string)),
		},
	}

	future, err := client.SqlResourcesCreateUpdateSqlRoleAssignment(ctx, id, parameters)
	if err != nil {
		return fmt.Errorf("creating/updating %s: %+v", id, err)
	}

	if err = future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for the completion of the creating/updating of %s: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceCosmosDbSQLRoleAssignmentRead(d, meta)
}

func resourceCosmosDbSQLRoleAssignmentRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := rbacs.ParseAccountID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.SqlResourcesGetSqlRoleAssignment(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[INFO] %s was not found - removing from state", id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	d.Set("name", id.RoleAssignmentId)
	d.Set("resource_group_name", id.ResourceGroupName)
	d.Set("account_name", id.DatabaseAccountName)

	if props := resp.Model.Properties; props != nil {
		d.Set("principal_id", props.PrincipalId)
		d.Set("role_definition_id", props.RoleDefinitionId)
		d.Set("scope", props.Scope)
	}

	return nil
}

func resourceCosmosDbSQLRoleAssignmentUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := rbacs.ParseAccountID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.DatabaseAccountName, CosmosDbAccountResourceName)
	defer locks.UnlockByName(id.DatabaseAccountName, CosmosDbAccountResourceName)

	parameters := rbacs.SqlRoleAssignmentCreateUpdateParameters{
		Properties: &rbacs.SqlRoleAssignmentResource{
			PrincipalId:      pointer.FromString(d.Get("principal_id").(string)),
			RoleDefinitionId: pointer.FromString(d.Get("role_definition_id").(string)),
			Scope:            pointer.FromString(d.Get("scope").(string)),
		},
	}

	future, err := client.SqlResourcesCreateUpdateSqlRoleAssignment(ctx, *id, parameters)
	if err != nil {
		return fmt.Errorf("updating %s: %+v", id, err)
	}

	if err = future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for the completion of the updating of %s: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceCosmosDbSQLRoleAssignmentRead(d, meta)
}

func resourceCosmosDbSQLRoleAssignmentDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Cosmos.RbacsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := rbacs.ParseAccountID(d.Id())
	if err != nil {
		return err
	}

	locks.ByName(id.DatabaseAccountName, CosmosDbAccountResourceName)
	defer locks.UnlockByName(id.DatabaseAccountName, CosmosDbAccountResourceName)

	future, err := client.SqlResourcesDeleteSqlRoleAssignment(ctx, *id)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(); err != nil {
		return fmt.Errorf("waiting for the completion of the deleting of %s: %+v", id, err)
	}

	return nil
}
