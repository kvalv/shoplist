package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/commands"
	"github.com/kvalv/shoplist/cron"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/views"
	"github.com/starfederation/datastar-go/datastar"
	_ "modernc.org/sqlite"
)

//go:embed migration.sql
var migrationSQL string

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log := logger("main")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		log.Info("received interrupt, shutting down...")
		cancel()
	}()

	if err := run(ctx, log); err != nil {
		log.Error(fmt.Sprintf("application error: %v", err))
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger) error {
	db, err := sql.Open("sqlite", "file:shop.db")
	if err != nil {
		log.Error("failed to open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	repo, err := carts.NewRepository(db)
	if err != nil {
		return fmt.Errorf("failed to create cart repository: %w", err)
	}

	cron := cron.
		New(ctx, cron.BackendSqlite(db)).
		WithLogger(logger("cron")).
		WithPollInterval(time.Minute*30).
		MustRegister("Create new cart on the start of next week", "0 0 * * mon", func(ctx context.Context, attempt int) error {
			// TODO: collaborator...
			cart := carts.New()
			if err := repo.Save(cart); err != nil {
				return fmt.Errorf("failed to create cart: %w", err)
			}
			log.Info("Created new cart", "cartID", cart.ID)
			return nil
		})
	go cron.Run()
	defer cron.Stop()

	// Whenever a cart (item) is updated, we'll broadcast the event, so
	// any client receives a new render.
	bus := events.NewBus(logger("bus"))

	// cart.SetupMockData(repo)

	go RunBackgroundWorker(
		ctx,
		repo,
		bus,
		logger("worker"),
	)

	r := chi.NewRouter().With(
		auth.NewMockAuth(&auth.Claims{
			UserID: "userID123",
			Name:   "Markus Berg Lavby",
			Email:  "kongenbefaler@email.com",
		}),
		auth.RegisterUsers(db, logger("auth"), bus),
	)
	server := http.Server{
		Addr:    ":3001",
		Handler: r,
	}
	go func() {
		<-ctx.Done()
		log.Info("shutting down server...")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("failed to shutdown server", "error", err)
		}
	}()

	// Initial render
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		carts, _ := repo.List(5)
		templ.Handler(views.Page(carts[0], carts)).ServeHTTP(w, r)
	})

	r.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+r.URL.Path)
	})

	// Render loop
	r.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
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

	r.HandleFunc("/add", commands.NewAddItem(repo, bus, log))
	r.HandleFunc("/check", commands.NewCheckItem(repo, bus, log))
	r.HandleFunc("/set-store", commands.NewSetStore(repo, bus, log))
	r.HandleFunc("/switch-cart", commands.NewSwitchCart(repo, bus, log))

	log.Info("starting server", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
	return nil
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

func migrate(db *sql.DB) error {
	for stmt := range strings.SplitSeq(migrationSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute %q: %w", stmt[:min(50, len(stmt))], err)
		}
	}
	return nil
}
