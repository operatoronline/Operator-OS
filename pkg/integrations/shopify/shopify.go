// Package shopify provides Shopify integration manifests and tool executors
// for the Shopify Admin API (Products, Orders, Customers).
package shopify

import (
	"github.com/standardws/operator/pkg/integrations"
	"github.com/standardws/operator/pkg/oauth"
)

// Provider IDs.
const (
	ProviderID = "shopify"
)

// Integration IDs.
const (
	ProductsIntegrationID  = "shopify_products"
	OrdersIntegrationID    = "shopify_orders"
	CustomersIntegrationID = "shopify_customers"
)

// Shopify OAuth endpoints (templates with {shop} placeholder).
const (
	AuthorizationURLTemplate = "https://{shop}.myshopify.com/admin/oauth/authorize"
	TokenURLTemplate         = "https://{shop}.myshopify.com/admin/oauth/access_token"
)

// Shopify Admin API base URL template.
const (
	AdminAPIBaseTemplate = "https://{shop}.myshopify.com"
)

// API version.
const (
	APIVersion = "2024-10"
)

// Shopify OAuth scopes.
const (
	ScopeReadProducts   = "read_products"
	ScopeWriteProducts  = "write_products"
	ScopeReadOrders     = "read_orders"
	ScopeWriteOrders    = "write_orders"
	ScopeReadCustomers  = "read_customers"
	ScopeWriteCustomers = "write_customers"
)

// NewOAuthProvider returns a Shopify OAuth 2.0 provider configuration.
// The shop parameter is the Shopify store subdomain (e.g. "mystore" for mystore.myshopify.com).
// clientID (API key) and clientSecret (API secret) are from the Shopify app.
func NewOAuthProvider(clientID, clientSecret, redirectURL, shop string) *oauth.Provider {
	authURL := "https://" + shop + ".myshopify.com/admin/oauth/authorize"
	tokenURL := "https://" + shop + ".myshopify.com/admin/oauth/access_token"

	return &oauth.Provider{
		ID:           ProviderID,
		Name:         "Shopify",
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		UsePKCE:      false, // Shopify doesn't support PKCE
		Scopes: []string{
			ScopeReadProducts,
			ScopeReadOrders,
			ScopeReadCustomers,
		},
	}
}

// ProductsIntegration returns the Shopify Products integration manifest.
func ProductsIntegration() *integrations.Integration {
	return &integrations.Integration{
		ID:          ProductsIntegrationID,
		Name:        "Shopify Products",
		Icon:        "shopify",
		Category:    "ecommerce",
		Description: "Manage products, variants, and inventory in your Shopify store.",
		AuthType:    integrations.AuthTypeOAuth2,
		OAuth: &integrations.OAuthConfig{
			AuthorizationURL: AuthorizationURLTemplate,
			TokenURL:         TokenURLTemplate,
			Scopes:           []string{ScopeReadProducts, ScopeWriteProducts},
			UsePKCE:          false,
			DynamicParams: map[string]integrations.DynamicParam{
				"shop": {
					Label:       "Shop Name",
					Placeholder: "mystore",
					Required:    true,
					Validation:  "^[a-zA-Z0-9][a-zA-Z0-9-]*$",
				},
			},
		},
		RequiredPlan: "starter",
		Tools:        productTools(),
		Status:       integrations.IntegrationStatusActive,
		Version:      "1.0.0",
	}
}

// OrdersIntegration returns the Shopify Orders integration manifest.
func OrdersIntegration() *integrations.Integration {
	return &integrations.Integration{
		ID:          OrdersIntegrationID,
		Name:        "Shopify Orders",
		Icon:        "shopify",
		Category:    "ecommerce",
		Description: "View and manage orders, fulfillments, and transactions in your Shopify store.",
		AuthType:    integrations.AuthTypeOAuth2,
		OAuth: &integrations.OAuthConfig{
			AuthorizationURL: AuthorizationURLTemplate,
			TokenURL:         TokenURLTemplate,
			Scopes:           []string{ScopeReadOrders, ScopeWriteOrders},
			UsePKCE:          false,
			DynamicParams: map[string]integrations.DynamicParam{
				"shop": {
					Label:       "Shop Name",
					Placeholder: "mystore",
					Required:    true,
					Validation:  "^[a-zA-Z0-9][a-zA-Z0-9-]*$",
				},
			},
		},
		RequiredPlan: "starter",
		Tools:        orderTools(),
		Status:       integrations.IntegrationStatusActive,
		Version:      "1.0.0",
	}
}

// CustomersIntegration returns the Shopify Customers integration manifest.
func CustomersIntegration() *integrations.Integration {
	return &integrations.Integration{
		ID:          CustomersIntegrationID,
		Name:        "Shopify Customers",
		Icon:        "shopify",
		Category:    "ecommerce",
		Description: "View, create, and manage customer records in your Shopify store.",
		AuthType:    integrations.AuthTypeOAuth2,
		OAuth: &integrations.OAuthConfig{
			AuthorizationURL: AuthorizationURLTemplate,
			TokenURL:         TokenURLTemplate,
			Scopes:           []string{ScopeReadCustomers, ScopeWriteCustomers},
			UsePKCE:          false,
			DynamicParams: map[string]integrations.DynamicParam{
				"shop": {
					Label:       "Shop Name",
					Placeholder: "mystore",
					Required:    true,
					Validation:  "^[a-zA-Z0-9][a-zA-Z0-9-]*$",
				},
			},
		},
		RequiredPlan: "starter",
		Tools:        customerTools(),
		Status:       integrations.IntegrationStatusActive,
		Version:      "1.0.0",
	}
}

// AllIntegrations returns all Shopify integration manifests.
func AllIntegrations() []*integrations.Integration {
	return []*integrations.Integration{
		ProductsIntegration(),
		OrdersIntegration(),
		CustomersIntegration(),
	}
}

// RegisterAll registers all Shopify integrations into the registry.
// Returns the number of integrations registered, or the first error.
func RegisterAll(registry *integrations.IntegrationRegistry) (int, error) {
	count := 0
	for _, integ := range AllIntegrations() {
		if err := registry.Register(integ); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
