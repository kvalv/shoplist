package clasohlson

import (
	"context"
	"testing"
)

func TestSearchAndAvailability(t *testing.T) {
	tests := []struct {
		query     string
		wantName  string
		wantID    string
		wantStock int
	}{
		{"skopose", "Skopose med sedertre, 2-pakning", "445689000", 42},
		{"fuglefrø", "Fuglefrø i spann med lokk, 4 kg", "316328000", 8},
	}

	for _, tt := range tests {
		got, err := NewClient(CCVest).Search(tt.query)
		if err != nil {
			t.Fatalf("Search(%q) error = %v", tt.query, err)
		}

		var item Item
		for _, elem := range got {
			if elem.ID == tt.wantID {
				item = elem
				break
			}
		}
		if item.ID == "" {
			t.Fatalf("Search(%q) did not find ID %q", tt.query, tt.wantID)
		}
		if item.URL == "" || item.Price == 0 {
			t.Fatalf("Search(%q) missing URL or Price: %+v", tt.query, item)
		}

		item, err = NewClient(CCVest).Availability(item)
		if err != nil {
			t.Fatalf("Availability(%v) error = %v", item.ID, err)
		}
		if item.Stock != tt.wantStock {
			t.Fatalf("Availability(%v) stock = %d, want %d", item.ID, item.Stock, tt.wantStock)
		}
		if len(item.Locations) == 0 {
			t.Fatalf("Availability(%v) has no locations", item.ID)
		}
		t.Logf("Item: %+v", item)
	}
}

func TestQuery(t *testing.T) {
	items, err := NewClient(CCVest).Query(context.Background(), "skopose", 1)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Query() returned %d items, want 1", len(items))
	}
	item := items[0]
	if item.Stock == 0 {
		t.Fatalf("Query() stock = 0, want non-zero")
	}
	if item.URL == "" || item.Price == 0 || len(item.Locations) == 0 {
		t.Fatalf("Query() missing fields: %+v", item)
	}
	t.Logf("Got: %+v", item)
}
