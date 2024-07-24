// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mssql_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/mssql/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

type MsSqlServerTransparentDataEncryptionResource struct{}

func TestAccMsSqlServerTransparentDataEncryption_keyVault(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_mssql_server_transparent_data_encryption", "test")
	r := MsSqlServerTransparentDataEncryptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.keyVault(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccMsSqlServerTransparentDataEncryption_managedHSM(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_mssql_server_transparent_data_encryption", "test")
	r := MsSqlServerTransparentDataEncryptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.managedHSM(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccMsSqlServerTransparentDataEncryption_autoRotate(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_mssql_server_transparent_data_encryption", "test")
	r := MsSqlServerTransparentDataEncryptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.autoRotate(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
		{
			Config: r.keyVault(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccMsSqlServerTransparentDataEncryption_systemManaged(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_mssql_server_transparent_data_encryption", "test")
	r := MsSqlServerTransparentDataEncryptionResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.systemManaged(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("key_vault_key_id").HasValue(""),
			),
		},
		data.ImportStep(),
	})
}

func TestAccMsSqlServerTransparentDataEncryption_update(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_mssql_server_transparent_data_encryption", "test")
	r := MsSqlServerTransparentDataEncryptionResource{}

	// Test going from systemManaged to keyVault and back
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.keyVault(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
		{
			Config: r.systemManaged(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("key_vault_key_id").HasValue(""),
			),
		},
		data.ImportStep(),
	})
}

func (MsSqlServerTransparentDataEncryptionResource) Exists(ctx context.Context, client *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := parse.EncryptionProtectorID(state.ID)
	if err != nil {
		return nil, err
	}

	resp, err := client.MSSQL.EncryptionProtectorClient.Get(ctx, id.ResourceGroup, id.ServerName)
	if err != nil {
		if utils.ResponseWasNotFound(resp.Response) {
			return nil, fmt.Errorf("Encryption protector for server %q (Resource Group %q) does not exist", id.ServerName, id.ResourceGroup)
		}

		return nil, fmt.Errorf("reading Encryption Protector for server %q (Resource Group %q): %v", id.ServerName, id.ResourceGroup, err)
	}

	return utils.Bool(resp.ID != nil), nil
}

func (r MsSqlServerTransparentDataEncryptionResource) baseKeyVault(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurerm_key_vault" "test" {
  name                        = "acctestsqlserver%[2]s"
  location                    = azurerm_resource_group.test.location
  resource_group_name         = azurerm_resource_group.test.name
  enabled_for_disk_encryption = true
  tenant_id                   = data.azurerm_client_config.current.tenant_id
  soft_delete_retention_days  = 7
  purge_protection_enabled    = true

  sku_name = "standard"

  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id

    key_permissions = [
      "Get", "List", "Create", "Delete", "Update", "Purge", "GetRotationPolicy", "SetRotationPolicy"
    ]
  }

  access_policy {
    tenant_id = azurerm_mssql_server.test.identity[0].tenant_id
    object_id = azurerm_mssql_server.test.identity[0].principal_id

    key_permissions = [
      "Get", "WrapKey", "UnwrapKey", "List", "Create", "GetRotationPolicy", "SetRotationPolicy"
    ]
  }
}

resource "azurerm_key_vault_key" "generated" {
  name         = "keyVault"
  key_vault_id = azurerm_key_vault.test.id
  key_type     = "RSA"
  key_size     = 2048

  key_opts = [
    "decrypt",
    "encrypt",
    "sign",
    "unwrapKey",
    "verify",
    "wrapKey",
  ]

  depends_on = [
    azurerm_key_vault.test,
  ]
}
`, r.server(data), data.RandomStringOfLength(5))
}

func (r MsSqlServerTransparentDataEncryptionResource) keyVault(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurerm_mssql_server_transparent_data_encryption" "test" {
  server_id        = azurerm_mssql_server.test.id
  key_vault_key_id = azurerm_key_vault_key.generated.id
}
`, r.baseKeyVault(data))
}

func (r MsSqlServerTransparentDataEncryptionResource) managedHSM(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurerm_mssql_server_transparent_data_encryption" "test" {
  server_id          = azurerm_mssql_server.test.id
  managed_hsm_key_id = azurerm_key_vault_managed_hardware_security_module_key.test.versioned_id
}
`, r.withManagedHSM(data))
}

func (r MsSqlServerTransparentDataEncryptionResource) autoRotate(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurerm_mssql_server_transparent_data_encryption" "test" {
  server_id             = azurerm_mssql_server.test.id
  key_vault_key_id      = azurerm_key_vault_key.generated.id
  auto_rotation_enabled = true
}
`, r.baseKeyVault(data))
}

func (r MsSqlServerTransparentDataEncryptionResource) systemManaged(data acceptance.TestData) string {
	return fmt.Sprintf(`
%s

resource "azurerm_mssql_server_transparent_data_encryption" "test" {
  server_id = azurerm_mssql_server.test.id
}
`, r.server(data))
}

func (MsSqlServerTransparentDataEncryptionResource) server(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-mssql-%[1]d"
  location = "%[2]s"
}

resource "azurerm_mssql_server" "test" {
  name                         = "acctestsqlserver%[1]d"
  resource_group_name          = azurerm_resource_group.test.name
  location                     = azurerm_resource_group.test.location
  version                      = "12.0"
  administrator_login          = "mradministrator"
  administrator_login_password = "thisIsDog11"

  identity {
    type = "SystemAssigned"
  }

  lifecycle {
    ignore_changes = [transparent_data_encryption_key_vault_key_id]
  }
}
`, data.RandomInteger, data.Locations.Primary)
}

func (r MsSqlServerTransparentDataEncryptionResource) withManagedHSM(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-mssql-%[2]s"
  location = "%[1]s"
}

resource "azurerm_key_vault" "test" {
  name                       = "acc%[2]s"
  location                   = azurerm_resource_group.test.location
  resource_group_name        = azurerm_resource_group.test.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  soft_delete_retention_days = 7
  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id
    key_permissions = [
      "Create",
      "Delete",
      "Get",
      "Purge",
      "Recover",
      "Update",
      "GetRotationPolicy",
    ]
    secret_permissions = [
      "Delete",
      "Get",
      "Set",
    ]
    certificate_permissions = [
      "Create",
      "Delete",
      "DeleteIssuers",
      "Get",
      "Purge",
      "Update"
    ]
  }
  tags = {
    environment = "Production"
  }
}
resource "azurerm_key_vault_certificate" "cert" {
  count        = 3
  name         = "acchsmcert${count.index}"
  key_vault_id = azurerm_key_vault.test.id
  certificate_policy {
    issuer_parameters {
      name = "Self"
    }
    key_properties {
      exportable = true
      key_size   = 2048
      key_type   = "RSA"
      reuse_key  = true
    }
    lifetime_action {
      action {
        action_type = "AutoRenew"
      }
      trigger {
        days_before_expiry = 30
      }
    }
    secret_properties {
      content_type = "application/x-pkcs12"
    }
    x509_certificate_properties {
      extended_key_usage = []
      key_usage = [
        "cRLSign",
        "dataEncipherment",
        "digitalSignature",
        "keyAgreement",
        "keyCertSign",
        "keyEncipherment",
      ]
      subject            = "CN=hello-world"
      validity_in_months = 12
    }
  }
}

resource "azurerm_key_vault_managed_hardware_security_module" "test" {
  name                     = "kvHsm%[2]s"
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  sku_name                 = "Standard_B1"
  tenant_id                = data.azurerm_client_config.current.tenant_id
  admin_object_ids         = [data.azurerm_client_config.current.object_id]
  purge_protection_enabled = false

  security_domain_key_vault_certificate_ids = [for cert in azurerm_key_vault_certificate.cert : cert.id]
  security_domain_quorum                    = 3
}

resource "azurerm_user_assigned_identity" "test" {
  name                = "acctestmi%[2]s"
  location            = azurerm_resource_group.test.location
  resource_group_name = azurerm_resource_group.test.name
}

resource "azurerm_key_vault_managed_hardware_security_module_role_assignment" "test" {
  vault_base_url     = azurerm_key_vault_managed_hardware_security_module.test.hsm_uri
  name               = "1e243909-064c-6ac3-84e9-1c8bf8d6ad22"
  scope              = "/keys"
  role_definition_id = "/Microsoft.KeyVault/providers/Microsoft.Authorization/roleDefinitions/21dbd100-6940-42c2-9190-5d6cb909625b"
  principal_id       = data.azurerm_client_config.current.object_id
}

resource "azurerm_key_vault_managed_hardware_security_module_role_assignment" "test1" {
  vault_base_url     = azurerm_key_vault_managed_hardware_security_module.test.hsm_uri
  name               = "1e243909-064c-6ac3-84e9-1c8bf8d6ad23"
  scope              = "/keys"
  role_definition_id = "/Microsoft.KeyVault/providers/Microsoft.Authorization/roleDefinitions/515eb02d-2335-4d2d-92f2-b1cbdf9c3778"
  principal_id       = data.azurerm_client_config.current.object_id
}

resource "azurerm_key_vault_managed_hardware_security_module_role_assignment" "user" {
  vault_base_url     = azurerm_key_vault_managed_hardware_security_module.test.hsm_uri
  name               = "1e243909-064c-6ac3-84e9-1c8bf8d6ad20"
  scope              = "/keys"
  role_definition_id = "/Microsoft.KeyVault/providers/Microsoft.Authorization/roleDefinitions/21dbd100-6940-42c2-9190-5d6cb909625b"
  principal_id       = azurerm_user_assigned_identity.test.principal_id
}

resource "azurerm_key_vault_managed_hardware_security_module_key" "test" {
  name           = "acctestHSMK-%[2]s"
  managed_hsm_id = azurerm_key_vault_managed_hardware_security_module.test.id
  key_type       = "RSA-HSM"
  key_size       = 2048
  key_opts       = ["unwrapKey", "wrapKey"]

  depends_on = [
    azurerm_key_vault_managed_hardware_security_module_role_assignment.test,
    azurerm_key_vault_managed_hardware_security_module_role_assignment.test1
  ]
}

resource "azurerm_mssql_server" "test" {
  name                         = "acctestsqlserver-%[2]s"
  resource_group_name          = azurerm_resource_group.test.name
  location                     = azurerm_resource_group.test.location
  version                      = "12.0"
  administrator_login          = "mradministrator"
  administrator_login_password = "thisIsDog11"

  identity {
    type = "SystemAssigned, UserAssigned"
    identity_ids = [
      azurerm_user_assigned_identity.test.id,
    ]
  }

  primary_user_assigned_identity_id = azurerm_user_assigned_identity.test.id

  lifecycle {
    ignore_changes = [transparent_data_encryption_key_vault_key_id]
  }
}
`, data.Locations.Primary, data.RandomStringOfLength(5))
}
