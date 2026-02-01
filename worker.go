package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kvalv/shoplist/broadcast"
	"github.com/kvalv/shoplist/cart"
	"github.com/kvalv/shoplist/cron"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/stores"
	"github.com/kvalv/shoplist/stores/clasohlson"
)

func RunBackgroundWorker(
	ctx context.Context,
	repo cart.Repository,
	bus *broadcast.Broadcast[events.Event],
	cron *cron.Cron,
	log *slog.Logger,
) {
	sub := bus.Subscribe()
	log.Info("Started")

	client := clasohlson.NewClient(clasohlson.CCVest)

	cron.Must("test hvert 1. minutt", "* * * * *", func(ctx context.Context, attempt int) error {
		log.Info("I got triggered yo")
		return nil
	})

	go cron.Run()
	defer cron.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Done")
			return
		case ev := <-sub.Ch:
			switch ev := ev.(type) {
			case events.CartUpdated:
				log.Info("Received event", "type", fmt.Sprintf("%T", ev), "event", ev)
				c, err := repo.Cart(ev.CartID)
				if err != nil {
					log.Error("Failed to get cart", "error", err)
					continue
				}
				if c.TargetStore == stores.ClasOhlson {
					log.Info("Processing cart for Clas Ohlson", "cartID", c.ID)
					for _, ID := range ev.ItemIDs {
						item := c.Get(ID)
						if item == nil {
							log.Error("Item not found in cart", "itemID", ID)
							continue
						}
						if item.Clas != nil {
							continue
						}

						results, err := client.Query(ctx, item.Text, 5)
						if err != nil {
							log.Error("Failed to search items", "error", err)
							continue
						}
						if len(results) == 0 {
							log.Info("No items found", "query", item.Text)
							continue
						}

						log.Info("Found candidates", "count", len(results), "query", item.Text)
						for i, cl := range results {
							locations := ""
							for j, loc := range cl.Locations {
								if j > 0 {
									locations += ", "
								}
								locations += loc.Area + " " + loc.Shelf
							}
							log.Info("Candidate", "rank", i+1, "name", cl.Name, "price", cl.Price, "stock", cl.Stock, "locations", locations)
							log.Debug("Candidate URLs", "url", cl.URL, "picture", cl.Picture)
						}

						chosen := 0
						item.Clas = &cart.ClasSearch{
							Candidates: results,
							Chosen:     &chosen,
						}
						repo.Save(c)
						bus.Publish(events.CartUpdated{
							CartID:  c.ID,
							ItemIDs: []string{item.ID},
						})

					}
				}

			}
		}
	}
}
