package clasohlson

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kvalv/shoplist/llm"
)

type client struct {
	storeID string
}

func NewClient(storeID string) *client {
	return &client{storeID: storeID}
}

var CCVest = "200"

type ShelfLocation struct {
	Area  string
	Shelf string
}

type Item struct {
	ID        string // not persisted to DB, but needed for API calls
	Name      string
	Price     float64
	URL       string
	Picture   string
	Reviews   int // not persisted to DB, but used for ranking
	Stock     int
	Locations []ShelfLocation
}

func (c *client) Search(text string) ([]Item, error) {
	resp, err := http.Get("https://www.clasohlson.com/no/search/getSearchResults?text=" + url.QueryEscape(text))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		CategoryResponseDTOs struct {
			Products []struct {
				GtmID         string  `json:"gtmId"`
				Name          string  `json:"name"`
				CurrentPrice  float64 `json:"currentPrice"`
				URL           string  `json:"url"`
				GridViewImage string  `json:"gridViewImage"`
				Reviews       int     `json:"reviews"`
				InStock       bool    `json:"inStock"`
			} `json:"products"`
		} `json:"categoryResponseDTOs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]Item, len(result.CategoryResponseDTOs.Products))
	for i, p := range result.CategoryResponseDTOs.Products {
		items[i] = Item{
			ID:      p.GtmID,
			Name:    p.Name,
			Price:   p.CurrentPrice,
			URL:     "https://www.clasohlson.com/no" + p.URL,
			Picture: "https://www.clasohlson.com" + p.GridViewImage,
			Reviews: p.Reviews,
		}
	}
	return items, nil
}

func (c *client) Availability(item Item) (Item, error) {
	req, _ := http.NewRequest("GET", "https://www.clasohlson.com/no/cocheckout/getCartDataOnReload?variantProductCode="+item.ID, nil)
	req.Header.Set("Cookie", "COStoreCookie="+c.storeID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return item, err
	}
	defer resp.Body.Close()

	var result struct {
		StoreStockList struct {
			StockData []struct {
				StoreID    string `json:"storeId"`
				StoreStock int    `json:"storeStock"`
				ShelfData  []struct {
					AreaName     string `json:"areaName"`
					ShelfNumbers string `json:"shelfNumbers"`
				} `json:"shelfData"`
			} `json:"stockData"`
		} `json:"storeStockList"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return item, err
	}

	for _, s := range result.StoreStockList.StockData {
		if s.StoreID == c.storeID {
			item.Stock = s.StoreStock
			for _, shelf := range s.ShelfData {
				item.Locations = append(item.Locations, ShelfLocation{
					Area:  shelf.AreaName,
					Shelf: shelf.ShelfNumbers,
				})
			}
			return item, nil
		}
	}
	return item, fmt.Errorf("store %s not found", c.storeID)
}

func (c *client) Query(ctx context.Context, query string, topk int) ([]Item, error) {
	var searchResults []Item
	searchTool := llm.Func("search", "Search for products by name", func(args struct {
		Query string `json:"query" desc:"Search query"`
	}) (map[string]any, error) {
		items, err := c.Search(args.Query)
		if err != nil {
			return nil, err
		}
		searchResults = items
		return map[string]any{"items": items}, nil
	})

	var result struct {
		ProductIDs []string `json:"product_ids"`
	}
	prompt := fmt.Sprintf(`Find and rank the top %d products matching: %s

Rank products by considering:
1. How well the product matches the search intent
2. Number of reviews (more reviews = more popular/commonly bought)
3. Whether it's in stock
4. How likely a typical customer would want this item

Return the product IDs ordered from best to worst match.`, topk, query)

	err := llm.StructuredQuery(ctx, prompt, &result, llm.Options{
		Tools: []llm.Tool{searchTool},
	})
	if err != nil {
		return nil, err
	}

	// Build map for quick lookup
	itemMap := make(map[string]Item)
	for _, item := range searchResults {
		itemMap[item.ID] = item
	}

	// Collect items in ranked order and fetch availability
	var items []Item
	for _, id := range result.ProductIDs {
		if item, ok := itemMap[id]; ok {
			item, err := c.Availability(item)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			if len(items) >= topk {
				break
			}
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no matching products found for %q", query)
	}
	return items, nil
}
