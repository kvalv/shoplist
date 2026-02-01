package main

import (
	"context"
	"log"

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
	log *log.Logger,
) {
	sub := bus.Subscribe()
	log.Printf("Started")

	client := clasohlson.NewClient(clasohlson.CCVest)

	cron.Must("test hvert 1. minutt", "* * * * *", func(ctx context.Context, attempt int) error {
		log.Printf("I got triggered yo")
		return nil
	})

	go cron.Run()
	defer cron.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[worker] Done")
			return
		case ev := <-sub.Ch:
			switch ev := ev.(type) {
			case events.CartUpdated:
				log.Printf("[worker] Received event, %T %+v", ev, ev)
				c, err := repo.Cart(ev.CartID)
				if err != nil {
					log.Printf("[worker] Failed to get cart: %s", err)
					continue
				}
				if c.TargetStore == stores.ClasOhlson {
					log.Printf("[worker] Processing cart %s for Clas Ohlson", c.ID)
					for _, ID := range ev.ItemIDs {
						item := c.Get(ID)
						if item == nil {
							log.Printf("[worker] Item not found in cart: %s", ID)
							continue
						}
						if item.Clas != nil {
							continue
						}

						results, err := client.Query(ctx, item.Text, 5)
						if err != nil {
							log.Printf("[worker] Failed to search items: %s", err)
							continue
						}
						if len(results) == 0 {
							log.Printf("[worker] No items found for '%s'", item.Text)
							continue
						}

						log.Printf("[worker] Found %d candidates for '%s':", len(results), item.Text)
						for i, cl := range results {
							locations := ""
							for j, loc := range cl.Locations {
								if j > 0 {
									locations += ", "
								}
								locations += loc.Area + " " + loc.Shelf
							}
							log.Printf("  [%d] %s | %.2f NOK | %d in stock | %s", i+1, cl.Name, cl.Price, cl.Stock, locations)
							log.Printf("      %s", cl.URL)
							log.Printf("      %s", cl.Picture)
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
