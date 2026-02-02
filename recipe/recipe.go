package recipe

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"strings"

	"github.com/kvalv/reciparse"
	"github.com/kvalv/shoplist/llm"
)

func Parse(ctx context.Context, url *url.URL) ([]string, error) {
	log.Printf("parsing recipe from url: %s", url.String())
	got, err := reciparse.New().ParseRecipe(*url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse recipe: %w", err)
	}

	var s strings.Builder
	if err := tpl.Execute(&s, got.Ingredients); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	var res struct {
		Items []string `json:"items"`
	}
	if err := llm.StructuredQuery(ctx, s.String(), &res); err != nil {
		return nil, fmt.Errorf("failed to query llm: %w", err)
	}

	return res.Items, nil
}

var tpl = template.Must(template.New("").Funcs(template.FuncMap{}).Parse(`
Your task is to figure out what ingredients the user most likely needs to
add to their shopping list.

The recipe is parsed and provided below:

{{ range . }}
- {{ . }}
{{ end }}

Ignore basic ingredients such as salt, water, pepper, and oil.
The language is either going to be in Norwegian or English.
Use the same language. Feel free to rewrite so it's more sensible. E.g.
"2 cloves of garlic" becomes "garlic cloves".

`))
