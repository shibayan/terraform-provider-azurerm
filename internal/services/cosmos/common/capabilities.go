// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2021-10-15/documentdb" // nolint: staticcheck
	"github.com/hashicorp/go-azure-sdk/resource-manager/cosmosdb/2023-04-15/cosmosdb"
)

func IsServerlessCapacityModeLegacy(accResp documentdb.DatabaseAccountGetResults) bool {
	if props := accResp.DatabaseAccountGetProperties; props != nil && props.Capabilities != nil {
		for _, v := range *props.Capabilities {
			if v.Name != nil && *v.Name == "EnableServerless" {
				return true
			}
		}
	}

	return false
}

func IsServerlessCapacityMode(accResp cosmosdb.DatabaseAccountGetResults) bool {
	if props := accResp.Properties; props != nil && props.Capabilities != nil {
		for _, v := range *props.Capabilities {
			if v.Name != nil && *v.Name == "EnableServerless" {
				return true
			}
		}
	}

	return false
}
