package main

import "testing"

func TestClasOhlson(t *testing.T) {
	tests := []struct {
		query        string
		wantName     string
		wantID       string
		wantAvail    int
	}{
		{"skopose", "Skopose med sedertre  2-pakning", "445689000", 42},
		{"fuglefrø", "Fuglefrø i spann med lokk  4 kg", "316328000", 8},
	}

	client := ClasOhlson{StoreID: "200"}
	for _, tt := range tests {
		got, err := client.Search(tt.query)
		if err != nil {
			t.Fatalf("Search(%q) error = %v", tt.query, err)
		}

		var item ClasOhlsonItem
		for _, elem := range got {
			if elem.Text == tt.wantName && elem.ID == tt.wantID {
				item = elem
				break
			}
		}
		if item.ID == "" {
			t.Fatalf("Search(%q) did not find %q", tt.query, tt.wantName)
		}

		avail, err := client.Availability(item)
		if err != nil {
			t.Fatalf("Availability(%v) error = %v", item, err)
		}
		if avail != tt.wantAvail {
			t.Fatalf("Availability(%v) = %d, want %d", item, avail, tt.wantAvail)
		}
	}
}
