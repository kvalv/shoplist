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
	"github.com/go-chi/chi/v5"
	"github.com/lmittmann/tint"
	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/commands"
	"github.com/kvalv/shoplist/cron"
	"github.com/kvalv/shoplist/devtools"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/migrations"
	"github.com/kvalv/shoplist/views"
	"github.com/starfederation/datastar-go/datastar"
	_ "modernc.org/sqlite"
)

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

	if err := migrations.Migrate(db); err != nil {
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
			cart := carts.New()
			if err := repo.Save(cart); err != nil {
				return fmt.Errorf("failed to create cart: %w", err)
			}
			// repo.AddCollaborators(cart.ID, "meg", "deg")

			log.Info("Created new cart", "cartID", cart.ID)
			return nil
		})
	go cron.Run()
	defer cron.Stop()

	// Whenever a cart (item) is updated, we'll broadcast the event, so
	// any client receives a new render.
	bus := events.NewBus(logger("bus"))

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

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var curr *carts.Cart
		carts, err := repo.List(5)
		if err != nil {
			log.Error("failed to fetch latest cart", "error", err)
		}
		if len(carts) > 0 {
			curr = carts[0]
		}
		log.Info("redirecting to latest cart", "cartID", curr.ID)
		// redirect to latest
		w.Header().Set("Location", "/"+curr.ID)
		w.WriteHeader(http.StatusFound)
	})

	// Initial render
	r.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		cart, err := repo.Cart(chi.URLParam(r, "id"))
		if err != nil {
			// favicon.ico
			log.Error("failed to fetch cart", "error", err, "id", chi.URLParam(r, "id"))
			return
			panic(fmt.Errorf("failed to fetch cart: %w id=%q", err, chi.URLParam(r, "id")))
		}
		templ.Handler(views.Page(cart, nil)).ServeHTTP(w, r)
	})

	r.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+r.URL.Path)
	})
	r.HandleFunc("/static/styles.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/styles.css")
	})

	// Render loop
	r.HandleFunc("/render", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		sub := bus.Subscribe()
		defer sub.Close()

		// send initial render
		var first *carts.Cart
		carts, _ := repo.List(5)
		if len(carts) > 0 {
			first = carts[0]
		}
		sse.PatchElementTempl(views.Page(first, carts))

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

	go func() {
		log.Info("watching styles.css for changes...")
		for range devtools.WatchFile(ctx, "static/styles.css") {
			log.Info("styles.css changed, broadcasting render event")
		}
	}()

	r.HandleFunc("/add", commands.NewAddItem(repo, bus, log))
	r.HandleFunc("/check", commands.NewCheckItem(repo, bus, log))
	r.HandleFunc("/set-name", commands.NewSetName(repo, bus, log))
	r.HandleFunc("/set-store", commands.NewSetStore(repo, bus, log))
	r.HandleFunc("/switch-cart", commands.NewSwitchCart(repo, bus, log))
	r.HandleFunc("/select-clas-item", commands.NewSelectClasItem(repo, bus, log))

	log.Info("starting server", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
	return nil
}

func logger(prefix string) *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: "15:04:05",
	})).With("srv", prefix)
}
