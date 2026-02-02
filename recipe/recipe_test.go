package recipe

import (
	"net/url"
	"testing"

	"github.com/kvalv/reciparse"
)

func TestRecipe(t *testing.T) {
	uri, _ := url.Parse("https://www.matprat.no/oppskrifter/tradisjon/finnbiff/")
	got, err := reciparse.New().ParseRecipe(*uri)
	if err != nil {
		t.Fatalf("failed: %s", err)
	}
	for _, ing := range got.Ingredients {
		t.Logf("got %s", ing)
	}
}
