package security

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"context"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/tracing"
	"net/http"
)

// DeviceClient is the API spec for Microsoft.Security (Azure Security Center) resource provider
type DeviceClient struct {
	BaseClient
}

// NewDeviceClient creates an instance of the DeviceClient client.
func NewDeviceClient(subscriptionID string, ascLocation string) DeviceClient {
	return NewDeviceClientWithBaseURI(DefaultBaseURI, subscriptionID, ascLocation)
}

// NewDeviceClientWithBaseURI creates an instance of the DeviceClient client using a custom endpoint.  Use this when
// interacting with an Azure cloud that uses a non-standard base URI (sovereign clouds, Azure stack).
func NewDeviceClientWithBaseURI(baseURI string, subscriptionID string, ascLocation string) DeviceClient {
	return DeviceClient{NewWithBaseURI(baseURI, subscriptionID, ascLocation)}
}

// Get get device.
// Parameters:
// resourceID - the identifier of the resource.
// deviceID - identifier of the device.
func (client DeviceClient) Get(ctx context.Context, resourceID string, deviceID string) (result Device, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/DeviceClient.Get")
		defer func() {
			sc := -1
			if result.Response.Response != nil {
				sc = result.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.GetPreparer(ctx, resourceID, deviceID)
	if err != nil {
		err = autorest.NewErrorWithError(err, "security.DeviceClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "security.DeviceClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "security.DeviceClient", "Get", resp, "Failure responding to request")
		return
	}

	return
}

// GetPreparer prepares the Get request.
func (client DeviceClient) GetPreparer(ctx context.Context, resourceID string, deviceID string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"deviceId":   autorest.Encode("path", deviceID),
		"resourceId": resourceID,
	}

	const APIVersion = "2020-08-06-preview"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/{resourceId}/providers/Microsoft.Security/devices/{deviceId}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client DeviceClient) GetSender(req *http.Request) (*http.Response, error) {
	return client.Send(req, autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client DeviceClient) GetResponder(resp *http.Response) (result Device, err error) {
	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
