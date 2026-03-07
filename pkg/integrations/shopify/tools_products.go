package shopify

import "github.com/standardws/operator/pkg/integrations"

// Product tool names.
const (
	ToolProductsList  = "shopify_list_products"
	ToolProductsGet   = "shopify_get_product"
	ToolProductsCreate = "shopify_create_product"
	ToolProductsUpdate = "shopify_update_product"
	ToolProductsDelete = "shopify_delete_product"
	ToolProductsCount  = "shopify_count_products"
	ToolProductsSearch = "shopify_search_products"
)

func productTools() []integrations.ToolManifest {
	return []integrations.ToolManifest{
		{
			Name:        ToolProductsList,
			Description: "List products in the Shopify store. Returns product titles, descriptions, variants, prices, and inventory status.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of products to return (1-250). Default: 50.",
					},
					"page_info": map[string]any{
						"type":        "string",
						"description": "Cursor for pagination. Use the value from the next/previous page link.",
					},
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"active", "archived", "draft"},
						"description": "Filter by product status. Default: all.",
					},
					"product_type": map[string]any{
						"type":        "string",
						"description": "Filter by product type.",
					},
					"vendor": map[string]any{
						"type":        "string",
						"description": "Filter by vendor name.",
					},
					"collection_id": map[string]any{
						"type":        "string",
						"description": "Filter by collection ID.",
					},
					"fields": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of fields to include in the response.",
					},
				},
			},
			RequiredScopes: []string{ScopeReadProducts},
			RateLimit:      20,
		},
		{
			Name:        ToolProductsGet,
			Description: "Get detailed information about a specific product by its ID. Returns title, description, variants (with prices, SKUs, inventory), images, tags, and metadata.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"product_id": map[string]any{
						"type":        "string",
						"description": "The Shopify product ID.",
					},
					"fields": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of fields to include in the response.",
					},
				},
				"required": []string{"product_id"},
			},
			RequiredScopes: []string{ScopeReadProducts},
			RateLimit:      20,
		},
		{
			Name:        ToolProductsCreate,
			Description: "Create a new product in the Shopify store. Supports title, description, variants, pricing, images, and tags.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "The product title.",
					},
					"body_html": map[string]any{
						"type":        "string",
						"description": "Product description in HTML format.",
					},
					"vendor": map[string]any{
						"type":        "string",
						"description": "The product vendor/manufacturer.",
					},
					"product_type": map[string]any{
						"type":        "string",
						"description": "The product type or category.",
					},
					"tags": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of tags.",
					},
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"active", "archived", "draft"},
						"description": "Product status. Default: draft.",
					},
					"variants": map[string]any{
						"type":        "array",
						"description": "Product variants with pricing and inventory.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"price":    map[string]any{"type": "string", "description": "Variant price."},
								"sku":      map[string]any{"type": "string", "description": "SKU code."},
								"title":    map[string]any{"type": "string", "description": "Variant title."},
								"option1":  map[string]any{"type": "string", "description": "First option value (e.g. size)."},
								"option2":  map[string]any{"type": "string", "description": "Second option value (e.g. color)."},
								"option3":  map[string]any{"type": "string", "description": "Third option value."},
								"inventory_quantity": map[string]any{"type": "integer", "description": "Stock quantity."},
							},
						},
					},
				},
				"required": []string{"title"},
			},
			RequiredScopes: []string{ScopeWriteProducts},
			RateLimit:      10,
		},
		{
			Name:        ToolProductsUpdate,
			Description: "Update an existing product. Only the specified fields are modified; omitted fields remain unchanged.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"product_id": map[string]any{
						"type":        "string",
						"description": "The Shopify product ID to update.",
					},
					"title": map[string]any{
						"type":        "string",
						"description": "Updated product title.",
					},
					"body_html": map[string]any{
						"type":        "string",
						"description": "Updated product description in HTML.",
					},
					"vendor": map[string]any{
						"type":        "string",
						"description": "Updated vendor name.",
					},
					"product_type": map[string]any{
						"type":        "string",
						"description": "Updated product type.",
					},
					"tags": map[string]any{
						"type":        "string",
						"description": "Updated comma-separated tags.",
					},
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"active", "archived", "draft"},
						"description": "Updated product status.",
					},
				},
				"required": []string{"product_id"},
			},
			RequiredScopes: []string{ScopeWriteProducts},
			RateLimit:      10,
		},
		{
			Name:        ToolProductsDelete,
			Description: "Permanently delete a product from the Shopify store. This action cannot be undone.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"product_id": map[string]any{
						"type":        "string",
						"description": "The Shopify product ID to delete.",
					},
				},
				"required": []string{"product_id"},
			},
			RequiredScopes: []string{ScopeWriteProducts},
			RateLimit:      10,
		},
		{
			Name:        ToolProductsCount,
			Description: "Get the total number of products in the store, optionally filtered by status, vendor, or product type.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"active", "archived", "draft"},
						"description": "Filter count by product status.",
					},
					"vendor": map[string]any{
						"type":        "string",
						"description": "Filter count by vendor.",
					},
					"product_type": map[string]any{
						"type":        "string",
						"description": "Filter count by product type.",
					},
				},
			},
			RequiredScopes: []string{ScopeReadProducts},
			RateLimit:      20,
		},
		{
			Name:        ToolProductsSearch,
			Description: "Search for products by title, vendor, product type, or tag. Uses Shopify's search syntax.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query. Supports Shopify search syntax (e.g. 'title:shoes AND vendor:nike').",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results (1-250). Default: 50.",
					},
				},
				"required": []string{"query"},
			},
			RequiredScopes: []string{ScopeReadProducts},
			RateLimit:      10,
		},
	}
}
