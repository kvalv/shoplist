package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/a-h/templ"
	"github.com/kvalv/shoplist/cart"
	"github.com/kvalv/shoplist/commands"
	"github.com/kvalv/shoplist/cron"
	"github.com/kvalv/shoplist/events"
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

	ch := make(chan os.Signal, 1)
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

	repo, err := cart.NewRepository(db)
	if err != nil {
		log.Error("failed to create repo", "error", err)
		os.Exit(1)
	}

	cron := cron.New(ctx, cron.BackendSqlite(db)).
		WithLogger(logger("cron")).
		WithPollInterval(time.Minute * 30)
	defer cron.Stop()

	// Whenever a cart (item) is updated, we'll broadcast the event, so
	// any client receives a new render.
	bus := events.NewBus(logger("bus"))

	cart.SetupMockData(repo)

	go RunBackgroundWorker(
		ctx,
		repo,
		bus,
		cron,
		logger("worker"),
	)

	// Initial render
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		carts, _ := repo.List(5)
		templ.Handler(views.Page(carts[0], carts)).ServeHTTP(w, r)
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
		carts, _ := repo.List(5)
		if len(carts) == 0 {
			panic("no latest cart")
		}
		sse.PatchElementTempl(views.Page(carts[0], carts))

		done := r.Context().Done()
		for {
			select {
			case <-done:
				return
			case event := <-sub.Ch:
				carts, _ := repo.List(5)
				log.Info("render fat morph",
					"event", fmt.Sprintf("%T", event),
					"cartID", carts[0].ID,
				)
				sse.PatchElementTempl(views.Page(carts[0], carts))
			}
		}
	})

	http.HandleFunc("/add", commands.NewAddItem(repo, bus, log))
	http.HandleFunc("/check", commands.NewCheckItem(repo, bus, log))
	http.HandleFunc("/set-store", commands.NewSetStore(repo, bus, log))
	http.HandleFunc("/switch-cart", commands.NewSwitchCart(repo, bus, log))

	log.Info("starting server", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
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
