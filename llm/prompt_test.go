package llm

import "testing"

func TestQuery(t *testing.T) {
	var res struct {
		Answer int `json:"answer"`
	}
	if err := StructuredQuery(t.Context(), "what is 2+2", &res); err != nil {
		t.Fatalf("StructuredQuery failed: %v", err)
	}

	if res.Answer != 4 {
		t.Fatalf("expected answer to be 4, got %d", res.Answer)
	}
}
