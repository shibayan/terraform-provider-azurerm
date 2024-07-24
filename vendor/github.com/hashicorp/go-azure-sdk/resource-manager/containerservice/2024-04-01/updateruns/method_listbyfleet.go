package updateruns

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-azure-sdk/sdk/client"
	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type ListByFleetOperationResponse struct {
	HttpResponse *http.Response
	OData        *odata.OData
	Model        *[]UpdateRun
}

type ListByFleetCompleteResult struct {
	LatestHttpResponse *http.Response
	Items              []UpdateRun
}

type ListByFleetCustomPager struct {
	NextLink *odata.Link `json:"nextLink"`
}

func (p *ListByFleetCustomPager) NextPageLink() *odata.Link {
	defer func() {
		p.NextLink = nil
	}()

	return p.NextLink
}

// ListByFleet ...
func (c UpdateRunsClient) ListByFleet(ctx context.Context, id FleetId) (result ListByFleetOperationResponse, err error) {
	opts := client.RequestOptions{
		ContentType: "application/json; charset=utf-8",
		ExpectedStatusCodes: []int{
			http.StatusOK,
		},
		HttpMethod: http.MethodGet,
		Pager:      &ListByFleetCustomPager{},
		Path:       fmt.Sprintf("%s/updateRuns", id.ID()),
	}

	req, err := c.Client.NewRequest(ctx, opts)
	if err != nil {
		return
	}

	var resp *client.Response
	resp, err = req.ExecutePaged(ctx)
	if resp != nil {
		result.OData = resp.OData
		result.HttpResponse = resp.Response
	}
	if err != nil {
		return
	}

	var values struct {
		Values *[]UpdateRun `json:"value"`
	}
	if err = resp.Unmarshal(&values); err != nil {
		return
	}

	result.Model = values.Values

	return
}

// ListByFleetComplete retrieves all the results into a single object
func (c UpdateRunsClient) ListByFleetComplete(ctx context.Context, id FleetId) (ListByFleetCompleteResult, error) {
	return c.ListByFleetCompleteMatchingPredicate(ctx, id, UpdateRunOperationPredicate{})
}

// ListByFleetCompleteMatchingPredicate retrieves all the results and then applies the predicate
func (c UpdateRunsClient) ListByFleetCompleteMatchingPredicate(ctx context.Context, id FleetId, predicate UpdateRunOperationPredicate) (result ListByFleetCompleteResult, err error) {
	items := make([]UpdateRun, 0)

	resp, err := c.ListByFleet(ctx, id)
	if err != nil {
		result.LatestHttpResponse = resp.HttpResponse
		err = fmt.Errorf("loading results: %+v", err)
		return
	}
	if resp.Model != nil {
		for _, v := range *resp.Model {
			if predicate.Matches(v) {
				items = append(items, v)
			}
		}
	}

	result = ListByFleetCompleteResult{
		LatestHttpResponse: resp.HttpResponse,
		Items:              items,
	}
	return
}
