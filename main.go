package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/a-h/templ"
	"github.com/kvalv/shoplist/broadcast"
	"github.com/kvalv/shoplist/cart"
	"github.com/kvalv/shoplist/store"
	"github.com/starfederation/datastar-go/datastar"
)

func main() {

	repo := cart.NewMock()
	SetupMockData(repo)

	// Whenever a cart (item) is updated, we'll broadcast the event, so
	// any client receives a new render.
	bus := broadcast.New[Event]()

	// Initial render
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cart, _ := repo.Latest()
		templ.Handler(page(cart)).ServeHTTP(w, r)
	})

	// Render loop
	http.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		sub := bus.Subscribe()
		defer sub.Close()

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

		bus.Publish(CartUpdated{CartID: cart.ID})

		sse.PatchSignals([]byte(`{"text": ""}`))
	})

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		ID := r.URL.Query().Get("id")
		cart, _ := repo.Latest()
		cart.Get(ID).Toggle()
		repo.Save(cart)

		bus.Publish(CartUpdated{CartID: cart.ID})

		log.Printf("tick called")
	})

	http.HandleFunc("/set-store", func(w http.ResponseWriter, r *http.Request) {
		cart, _ := repo.Latest()
		var signals struct {
			Store string `json:"store"`
		}
		if err := datastar.ReadSignals(r, &signals); err != nil {
			log.Printf("failed to read signals: %s", err)
			return
		}

		var err error
		if cart.TargetStore, err = parseStore(signals.Store); err != nil {
			log.Printf("failed to parse: %s", err)
			return
		}
		repo.Save(cart)

		log.Printf("/store called -- set to %d", cart.TargetStore)
		bus.Publish(CartUpdated{CartID: cart.ID})
	})

	log.Printf("listening on :3001...")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatal(err)
	}

}

func parseStore(s string) (store.Store, error) {
	got, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return store.Store(got), nil
}
