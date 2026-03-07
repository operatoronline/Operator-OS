package shopify

import "github.com/standardws/operator/pkg/integrations"

// Customer tool names.
const (
	ToolCustomersList   = "shopify_list_customers"
	ToolCustomersGet    = "shopify_get_customer"
	ToolCustomersCreate = "shopify_create_customer"
	ToolCustomersUpdate = "shopify_update_customer"
	ToolCustomersSearch = "shopify_search_customers"
	ToolCustomersCount  = "shopify_count_customers"
	ToolCustomersOrders = "shopify_customer_orders"
)

func customerTools() []integrations.ToolManifest {
	return []integrations.ToolManifest{
		{
			Name:        ToolCustomersList,
			Description: "List customers in the Shopify store. Returns customer name, email, order count, total spent, and tags.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of customers to return (1-250). Default: 50.",
					},
					"page_info": map[string]any{
						"type":        "string",
						"description": "Cursor for pagination.",
					},
					"created_at_min": map[string]any{
						"type":        "string",
						"description": "Show customers created after this date (ISO 8601).",
					},
					"created_at_max": map[string]any{
						"type":        "string",
						"description": "Show customers created before this date (ISO 8601).",
					},
					"updated_at_min": map[string]any{
						"type":        "string",
						"description": "Show customers updated after this date (ISO 8601).",
					},
					"fields": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of fields to include.",
					},
				},
			},
			RequiredScopes: []string{ScopeReadCustomers},
			RateLimit:      20,
		},
		{
			Name:        ToolCustomersGet,
			Description: "Get detailed information about a specific customer by ID. Returns name, email, addresses, order history summary, tags, and metadata.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"customer_id": map[string]any{
						"type":        "string",
						"description": "The Shopify customer ID.",
					},
					"fields": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of fields to include.",
					},
				},
				"required": []string{"customer_id"},
			},
			RequiredScopes: []string{ScopeReadCustomers},
			RateLimit:      20,
		},
		{
			Name:        ToolCustomersCreate,
			Description: "Create a new customer record. Supports name, email, phone, address, tags, and marketing consent.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"first_name": map[string]any{
						"type":        "string",
						"description": "Customer's first name.",
					},
					"last_name": map[string]any{
						"type":        "string",
						"description": "Customer's last name.",
					},
					"email": map[string]any{
						"type":        "string",
						"description": "Customer's email address.",
					},
					"phone": map[string]any{
						"type":        "string",
						"description": "Customer's phone number (E.164 format recommended).",
					},
					"tags": map[string]any{
						"type":        "string",
						"description": "Comma-separated tags for the customer.",
					},
					"note": map[string]any{
						"type":        "string",
						"description": "Internal note about the customer.",
					},
					"accepts_marketing": map[string]any{
						"type":        "boolean",
						"description": "Whether the customer has consented to marketing emails. Default: false.",
					},
					"addresses": map[string]any{
						"type":        "array",
						"description": "Customer addresses.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"address1": map[string]any{"type": "string", "description": "Street address."},
								"address2": map[string]any{"type": "string", "description": "Apartment, suite, etc."},
								"city":     map[string]any{"type": "string", "description": "City."},
								"province": map[string]any{"type": "string", "description": "Province or state."},
								"country":  map[string]any{"type": "string", "description": "Country."},
								"zip":      map[string]any{"type": "string", "description": "Postal/ZIP code."},
								"phone":    map[string]any{"type": "string", "description": "Phone for this address."},
								"default":  map[string]any{"type": "boolean", "description": "Whether this is the default address."},
							},
						},
					},
				},
				"required": []string{"email"},
			},
			RequiredScopes: []string{ScopeWriteCustomers},
			RateLimit:      10,
		},
		{
			Name:        ToolCustomersUpdate,
			Description: "Update an existing customer record. Only the specified fields are modified.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"customer_id": map[string]any{
						"type":        "string",
						"description": "The Shopify customer ID to update.",
					},
					"first_name": map[string]any{
						"type":        "string",
						"description": "Updated first name.",
					},
					"last_name": map[string]any{
						"type":        "string",
						"description": "Updated last name.",
					},
					"email": map[string]any{
						"type":        "string",
						"description": "Updated email address.",
					},
					"phone": map[string]any{
						"type":        "string",
						"description": "Updated phone number.",
					},
					"tags": map[string]any{
						"type":        "string",
						"description": "Updated comma-separated tags.",
					},
					"note": map[string]any{
						"type":        "string",
						"description": "Updated internal note.",
					},
					"accepts_marketing": map[string]any{
						"type":        "boolean",
						"description": "Updated marketing consent.",
					},
				},
				"required": []string{"customer_id"},
			},
			RequiredScopes: []string{ScopeWriteCustomers},
			RateLimit:      10,
		},
		{
			Name:        ToolCustomersSearch,
			Description: "Search for customers by name, email, or other attributes. Uses Shopify's customer search syntax.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query (e.g. 'email:john@example.com' or 'first_name:John').",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results (1-250). Default: 50.",
					},
					"fields": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of fields to include.",
					},
				},
				"required": []string{"query"},
			},
			RequiredScopes: []string{ScopeReadCustomers},
			RateLimit:      10,
		},
		{
			Name:        ToolCustomersCount,
			Description: "Get the total number of customers in the store.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{},
			},
			RequiredScopes: []string{ScopeReadCustomers},
			RateLimit:      20,
		},
		{
			Name:        ToolCustomersOrders,
			Description: "List orders placed by a specific customer. Useful for viewing a customer's purchase history.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"customer_id": map[string]any{
						"type":        "string",
						"description": "The Shopify customer ID.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of orders to return (1-250). Default: 50.",
					},
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"open", "closed", "cancelled", "any"},
						"description": "Filter by order status. Default: any.",
					},
				},
				"required": []string{"customer_id"},
			},
			RequiredScopes: []string{ScopeReadCustomers, ScopeReadOrders},
			RateLimit:      20,
		},
	}
}
