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
	"github.com/kvalv/shoplist/auth"
	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/commands"
	"github.com/kvalv/shoplist/cron"
	"github.com/kvalv/shoplist/devtools"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/logger"
	"github.com/kvalv/shoplist/migrations"
	"github.com/kvalv/shoplist/views"
	"github.com/lmittmann/tint"
	"github.com/starfederation/datastar-go/datastar"
	_ "modernc.org/sqlite"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log := baseLogger("main")
	slog.SetDefault(log)

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
		WithLogger(baseLogger("cron")).
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
	bus := events.NewBus(baseLogger("bus"))

	go RunBackgroundWorker(
		ctx,
		repo,
		bus,
		baseLogger("worker"),
	)

	r := chi.NewRouter().With(
		logger.Middleware(log),
		auth.NewMockAuth(&auth.Claims{
			UserID: "userID123",
			Name:   "Markus Berg Lavby",
			Email:  "kongenbefaler@email.com",
		}),
		auth.RegisterUsers(db, baseLogger("auth"), bus),
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

	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	r.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/"+chi.URLParam(r, "id")+"/list", http.StatusFound)
	})
	r.HandleFunc("/{id}/{mode}", func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromRequest(r)
		mode := chi.URLParam(r, "mode")
		cart, err := repo.Cart(chi.URLParam(r, "id"))
		if err != nil {
			log.Error("failed to fetch cart", "error", err, "id", chi.URLParam(r, "id"))
			return
		}
		switch mode {
		case "chat":
			bus.Publish(events.ChatOpened{CartID: cart.ID, UserID: claims.UserID})
		case "list", "shop":
		default:
			http.NotFound(w, r)
			return
		}
		msgs, _ := repo.Messages(cart.ID)
		templ.Handler(views.Cart(cart, nil, msgs, mode)).ServeHTTP(w, r)
	})

	r.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+r.URL.Path)
	})
	r.HandleFunc("/static/styles.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/styles.css")
	})

	// Render loop (SSE)
	r.HandleFunc("/render/{id}/{mode}", func(w http.ResponseWriter, r *http.Request) {
		cartID := chi.URLParam(r, "id")
		mode := chi.URLParam(r, "mode")

		log.Info("SSE /render connected", "cartID", cartID, "mode", mode)

		sse := datastar.NewSSE(w, r)
		sub := bus.Subscribe()
		defer sub.Close()

		done := r.Context().Done()
		for {
			select {
			case <-done:
				log.Info("SSE /render disconnected", "cartID", cartID, "mode", mode)
				return
			case event := <-sub.Ch:
				log.Info("SSE sending update to frontend",
					"event", fmt.Sprintf("%T", event),
					"cartID", cartID,
					"mode", mode,
				)
				active, _ := repo.Cart(cartID)
				if active == nil {
					continue
				}
				msgs, _ := repo.Messages(active.ID)
				sse.PatchElementTempl(views.Cart(active, nil, msgs, mode))
			}
		}
	})

	go func() {
		log.Info("watching styles.css for changes...")
		for range devtools.WatchFile(ctx, "static/styles.css") {
			log.Info("styles.css changed, broadcasting render event")
		}
	}()

	r.HandleFunc("/drawer", func(w http.ResponseWriter, r *http.Request) {
		cartList, _ := repo.List(5)
		activeID := r.URL.Query().Get("active")
		templ.Handler(views.CartDrawerPage(activeID, cartList)).ServeHTTP(w, r)
	})

	r.HandleFunc("/new-cart", func(w http.ResponseWriter, r *http.Request) {
		claims := auth.ClaimsFromRequest(r)
		name := time.Now().Format("2 January")
		cart := carts.New().WithName(name).WithCreator(claims.UserID)
		if err := repo.Save(cart); err != nil {
			log.Error("failed to save cart", "error", err)
			http.Error(w, "failed to create cart", http.StatusInternalServerError)
			return
		}
		bus.Publish(events.CartCreated{CartID: cart.ID})
		http.Redirect(w, r, "/"+cart.ID, http.StatusSeeOther)
	})

	r.HandleFunc("/add", commands.NewAddItem(repo, bus))
	r.HandleFunc("/check", commands.NewCheckItem(repo, bus))
	r.HandleFunc("/set-name", commands.NewSetName(repo, bus))
	r.HandleFunc("/set-store", commands.NewSetStore(repo, bus))
	r.HandleFunc("/switch-cart", commands.NewSwitchCart(repo, bus))
	r.HandleFunc("/select-clas-item", commands.NewSelectClasItem(repo, bus))
	r.HandleFunc("/delete", commands.NewDeleteItem(repo, bus))
	r.HandleFunc("/not-found", commands.NewNotFound(repo, bus))
	r.HandleFunc("/discard/{id}/{reason}", commands.NewDiscardItem(repo, bus))

	log.Info("starting server", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
	return nil
}

func baseLogger(prefix string) *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: "15:04:05",
	})).With("srv", prefix)
}
