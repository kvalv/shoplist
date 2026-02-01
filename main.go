package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/a-h/templ"
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
	log := logger("main")

	server := http.Server{
		Addr: ":3001",
	}

	ch := make(chan os.Signal, 10)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		log.Info("received interrupt, shutting down...")
		cancel()
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("failed to shutdown server", "error", err)
		}
	}()

	db, err := sql.Open("sqlite", "file:shop.db")
	if err != nil {
		log.Error("failed to open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repo, err := cart.NewSqlite(db)
	if err != nil {
		log.Error("failed to create repo", "error", err)
		os.Exit(1)
	}
	cron := cron.New(ctx, cron.BackendSqlite(db)).
		WithLogger(logger("cron")).
		WithPollInterval(time.Minute * 30)
	defer cron.Stop()

	cart.SetupMockData(repo)

	// Whenever a cart (item) is updated, we'll broadcast the event, so
	// any client receives a new render.
	bus := events.NewBus[events.Event](logger("bus"))

	go RunBackgroundWorker(
		ctx,
		repo,
		bus,
		cron,
		logger("worker"),
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
				log.Info("got event, rendering new page")
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
			log.Error("failed to read signals", "error", err)
		}
		log.Info("/add invoked", "text", signals.Text)

		cart, _ := repo.Latest()

		event := events.CartUpdated{
			CartID: cart.ID,
		}
		if got, _ := url.ParseRequestURI(signals.Text); got != nil {
			log.Info("this is a recipe, trying to parse")
			parts, err := ParseRecipe(context.Background(), got)
			if err != nil {
				log.Error("failed to parse recipe", "error", err)
			}
			log.Info("parsed recipe", "parts", len(parts))
			for _, text := range parts {
				log.Info("adding item from recipe", "text", text)
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

		log.Info("tick called")
	})

	http.HandleFunc("/set-store", func(w http.ResponseWriter, r *http.Request) {
		cart, _ := repo.Latest()
		var signals struct {
			Store string `json:"store"`
		}
		if err := datastar.ReadSignals(r, &signals); err != nil {
			log.Error("failed to read signals", "error", err)
			return
		}

		var err error
		if cart.TargetStore, err = parseStore(signals.Store); err != nil {
			log.Error("failed to parse", "error", err)
			return
		}
		repo.Save(cart)

		log.Info("/store called", "store", cart.TargetStore)
		bus.Publish(events.CartUpdated{CartID: cart.ID})
	})

	log.Info("starting server", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
}

func parseStore(s string) (stores.Store, error) {
	got, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return stores.Store(got), nil
}

func logger(prefix string) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format("15:04:05"))
			}
			if a.Key == slog.LevelKey && a.Value.Any() == slog.LevelInfo {
				return slog.Attr{}
			}
			return a
		},
	})).With("srv", prefix)
}
