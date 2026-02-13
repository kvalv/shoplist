package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/llm"
	"github.com/kvalv/shoplist/stores"
	"github.com/kvalv/shoplist/stores/clasohlson"
)

func RunBackgroundWorker(
	ctx context.Context,
	repo *carts.SqliteRepository,
	bus *events.Bus,
	log *slog.Logger,
) {
	sub := bus.Subscribe()
	log.Info("Started")

	client := clasohlson.NewClient(clasohlson.CCVest)

	for {
		select {
		case <-ctx.Done():
			log.Info("Done")
			return
		case ev := <-sub.Ch:
			switch ev := ev.(type) {
			case events.UserRegistered:
				log.Info("User registered", "userID", ev.UserID)
				repo.Save(carts.New().
					WithName("Min første handleliste").
					WithCreator(ev.UserID),
				)

			case events.ChatAdded:
				log.Info("ChatAdded", "cartID", ev.CartID, "messageID", ev.MessageID)
				msg, err := repo.Message(ev.MessageID)
				if err != nil {
					log.Error("Failed to get message", "error", err)
					continue
				}
				if !strings.HasPrefix(msg.Text, "@assistant:") {
					continue
				}
				prompt := strings.TrimSpace(strings.TrimPrefix(msg.Text, "@assistant:"))
				var resp struct {
					Answer string `json:"answer" desc:"a helpful answer to the user's question"`
				}
				if err := llm.StructuredQuery(ctx, prompt, &resp); err != nil {
					log.Error("LLM query failed", "error", err)
					continue
				}
				reply := carts.NewMessage(ev.CartID, resp.Answer).WithAssistant()
				if err := repo.AddMessage(reply); err != nil {
					log.Error("Failed to add assistant reply", "error", err)
					continue
				}
				bus.Publish(events.CartUpdated{CartID: ev.CartID})

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

						item.Clas = &carts.ClasSearch{
							Candidates: results,
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
