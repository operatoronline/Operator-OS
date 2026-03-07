package shopify

import "github.com/standardws/operator/pkg/integrations"

// Order tool names.
const (
	ToolOrdersList   = "shopify_list_orders"
	ToolOrdersGet    = "shopify_get_order"
	ToolOrdersCount  = "shopify_count_orders"
	ToolOrdersClose  = "shopify_close_order"
	ToolOrdersCancel = "shopify_cancel_order"
	ToolOrdersCreate = "shopify_create_order"
	ToolOrdersUpdate = "shopify_update_order"
)

func orderTools() []integrations.ToolManifest {
	return []integrations.ToolManifest{
		{
			Name:        ToolOrdersList,
			Description: "List orders from the Shopify store. Returns order number, customer info, line items, totals, and fulfillment status.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of orders to return (1-250). Default: 50.",
					},
					"page_info": map[string]any{
						"type":        "string",
						"description": "Cursor for pagination.",
					},
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"open", "closed", "cancelled", "any"},
						"description": "Filter by order status. Default: open.",
					},
					"financial_status": map[string]any{
						"type":        "string",
						"enum":        []string{"authorized", "pending", "paid", "partially_paid", "refunded", "voided", "partially_refunded", "any", "unpaid"},
						"description": "Filter by financial status.",
					},
					"fulfillment_status": map[string]any{
						"type":        "string",
						"enum":        []string{"shipped", "partial", "unshipped", "any", "unfulfilled"},
						"description": "Filter by fulfillment status.",
					},
					"created_at_min": map[string]any{
						"type":        "string",
						"description": "Show orders created after this date (ISO 8601 format).",
					},
					"created_at_max": map[string]any{
						"type":        "string",
						"description": "Show orders created before this date (ISO 8601 format).",
					},
					"fields": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of fields to include.",
					},
				},
			},
			RequiredScopes: []string{ScopeReadOrders},
			RateLimit:      20,
		},
		{
			Name:        ToolOrdersGet,
			Description: "Get detailed information about a specific order by its ID. Returns complete order data including line items, customer, shipping, payment, and fulfillment details.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The Shopify order ID.",
					},
					"fields": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of fields to include.",
					},
				},
				"required": []string{"order_id"},
			},
			RequiredScopes: []string{ScopeReadOrders},
			RateLimit:      20,
		},
		{
			Name:        ToolOrdersCount,
			Description: "Get the total number of orders, optionally filtered by status, financial status, or date range.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"open", "closed", "cancelled", "any"},
						"description": "Filter count by order status.",
					},
					"financial_status": map[string]any{
						"type":        "string",
						"enum":        []string{"authorized", "pending", "paid", "partially_paid", "refunded", "voided", "partially_refunded", "any", "unpaid"},
						"description": "Filter count by financial status.",
					},
					"created_at_min": map[string]any{
						"type":        "string",
						"description": "Count orders created after this date (ISO 8601).",
					},
					"created_at_max": map[string]any{
						"type":        "string",
						"description": "Count orders created before this date (ISO 8601).",
					},
				},
			},
			RequiredScopes: []string{ScopeReadOrders},
			RateLimit:      20,
		},
		{
			Name:        ToolOrdersClose,
			Description: "Close an open order. Closed orders cannot be edited but can be reopened.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The Shopify order ID to close.",
					},
				},
				"required": []string{"order_id"},
			},
			RequiredScopes: []string{ScopeWriteOrders},
			RateLimit:      10,
		},
		{
			Name:        ToolOrdersCancel,
			Description: "Cancel an order. Optionally specify a reason and whether to restock items or send a notification email.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The Shopify order ID to cancel.",
					},
					"reason": map[string]any{
						"type":        "string",
						"enum":        []string{"customer", "fraud", "inventory", "declined", "other"},
						"description": "The reason for cancellation.",
					},
					"restock": map[string]any{
						"type":        "boolean",
						"description": "Whether to restock the line items. Default: false.",
					},
					"email": map[string]any{
						"type":        "boolean",
						"description": "Whether to send a cancellation email to the customer. Default: false.",
					},
				},
				"required": []string{"order_id"},
			},
			RequiredScopes: []string{ScopeWriteOrders},
			RateLimit:      10,
		},
		{
			Name:        ToolOrdersCreate,
			Description: "Create a draft order or order in the Shopify store. Supports line items, customer association, shipping address, and notes.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"line_items": map[string]any{
						"type":        "array",
						"description": "Order line items.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"variant_id": map[string]any{"type": "string", "description": "Product variant ID."},
								"quantity":   map[string]any{"type": "integer", "description": "Quantity to order."},
								"title":      map[string]any{"type": "string", "description": "Line item title (for custom items)."},
								"price":      map[string]any{"type": "string", "description": "Price per unit (for custom items)."},
							},
						},
					},
					"customer_id": map[string]any{
						"type":        "string",
						"description": "Associate the order with an existing customer ID.",
					},
					"email": map[string]any{
						"type":        "string",
						"description": "Customer email address.",
					},
					"note": map[string]any{
						"type":        "string",
						"description": "Optional note on the order.",
					},
					"tags": map[string]any{
						"type":        "string",
						"description": "Comma-separated tags for the order.",
					},
					"financial_status": map[string]any{
						"type":        "string",
						"enum":        []string{"pending", "authorized", "paid"},
						"description": "Financial status. Default: pending.",
					},
				},
				"required": []string{"line_items"},
			},
			RequiredScopes: []string{ScopeWriteOrders},
			RateLimit:      10,
		},
		{
			Name:        ToolOrdersUpdate,
			Description: "Update an existing order. Only the specified fields are modified. Supports updating note, tags, email, and shipping address.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"order_id": map[string]any{
						"type":        "string",
						"description": "The Shopify order ID to update.",
					},
					"note": map[string]any{
						"type":        "string",
						"description": "Updated order note.",
					},
					"tags": map[string]any{
						"type":        "string",
						"description": "Updated comma-separated tags.",
					},
					"email": map[string]any{
						"type":        "string",
						"description": "Updated customer email.",
					},
				},
				"required": []string{"order_id"},
			},
			RequiredScopes: []string{ScopeWriteOrders},
			RateLimit:      10,
		},
	}
}
