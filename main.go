package main

import (
	"context"
	"log"
	"net/http"
	"net/url"

	"github.com/a-h/templ"
	"github.com/kvalv/shoplist/broadcast"
	"github.com/kvalv/shoplist/cart"
	"github.com/starfederation/datastar-go/datastar"
)

func main() {

	repo := cart.NewMock()
	SetupMockData(repo)

	// Whenever a cart (item) is updated, we'll broadcast the event, so
	// any client receives a new render.
	br := broadcast.New()

	// Initial render
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cart, _ := repo.Latest()
		templ.Handler(page(cart)).ServeHTTP(w, r)
	})

	// Render loop
	http.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		sub := br.Subscribe()
		defer sub.Close()
		sse := datastar.NewSSE(w, r)

		// send initial render
		cart, _ := repo.Latest()
		sse.PatchElementTempl(page(cart))

		done := r.Context().Done()
		for {
			select {
			case <-done:
				return
			case <-sub.Ch:
				log.Printf("got event, rendering new page")
				cart, _ := repo.Latest()
				sse.PatchElementTempl(page(cart))
			}
		}
	})

	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		var signals struct {
			Text string `json:"text"`
		}
		if err := datastar.ReadSignals(r, &signals); err != nil {
			log.Printf("failed to read signals")
		}
		log.Printf("/add invoked %q", signals.Text)

		cart, _ := repo.Latest()

		// we'll check if it's an url...
		if got, _ := url.ParseRequestURI(signals.Text); got != nil {
			log.Printf("this is a recipe, trying to parse")
			// ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
			// defer cancel()
			parts, err := ParseRecipe(context.Background(), got)
			if err != nil {
				log.Printf("failed to parse recipe: %v", err)
			}
			log.Printf("parsed recipe into %d parts", len(parts))
			for _, item := range parts {
				log.Printf("adding item from recipe: %s", item)
				cart.Add(item)
			}
		} else {
			cart.Add(signals.Text)
		}
		repo.Save(cart)
		sse := datastar.NewSSE(w, r)
		br.Publish()

		sse.PatchSignals([]byte(`{"text": ""}`))
	})

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		ID := r.URL.Query().Get("id")
		cart, _ := repo.Latest()
		cart.Get(ID).Toggle()
		repo.Save(cart)
		br.Publish()

		log.Printf("tick called")
	})

	log.Printf("listening on :3001...")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatal(err)
	}

}

// fucn
