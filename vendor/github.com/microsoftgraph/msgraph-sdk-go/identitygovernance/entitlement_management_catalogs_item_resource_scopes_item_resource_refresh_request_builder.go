// Code generated by Microsoft Kiota - DO NOT EDIT.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

package identitygovernance

import (
    "context"
    i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f "github.com/microsoft/kiota-abstractions-go"
    ia572726a95efa92ddd544552cd950653dc691023836923576b2f4bf716cf204a "github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
)

// EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder provides operations to call the refresh method.
type EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder struct {
    i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.BaseRequestBuilder
}
// EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilderPostRequestConfiguration configuration for the request such as headers, query parameters, and middleware options.
type EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilderPostRequestConfiguration struct {
    // Request headers
    Headers *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestHeaders
    // Request options
    Options []i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestOption
}
// NewEntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilderInternal instantiates a new EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder and sets the default values.
func NewEntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilderInternal(pathParameters map[string]string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter)(*EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder) {
    m := &EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder{
        BaseRequestBuilder: *i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewBaseRequestBuilder(requestAdapter, "{+baseurl}/identityGovernance/entitlementManagement/catalogs/{accessPackageCatalog%2Did}/resourceScopes/{accessPackageResourceScope%2Did}/resource/refresh", pathParameters),
    }
    return m
}
// NewEntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder instantiates a new EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder and sets the default values.
func NewEntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder(rawUrl string, requestAdapter i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestAdapter)(*EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder) {
    urlParams := make(map[string]string)
    urlParams["request-raw-url"] = rawUrl
    return NewEntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilderInternal(urlParams, requestAdapter)
}
// Post invoke action refresh
// returns a ODataError error when the service returns a 4XX or 5XX status code
func (m *EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder) Post(ctx context.Context, requestConfiguration *EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilderPostRequestConfiguration)(error) {
    requestInfo, err := m.ToPostRequestInformation(ctx, requestConfiguration);
    if err != nil {
        return err
    }
    errorMapping := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.ErrorMappings {
        "XXX": ia572726a95efa92ddd544552cd950653dc691023836923576b2f4bf716cf204a.CreateODataErrorFromDiscriminatorValue,
    }
    err = m.BaseRequestBuilder.RequestAdapter.SendNoContent(ctx, requestInfo, errorMapping)
    if err != nil {
        return err
    }
    return nil
}
// ToPostRequestInformation invoke action refresh
// returns a *RequestInformation when successful
func (m *EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder) ToPostRequestInformation(ctx context.Context, requestConfiguration *EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilderPostRequestConfiguration)(*i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.RequestInformation, error) {
    requestInfo := i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.NewRequestInformationWithMethodAndUrlTemplateAndPathParameters(i2ae4187f7daee263371cb1c977df639813ab50ffa529013b7437480d1ec0158f.POST, m.BaseRequestBuilder.UrlTemplate, m.BaseRequestBuilder.PathParameters)
    if requestConfiguration != nil {
        requestInfo.Headers.AddAll(requestConfiguration.Headers)
        requestInfo.AddRequestOptions(requestConfiguration.Options)
    }
    requestInfo.Headers.TryAdd("Accept", "application/json")
    return requestInfo, nil
}
// WithUrl returns a request builder with the provided arbitrary URL. Using this method means any other path or query parameters are ignored.
// returns a *EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder when successful
func (m *EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder) WithUrl(rawUrl string)(*EntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder) {
    return NewEntitlementManagementCatalogsItemResourceScopesItemResourceRefreshRequestBuilder(rawUrl, m.BaseRequestBuilder.RequestAdapter);
}
