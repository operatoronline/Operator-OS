package shopify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/standardws/operator/pkg/integrations"
)

// EndpointSpec describes an HTTP endpoint for a tool.
type EndpointSpec struct {
	Method   string
	BasePath string // Path template, may contain {param} placeholders.
}

// productEndpoints maps product tool names to their Admin API endpoints.
var productEndpoints = map[string]EndpointSpec{
	ToolProductsList:   {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/products.json"},
	ToolProductsGet:    {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/products/{product_id}.json"},
	ToolProductsCreate: {Method: http.MethodPost, BasePath: "/admin/api/" + APIVersion + "/products.json"},
	ToolProductsUpdate: {Method: http.MethodPut, BasePath: "/admin/api/" + APIVersion + "/products/{product_id}.json"},
	ToolProductsDelete: {Method: http.MethodDelete, BasePath: "/admin/api/" + APIVersion + "/products/{product_id}.json"},
	ToolProductsCount:  {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/products/count.json"},
	ToolProductsSearch: {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/products.json"},
}

// orderEndpoints maps order tool names to their Admin API endpoints.
var orderEndpoints = map[string]EndpointSpec{
	ToolOrdersList:   {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/orders.json"},
	ToolOrdersGet:    {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/orders/{order_id}.json"},
	ToolOrdersCount:  {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/orders/count.json"},
	ToolOrdersClose:  {Method: http.MethodPost, BasePath: "/admin/api/" + APIVersion + "/orders/{order_id}/close.json"},
	ToolOrdersCancel: {Method: http.MethodPost, BasePath: "/admin/api/" + APIVersion + "/orders/{order_id}/cancel.json"},
	ToolOrdersCreate: {Method: http.MethodPost, BasePath: "/admin/api/" + APIVersion + "/orders.json"},
	ToolOrdersUpdate: {Method: http.MethodPut, BasePath: "/admin/api/" + APIVersion + "/orders/{order_id}.json"},
}

// customerEndpoints maps customer tool names to their Admin API endpoints.
var customerEndpoints = map[string]EndpointSpec{
	ToolCustomersList:   {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/customers.json"},
	ToolCustomersGet:    {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/customers/{customer_id}.json"},
	ToolCustomersCreate: {Method: http.MethodPost, BasePath: "/admin/api/" + APIVersion + "/customers.json"},
	ToolCustomersUpdate: {Method: http.MethodPut, BasePath: "/admin/api/" + APIVersion + "/customers/{customer_id}.json"},
	ToolCustomersSearch: {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/customers/search.json"},
	ToolCustomersCount:  {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/customers/count.json"},
	ToolCustomersOrders: {Method: http.MethodGet, BasePath: "/admin/api/" + APIVersion + "/customers/{customer_id}/orders.json"},
}

// allEndpoints is a combined map of all Shopify endpoints.
var allEndpoints map[string]EndpointSpec

func init() {
	allEndpoints = make(map[string]EndpointSpec)
	for k, v := range productEndpoints {
		allEndpoints[k] = v
	}
	for k, v := range orderEndpoints {
		allEndpoints[k] = v
	}
	for k, v := range customerEndpoints {
		allEndpoints[k] = v
	}
}

// ResolveEndpoint returns the HTTP method and path for a Shopify tool,
// substituting any {param} placeholders from the args map.
func ResolveEndpoint(integrationID, toolName string, args map[string]any) (method, path string, err error) {
	spec, ok := allEndpoints[toolName]
	if !ok {
		return "", "", fmt.Errorf("unknown tool %q for integration %q", toolName, integrationID)
	}

	p := spec.BasePath

	// Replace path parameters from args.
	for key, val := range args {
		placeholder := "{" + key + "}"
		if strings.Contains(p, placeholder) {
			if s, ok := val.(string); ok {
				p = strings.ReplaceAll(p, placeholder, url.PathEscape(s))
			}
		}
	}

	// Check for unreplaced placeholders.
	if strings.Contains(p, "{") {
		return "", "", fmt.Errorf("unresolved path parameters in %q for tool %q", p, toolName)
	}

	return spec.Method, p, nil
}

// ResolveBaseURL returns the Shopify Admin API base URL for a shop.
// The shop parameter should be the store subdomain (e.g. "mystore").
func ResolveBaseURL(shop string) string {
	if shop == "" {
		return "https://unknown.myshopify.com"
	}
	return "https://" + shop + ".myshopify.com"
}

// NewShopifyHTTPExecutor creates an IntegrationToolExecutor for Shopify APIs
// using the generic HTTP executor from the integrations package.
// shopResolver maps a user ID to their Shopify shop subdomain.
func NewShopifyHTTPExecutor(
	tokenResolver func(ctx context.Context, userID, integrationID string) (string, error),
	shopResolver func(ctx context.Context, userID string) (string, error),
) integrations.IntegrationToolExecutor {
	return integrations.NewHTTPToolExecutor(integrations.HTTPToolExecutorConfig{
		BaseURLResolver: func(integrationID string) string {
			// The base URL depends on the shop, which is user-specific.
			// This is resolved dynamically in the executor; return empty here.
			return ""
		},
		TokenResolver: tokenResolver,
		EndpointResolver: func(integrationID, toolName string) (string, string, error) {
			spec, ok := allEndpoints[toolName]
			if !ok {
				return "", "", fmt.Errorf("unknown Shopify tool: %s", toolName)
			}
			return spec.Method, spec.BasePath, nil
		},
	})
}

// GetEndpointSpec returns the endpoint specification for a tool name.
// Returns false if the tool is not a Shopify tool.
func GetEndpointSpec(toolName string) (EndpointSpec, bool) {
	spec, ok := allEndpoints[toolName]
	return spec, ok
}

// AllToolNames returns the names of all Shopify tools across all integrations.
func AllToolNames() []string {
	names := make([]string, 0, len(allEndpoints))
	for name := range allEndpoints {
		names = append(names, name)
	}
	return names
}
