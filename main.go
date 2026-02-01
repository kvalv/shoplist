package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"

	"github.com/a-h/templ"
	"github.com/kvalv/shoplist/broadcast"
	"github.com/kvalv/shoplist/cart"
	"github.com/kvalv/shoplist/cron"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/stores"
	"github.com/kvalv/shoplist/views"
	"github.com/starfederation/datastar-go/datastar"
	_ "modernc.org/sqlite"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log := logger("[main] ")

	server := http.Server{
		Addr: ":3001",
	}

	ch := make(chan os.Signal, 10)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		log.Printf("received interrupt, shutting down...")
		cancel()
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown server: %s", err)
		}
	}()

	db, err := sql.Open("sqlite", "file:shop.db")
	if err != nil {
		log.Fatalf("failed to open db: %s", err)
	}
	defer db.Close()

	repo, err := cart.NewSqlite(db)
	if err != nil {
		log.Fatalf("failed to create repo: %s", err)
	}
	cron := cron.New(ctx, cron.BackendSqlite(db)).WithLogger(logger("[cron] "))
	defer cron.Stop()

	cart.SetupMockData(repo)

	// Whenever a cart (item) is updated, we'll broadcast the event, so
	// any client receives a new render.
	bus := broadcast.New[events.Event]()

	go RunBackgroundWorker(
		ctx,
		repo,
		bus,
		cron,
		logger("[worker] "),
	)

	// Initial render
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cart, _ := repo.Latest()
		templ.Handler(views.Page(cart)).ServeHTTP(w, r)
	})

	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+r.URL.Path)
	})

	// Render loop
	http.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		sub := bus.Subscribe()
		defer sub.Close()

		// send initial render
		cart, _ := repo.Latest()
		if cart == nil {
			panic("no latest cart")
		}
		sse.PatchElementTempl(views.Page(cart))

		done := r.Context().Done()
		for {
			select {
			case <-done:
				return
			case <-sub.Ch:
				log.Printf("got event, rendering new page")
				cart, _ := repo.Latest()
				sse.PatchElementTempl(views.Page(cart))
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

		event := events.CartUpdated{
			CartID: cart.ID,
		}
		if got, _ := url.ParseRequestURI(signals.Text); got != nil {
			log.Printf("this is a recipe, trying to parse")
			parts, err := ParseRecipe(context.Background(), got)
			if err != nil {
				log.Printf("failed to parse recipe: %v", err)
			}
			log.Printf("parsed recipe into %d parts", len(parts))
			for _, text := range parts {
				log.Printf("adding item from recipe: %s", text)
				item := cart.Add(text)
				event.ItemIDs = append(event.ItemIDs, item.ID)
			}
		} else {
			item := cart.Add(signals.Text)
			event.ItemIDs = append(event.ItemIDs, item.ID)
		}
		repo.Save(cart)
		bus.Publish(event)
		datastar.NewSSE(w, r).PatchSignals([]byte(`{"text": ""}`))
	})

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		ID := r.URL.Query().Get("id")
		cart, _ := repo.Latest()
		cart.Get(ID).Toggle()
		repo.Save(cart)

		bus.Publish(events.CartUpdated{CartID: cart.ID})

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
		bus.Publish(events.CartUpdated{CartID: cart.ID})
	})

	log.Printf("starting server on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func parseStore(s string) (stores.Store, error) {
	got, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return stores.Store(got), nil
}

func logger(prefix string) *log.Logger {
	return log.New(os.Stdout, prefix, log.Ltime)
}
