package restorables

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type RestorableDatabaseAccountsListByLocationOperationResponse struct {
	HttpResponse *http.Response
	Model        *RestorableDatabaseAccountsListResult
}

// RestorableDatabaseAccountsListByLocation ...
func (c RestorablesClient) RestorableDatabaseAccountsListByLocation(ctx context.Context, id LocationId) (result RestorableDatabaseAccountsListByLocationOperationResponse, err error) {
	req, err := c.preparerForRestorableDatabaseAccountsListByLocation(ctx, id)
	if err != nil {
		err = autorest.NewErrorWithError(err, "restorables.RestorablesClient", "RestorableDatabaseAccountsListByLocation", nil, "Failure preparing request")
		return
	}

	result.HttpResponse, err = c.Client.Send(req, azure.DoRetryWithRegistration(c.Client))
	if err != nil {
		err = autorest.NewErrorWithError(err, "restorables.RestorablesClient", "RestorableDatabaseAccountsListByLocation", result.HttpResponse, "Failure sending request")
		return
	}

	result, err = c.responderForRestorableDatabaseAccountsListByLocation(result.HttpResponse)
	if err != nil {
		err = autorest.NewErrorWithError(err, "restorables.RestorablesClient", "RestorableDatabaseAccountsListByLocation", result.HttpResponse, "Failure responding to request")
		return
	}

	return
}

// preparerForRestorableDatabaseAccountsListByLocation prepares the RestorableDatabaseAccountsListByLocation request.
func (c RestorablesClient) preparerForRestorableDatabaseAccountsListByLocation(ctx context.Context, id LocationId) (*http.Request, error) {
	queryParameters := map[string]interface{}{
		"api-version": defaultApiVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsGet(),
		autorest.WithBaseURL(c.baseUri),
		autorest.WithPath(fmt.Sprintf("%s/restorableDatabaseAccounts", id.ID())),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// responderForRestorableDatabaseAccountsListByLocation handles the response to the RestorableDatabaseAccountsListByLocation request. The method always
// closes the http.Response Body.
func (c RestorablesClient) responderForRestorableDatabaseAccountsListByLocation(resp *http.Response) (result RestorableDatabaseAccountsListByLocationOperationResponse, err error) {
	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result.Model),
		autorest.ByClosing())
	result.HttpResponse = resp

	return
}
