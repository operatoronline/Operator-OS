package shopify

import (
	"context"
	"sort"
	"testing"

	"github.com/standardws/operator/pkg/integrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- OAuth Provider ---

func TestNewOAuthProvider(t *testing.T) {
	p := NewOAuthProvider("client-id", "client-secret", "https://example.com/callback", "mystore")
	require.NotNil(t, p)
	assert.Equal(t, ProviderID, p.ID)
	assert.Equal(t, "Shopify", p.Name)
	assert.Equal(t, "https://mystore.myshopify.com/admin/oauth/authorize", p.AuthURL)
	assert.Equal(t, "https://mystore.myshopify.com/admin/oauth/access_token", p.TokenURL)
	assert.Equal(t, "client-id", p.ClientID)
	assert.Equal(t, "client-secret", p.ClientSecret)
	assert.Equal(t, "https://example.com/callback", p.RedirectURL)
	assert.False(t, p.UsePKCE, "Shopify doesn't support PKCE")
	assert.Contains(t, p.Scopes, ScopeReadProducts)
	assert.Contains(t, p.Scopes, ScopeReadOrders)
	assert.Contains(t, p.Scopes, ScopeReadCustomers)
}

func TestNewOAuthProviderDifferentShop(t *testing.T) {
	p := NewOAuthProvider("key", "secret", "https://cb.com", "other-store")
	assert.Contains(t, p.AuthURL, "other-store.myshopify.com")
	assert.Contains(t, p.TokenURL, "other-store.myshopify.com")
}

// --- Integration Manifests ---

func TestProductsIntegration(t *testing.T) {
	integ := ProductsIntegration()
	require.NotNil(t, integ)
	assert.Equal(t, ProductsIntegrationID, integ.ID)
	assert.Equal(t, "Shopify Products", integ.Name)
	assert.Equal(t, "ecommerce", integ.Category)
	assert.Equal(t, integrations.AuthTypeOAuth2, integ.AuthType)
	assert.Equal(t, integrations.IntegrationStatusActive, integ.Status)
	assert.Equal(t, "starter", integ.RequiredPlan)
	assert.Equal(t, "1.0.0", integ.Version)

	// OAuth config.
	require.NotNil(t, integ.OAuth)
	assert.Equal(t, AuthorizationURLTemplate, integ.OAuth.AuthorizationURL)
	assert.Equal(t, TokenURLTemplate, integ.OAuth.TokenURL)
	assert.False(t, integ.OAuth.UsePKCE)
	assert.Contains(t, integ.OAuth.Scopes, ScopeReadProducts)
	assert.Contains(t, integ.OAuth.Scopes, ScopeWriteProducts)

	// Dynamic params.
	require.Contains(t, integ.OAuth.DynamicParams, "shop")
	shopParam := integ.OAuth.DynamicParams["shop"]
	assert.Equal(t, "Shop Name", shopParam.Label)
	assert.True(t, shopParam.Required)
	assert.NotEmpty(t, shopParam.Validation)

	// Tools.
	assert.Len(t, integ.Tools, 7)
	err := integ.Validate()
	assert.NoError(t, err)
}

func TestOrdersIntegration(t *testing.T) {
	integ := OrdersIntegration()
	require.NotNil(t, integ)
	assert.Equal(t, OrdersIntegrationID, integ.ID)
	assert.Equal(t, "Shopify Orders", integ.Name)
	assert.Equal(t, "ecommerce", integ.Category)
	assert.Equal(t, integrations.AuthTypeOAuth2, integ.AuthType)
	assert.Contains(t, integ.OAuth.Scopes, ScopeReadOrders)
	assert.Contains(t, integ.OAuth.Scopes, ScopeWriteOrders)
	assert.Len(t, integ.Tools, 7)
	assert.NoError(t, integ.Validate())
}

func TestCustomersIntegration(t *testing.T) {
	integ := CustomersIntegration()
	require.NotNil(t, integ)
	assert.Equal(t, CustomersIntegrationID, integ.ID)
	assert.Equal(t, "Shopify Customers", integ.Name)
	assert.Equal(t, "ecommerce", integ.Category)
	assert.Contains(t, integ.OAuth.Scopes, ScopeReadCustomers)
	assert.Contains(t, integ.OAuth.Scopes, ScopeWriteCustomers)
	assert.Len(t, integ.Tools, 7)
	assert.NoError(t, integ.Validate())
}

func TestAllIntegrations(t *testing.T) {
	all := AllIntegrations()
	assert.Len(t, all, 3)

	ids := make(map[string]bool)
	for _, integ := range all {
		ids[integ.ID] = true
		assert.NoError(t, integ.Validate())
	}
	assert.True(t, ids[ProductsIntegrationID])
	assert.True(t, ids[OrdersIntegrationID])
	assert.True(t, ids[CustomersIntegrationID])
}

func TestAllIntegrationsEcommerce(t *testing.T) {
	for _, integ := range AllIntegrations() {
		assert.Equal(t, "ecommerce", integ.Category)
	}
}

func TestAllIntegrationsDynamicShop(t *testing.T) {
	for _, integ := range AllIntegrations() {
		require.NotNil(t, integ.OAuth)
		require.Contains(t, integ.OAuth.DynamicParams, "shop")
	}
}

// --- Tool Definitions ---

func TestProductToolNames(t *testing.T) {
	integ := ProductsIntegration()
	names := integ.ToolNames()
	assert.Contains(t, names, ToolProductsList)
	assert.Contains(t, names, ToolProductsGet)
	assert.Contains(t, names, ToolProductsCreate)
	assert.Contains(t, names, ToolProductsUpdate)
	assert.Contains(t, names, ToolProductsDelete)
	assert.Contains(t, names, ToolProductsCount)
	assert.Contains(t, names, ToolProductsSearch)
}

func TestOrderToolNames(t *testing.T) {
	integ := OrdersIntegration()
	names := integ.ToolNames()
	assert.Contains(t, names, ToolOrdersList)
	assert.Contains(t, names, ToolOrdersGet)
	assert.Contains(t, names, ToolOrdersCount)
	assert.Contains(t, names, ToolOrdersClose)
	assert.Contains(t, names, ToolOrdersCancel)
	assert.Contains(t, names, ToolOrdersCreate)
	assert.Contains(t, names, ToolOrdersUpdate)
}

func TestCustomerToolNames(t *testing.T) {
	integ := CustomersIntegration()
	names := integ.ToolNames()
	assert.Contains(t, names, ToolCustomersList)
	assert.Contains(t, names, ToolCustomersGet)
	assert.Contains(t, names, ToolCustomersCreate)
	assert.Contains(t, names, ToolCustomersUpdate)
	assert.Contains(t, names, ToolCustomersSearch)
	assert.Contains(t, names, ToolCustomersCount)
	assert.Contains(t, names, ToolCustomersOrders)
}

func TestToolsHaveDescriptions(t *testing.T) {
	for _, integ := range AllIntegrations() {
		for _, tool := range integ.Tools {
			assert.NotEmpty(t, tool.Description, "tool %s should have a description", tool.Name)
		}
	}
}

func TestToolsHaveParameters(t *testing.T) {
	for _, integ := range AllIntegrations() {
		for _, tool := range integ.Tools {
			assert.NotNil(t, tool.Parameters, "tool %s should have parameters", tool.Name)
			assert.Equal(t, "object", tool.Parameters["type"], "tool %s params should be object", tool.Name)
		}
	}
}

func TestToolsHaveRequiredScopes(t *testing.T) {
	for _, integ := range AllIntegrations() {
		for _, tool := range integ.Tools {
			assert.NotEmpty(t, tool.RequiredScopes, "tool %s should have required scopes", tool.Name)
		}
	}
}

func TestToolsHaveRateLimits(t *testing.T) {
	for _, integ := range AllIntegrations() {
		for _, tool := range integ.Tools {
			assert.Greater(t, tool.RateLimit, 0, "tool %s should have a rate limit", tool.Name)
		}
	}
}

func TestToolNamesUniqueAcrossIntegrations(t *testing.T) {
	seen := make(map[string]bool)
	for _, integ := range AllIntegrations() {
		for _, tool := range integ.Tools {
			assert.False(t, seen[tool.Name], "duplicate tool name: %s", tool.Name)
			seen[tool.Name] = true
		}
	}
}

func TestProductCreateRequiredFields(t *testing.T) {
	integ := ProductsIntegration()
	for _, tool := range integ.Tools {
		if tool.Name == ToolProductsCreate {
			required, ok := tool.Parameters["required"].([]string)
			require.True(t, ok)
			assert.Contains(t, required, "title")
			return
		}
	}
	t.Fatal("product create tool not found")
}

func TestOrderCreateRequiredFields(t *testing.T) {
	integ := OrdersIntegration()
	for _, tool := range integ.Tools {
		if tool.Name == ToolOrdersCreate {
			required, ok := tool.Parameters["required"].([]string)
			require.True(t, ok)
			assert.Contains(t, required, "line_items")
			return
		}
	}
	t.Fatal("order create tool not found")
}

func TestCustomerCreateRequiredFields(t *testing.T) {
	integ := CustomersIntegration()
	for _, tool := range integ.Tools {
		if tool.Name == ToolCustomersCreate {
			required, ok := tool.Parameters["required"].([]string)
			require.True(t, ok)
			assert.Contains(t, required, "email")
			return
		}
	}
	t.Fatal("customer create tool not found")
}

func TestWriteToolsRequireWriteScopes(t *testing.T) {
	writeTools := map[string]string{
		ToolProductsCreate: ScopeWriteProducts,
		ToolProductsUpdate: ScopeWriteProducts,
		ToolProductsDelete: ScopeWriteProducts,
		ToolOrdersCreate:   ScopeWriteOrders,
		ToolOrdersUpdate:   ScopeWriteOrders,
		ToolOrdersClose:    ScopeWriteOrders,
		ToolOrdersCancel:   ScopeWriteOrders,
		ToolCustomersCreate: ScopeWriteCustomers,
		ToolCustomersUpdate: ScopeWriteCustomers,
	}

	allTools := make(map[string]integrations.ToolManifest)
	for _, integ := range AllIntegrations() {
		for _, tool := range integ.Tools {
			allTools[tool.Name] = tool
		}
	}

	for toolName, expectedScope := range writeTools {
		tool, ok := allTools[toolName]
		require.True(t, ok, "tool %s not found", toolName)
		assert.Contains(t, tool.RequiredScopes, expectedScope, "tool %s should require %s", toolName, expectedScope)
	}
}

// --- Registry Integration ---

func TestRegisterAll(t *testing.T) {
	registry := integrations.NewIntegrationRegistry()
	count, err := RegisterAll(registry)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, 3, registry.Count())
}

func TestRegisterAllCategories(t *testing.T) {
	registry := integrations.NewIntegrationRegistry()
	_, err := RegisterAll(registry)
	require.NoError(t, err)

	cats := registry.Categories()
	assert.Contains(t, cats, "ecommerce")
}

func TestRegisterAllListByCategory(t *testing.T) {
	registry := integrations.NewIntegrationRegistry()
	_, err := RegisterAll(registry)
	require.NoError(t, err)

	ecomm := registry.ListByCategory("ecommerce")
	assert.Len(t, ecomm, 3)
}

func TestRegisterAllDuplicateError(t *testing.T) {
	registry := integrations.NewIntegrationRegistry()
	_, err := RegisterAll(registry)
	require.NoError(t, err)

	// Registering again should fail.
	_, err = RegisterAll(registry)
	assert.Error(t, err)
}

func TestRegisterAllToolLookup(t *testing.T) {
	registry := integrations.NewIntegrationRegistry()
	_, err := RegisterAll(registry)
	require.NoError(t, err)

	tool, integID := registry.GetToolManifest(ToolProductsList)
	require.NotNil(t, tool)
	assert.Equal(t, ProductsIntegrationID, integID)
	assert.Equal(t, ToolProductsList, tool.Name)

	tool, integID = registry.GetToolManifest(ToolOrdersGet)
	require.NotNil(t, tool)
	assert.Equal(t, OrdersIntegrationID, integID)

	tool, integID = registry.GetToolManifest(ToolCustomersSearch)
	require.NotNil(t, tool)
	assert.Equal(t, CustomersIntegrationID, integID)
}

func TestRegisterAllToolNames(t *testing.T) {
	registry := integrations.NewIntegrationRegistry()
	_, err := RegisterAll(registry)
	require.NoError(t, err)

	names := registry.AllToolNames()
	assert.Len(t, names, 21) // 7 products + 7 orders + 7 customers
	assert.True(t, sort.StringsAreSorted(names))
}

// --- Endpoint Resolution ---

func TestResolveEndpointProducts(t *testing.T) {
	tests := []struct {
		tool   string
		args   map[string]any
		method string
		path   string
	}{
		{ToolProductsList, nil, "GET", "/admin/api/" + APIVersion + "/products.json"},
		{ToolProductsGet, map[string]any{"product_id": "123"}, "GET", "/admin/api/" + APIVersion + "/products/123.json"},
		{ToolProductsCreate, nil, "POST", "/admin/api/" + APIVersion + "/products.json"},
		{ToolProductsUpdate, map[string]any{"product_id": "456"}, "PUT", "/admin/api/" + APIVersion + "/products/456.json"},
		{ToolProductsDelete, map[string]any{"product_id": "789"}, "DELETE", "/admin/api/" + APIVersion + "/products/789.json"},
		{ToolProductsCount, nil, "GET", "/admin/api/" + APIVersion + "/products/count.json"},
		{ToolProductsSearch, nil, "GET", "/admin/api/" + APIVersion + "/products.json"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			method, path, err := ResolveEndpoint(ProductsIntegrationID, tt.tool, tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.method, method)
			assert.Equal(t, tt.path, path)
		})
	}
}

func TestResolveEndpointOrders(t *testing.T) {
	tests := []struct {
		tool   string
		args   map[string]any
		method string
		path   string
	}{
		{ToolOrdersList, nil, "GET", "/admin/api/" + APIVersion + "/orders.json"},
		{ToolOrdersGet, map[string]any{"order_id": "100"}, "GET", "/admin/api/" + APIVersion + "/orders/100.json"},
		{ToolOrdersCount, nil, "GET", "/admin/api/" + APIVersion + "/orders/count.json"},
		{ToolOrdersClose, map[string]any{"order_id": "100"}, "POST", "/admin/api/" + APIVersion + "/orders/100/close.json"},
		{ToolOrdersCancel, map[string]any{"order_id": "100"}, "POST", "/admin/api/" + APIVersion + "/orders/100/cancel.json"},
		{ToolOrdersCreate, nil, "POST", "/admin/api/" + APIVersion + "/orders.json"},
		{ToolOrdersUpdate, map[string]any{"order_id": "100"}, "PUT", "/admin/api/" + APIVersion + "/orders/100.json"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			method, path, err := ResolveEndpoint(OrdersIntegrationID, tt.tool, tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.method, method)
			assert.Equal(t, tt.path, path)
		})
	}
}

func TestResolveEndpointCustomers(t *testing.T) {
	tests := []struct {
		tool   string
		args   map[string]any
		method string
		path   string
	}{
		{ToolCustomersList, nil, "GET", "/admin/api/" + APIVersion + "/customers.json"},
		{ToolCustomersGet, map[string]any{"customer_id": "200"}, "GET", "/admin/api/" + APIVersion + "/customers/200.json"},
		{ToolCustomersCreate, nil, "POST", "/admin/api/" + APIVersion + "/customers.json"},
		{ToolCustomersUpdate, map[string]any{"customer_id": "200"}, "PUT", "/admin/api/" + APIVersion + "/customers/200.json"},
		{ToolCustomersSearch, nil, "GET", "/admin/api/" + APIVersion + "/customers/search.json"},
		{ToolCustomersCount, nil, "GET", "/admin/api/" + APIVersion + "/customers/count.json"},
		{ToolCustomersOrders, map[string]any{"customer_id": "200"}, "GET", "/admin/api/" + APIVersion + "/customers/200/orders.json"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			method, path, err := ResolveEndpoint(CustomersIntegrationID, tt.tool, tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.method, method)
			assert.Equal(t, tt.path, path)
		})
	}
}

func TestResolveEndpointUnknownTool(t *testing.T) {
	_, _, err := ResolveEndpoint(ProductsIntegrationID, "unknown_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestResolveEndpointUnresolvedPlaceholders(t *testing.T) {
	// product_id placeholder not provided.
	_, _, err := ResolveEndpoint(ProductsIntegrationID, ToolProductsGet, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved")
}

func TestResolveEndpointPathEscaping(t *testing.T) {
	method, path, err := ResolveEndpoint(ProductsIntegrationID, ToolProductsGet, map[string]any{
		"product_id": "id/with/slashes",
	})
	require.NoError(t, err)
	assert.Equal(t, "GET", method)
	assert.Contains(t, path, "id%2Fwith%2Fslashes")
	assert.NotContains(t, path, "{")
}

// --- Base URL ---

func TestResolveBaseURL(t *testing.T) {
	assert.Equal(t, "https://mystore.myshopify.com", ResolveBaseURL("mystore"))
	assert.Equal(t, "https://test-shop.myshopify.com", ResolveBaseURL("test-shop"))
}

func TestResolveBaseURLEmpty(t *testing.T) {
	url := ResolveBaseURL("")
	assert.Equal(t, "https://unknown.myshopify.com", url)
}

// --- Endpoint Spec ---

func TestGetEndpointSpec(t *testing.T) {
	spec, ok := GetEndpointSpec(ToolProductsList)
	assert.True(t, ok)
	assert.Equal(t, "GET", spec.Method)
	assert.Contains(t, spec.BasePath, "/products.json")

	spec, ok = GetEndpointSpec(ToolOrdersCancel)
	assert.True(t, ok)
	assert.Equal(t, "POST", spec.Method)
	assert.Contains(t, spec.BasePath, "/cancel.json")
}

func TestGetEndpointSpecNotFound(t *testing.T) {
	_, ok := GetEndpointSpec("nonexistent_tool")
	assert.False(t, ok)
}

// --- AllToolNames ---

func TestAllToolNamesCount(t *testing.T) {
	names := AllToolNames()
	assert.Len(t, names, 21) // 7 + 7 + 7
}

func TestAllToolNamesContains(t *testing.T) {
	names := AllToolNames()
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	// Spot check each category.
	assert.True(t, nameSet[ToolProductsList])
	assert.True(t, nameSet[ToolOrdersGet])
	assert.True(t, nameSet[ToolCustomersSearch])
}

// --- Constants ---

func TestConstants(t *testing.T) {
	assert.Equal(t, "shopify", ProviderID)
	assert.Equal(t, "shopify_products", ProductsIntegrationID)
	assert.Equal(t, "shopify_orders", OrdersIntegrationID)
	assert.Equal(t, "shopify_customers", CustomersIntegrationID)
}

func TestScopeConstants(t *testing.T) {
	assert.Equal(t, "read_products", ScopeReadProducts)
	assert.Equal(t, "write_products", ScopeWriteProducts)
	assert.Equal(t, "read_orders", ScopeReadOrders)
	assert.Equal(t, "write_orders", ScopeWriteOrders)
	assert.Equal(t, "read_customers", ScopeReadCustomers)
	assert.Equal(t, "write_customers", ScopeWriteCustomers)
}

func TestAPIVersion(t *testing.T) {
	assert.Equal(t, "2024-10", APIVersion)
}

func TestAuthorizationURLTemplate(t *testing.T) {
	assert.Contains(t, AuthorizationURLTemplate, "{shop}")
	assert.Contains(t, TokenURLTemplate, "{shop}")
}

// --- Validation ---

func TestIntegrationValidation(t *testing.T) {
	for _, integ := range AllIntegrations() {
		t.Run(integ.ID, func(t *testing.T) {
			err := integ.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestIntegrationValidationMutation(t *testing.T) {
	integ := ProductsIntegration()
	integ.Name = ""
	err := integ.Validate()
	assert.Error(t, err)
}

// --- Customer Orders cross-scope ---

func TestCustomerOrdersRequiresBothScopes(t *testing.T) {
	integ := CustomersIntegration()
	for _, tool := range integ.Tools {
		if tool.Name == ToolCustomersOrders {
			assert.Contains(t, tool.RequiredScopes, ScopeReadCustomers)
			assert.Contains(t, tool.RequiredScopes, ScopeReadOrders)
			return
		}
	}
	t.Fatal("customer orders tool not found")
}

// --- Versioning ---

func TestAllIntegrationsVersioned(t *testing.T) {
	for _, integ := range AllIntegrations() {
		assert.Equal(t, "1.0.0", integ.Version, "integration %s should be versioned", integ.ID)
	}
}

// --- Required Plan ---

func TestAllIntegrationsRequireStarter(t *testing.T) {
	for _, integ := range AllIntegrations() {
		assert.Equal(t, "starter", integ.RequiredPlan, "integration %s should require starter plan", integ.ID)
	}
}

// --- HTTP Executor ---

func TestNewShopifyHTTPExecutor(t *testing.T) {
	executor := NewShopifyHTTPExecutor(
		func(ctx context.Context, userID, integrationID string) (string, error) {
			return "token", nil
		},
		func(ctx context.Context, userID string) (string, error) {
			return "myshop", nil
		},
	)
	assert.NotNil(t, executor)
}
