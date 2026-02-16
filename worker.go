package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/url"
	"strings"

	"github.com/kvalv/shoplist/carts"
	"github.com/kvalv/shoplist/events"
	"github.com/kvalv/shoplist/llm"
	"github.com/kvalv/shoplist/recipe"
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
			case events.ChatOpened:
				log.Info("ChatOpened", "cartID", ev.CartID, "userID", ev.UserID)
				if err := repo.UpdateChatSeenAt(ev.CartID, ev.UserID); err != nil {
					log.Error("Failed to update chat_seen_at", "error", err)
				}

			case events.UserRegistered:
				log.Info("User registered", "userID", ev.UserID)
				repo.Save(carts.New().
					WithName("Min første handleliste").
					WithCreator(ev.UserID),
				)

			case events.ChatAdded:
				log.Info("ChatAdded", "cartID", ev.CartID, "messageID", ev.MessageID)
				cart, err := repo.Cart(ev.CartID)
				if err != nil {
					log.Error("Failed to get cart", "error", err)
					continue
				}

				messages, _ := repo.Messages(ev.CartID)
				var prompt strings.Builder
				if err := chatPromptTpl.Execute(&prompt, messages); err != nil {
					log.Error("Failed to render chat prompt", "error", err)
					continue
				}

				tools := chatTools(ctx, repo, bus, cart, log)
				var resp struct {
					Answer string `json:"answer" desc:"your reply to the user. Leave empty if the message is not addressed to you."`
				}
				if err := llm.StructuredQuery(ctx, prompt.String(), &resp, llm.Options{Tools: tools}); err != nil {
					log.Error("LLM query failed", "error", err)
					continue
				}
				if resp.Answer == "" {
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

func chatTools(
	ctx context.Context,
	repo *carts.SqliteRepository,
	bus *events.Bus,
	cart *carts.Cart,
	log *slog.Logger,
) []llm.Tool {
	return []llm.Tool{
		llm.Func("add_item", "Add an item to the shopping list", func(args struct {
			Text string `json:"text" desc:"the item to add, e.g. 'milk' or 'bread'"`
		}) (string, error) {
			item := cart.Add(args.Text, "assistant")
			if err := repo.Save(cart); err != nil {
				return "", err
			}
			log.Info("Assistant added item", "text", args.Text, "itemID", item.ID)
			bus.Publish(events.CartUpdated{CartID: cart.ID, ItemIDs: []string{item.ID}})
			return fmt.Sprintf("added %q", args.Text), nil
		}),

		llm.Func("remove_item", "Remove an item from the shopping list by name", func(args struct {
			Text string `json:"text" desc:"the item name to remove"`
		}) (string, error) {
			lower := strings.ToLower(args.Text)
			for _, item := range cart.Items {
				if strings.ToLower(item.Text) == lower {
					if err := repo.DeleteItem(item.ID); err != nil {
						return "", err
					}
					log.Info("Assistant removed item", "text", item.Text, "itemID", item.ID)
					bus.Publish(events.CartUpdated{CartID: cart.ID})
					return fmt.Sprintf("removed %q", item.Text), nil
				}
			}
			return fmt.Sprintf("item %q not found in the list", args.Text), nil
		}),

		llm.Func("rename_item", "Rename an item on the shopping list", func(args struct {
			OldName string `json:"old_name" desc:"the current item name"`
			NewName string `json:"new_name" desc:"the new name for the item"`
		}) (string, error) {
			lower := strings.ToLower(args.OldName)
			for _, item := range cart.Items {
				if strings.ToLower(item.Text) == lower {
					old := item.Text
					item.Text = args.NewName
					if err := repo.Save(cart); err != nil {
						return "", err
					}
					log.Info("Assistant renamed item", "old", old, "new", args.NewName)
					bus.Publish(events.CartUpdated{CartID: cart.ID})
					return fmt.Sprintf("renamed %q to %q", old, args.NewName), nil
				}
			}
			return fmt.Sprintf("item %q not found in the list", args.OldName), nil
		}),

		llm.Func("toggle_item", "Check or uncheck an item on the shopping list by name", func(args struct {
			Text string `json:"text" desc:"the item name to toggle"`
		}) (string, error) {
			lower := strings.ToLower(args.Text)
			for _, item := range cart.Items {
				if strings.ToLower(item.Text) == lower {
					item.Toggle("assistant")
					if err := repo.Save(cart); err != nil {
						return "", err
					}
					status := "checked"
					if !item.Checked {
						status = "unchecked"
					}
					log.Info("Assistant toggled item", "text", item.Text, "checked", item.Checked)
					bus.Publish(events.CartUpdated{CartID: cart.ID})
					return fmt.Sprintf("%s %q", status, item.Text), nil
				}
			}
			return fmt.Sprintf("item %q not found in the list", args.Text), nil
		}),

		llm.Func("list_items", "List all items currently in the shopping list", func(args struct {
			IncludeChecked bool `json:"include_checked" desc:"whether to include already checked-off items" required:"false"`
		}) ([]string, error) {
			var items []string
			for _, item := range cart.Items {
				if !args.IncludeChecked && item.Checked {
					continue
				}
				status := ""
				if item.Checked {
					status = " (checked)"
				}
				items = append(items, item.Text+status)
			}
			log.Info("Assistant listed items", "count", len(items))
			return items, nil
		}),

		llm.Func("parse_recipe", "Parse ingredients from a recipe URL and add them to the shopping list", func(args struct {
			URL string `json:"url" desc:"the recipe URL to parse ingredients from"`
		}) ([]string, error) {
			u, err := url.Parse(args.URL)
			if err != nil {
				return nil, fmt.Errorf("invalid URL: %w", err)
			}
			ingredients, err := recipe.Parse(ctx, u)
			if err != nil {
				return nil, err
			}
			for _, text := range ingredients {
				item := cart.Add(text, "assistant")
				log.Info("Assistant added ingredient", "text", text, "itemID", item.ID)
			}
			if err := repo.Save(cart); err != nil {
				return nil, err
			}
			bus.Publish(events.CartUpdated{CartID: cart.ID})
			return ingredients, nil
		}),
	}
}

var chatPromptTpl = template.Must(template.New("").Funcs(template.FuncMap{
	"name": func(m *carts.Message) string {
		if m.UserID != nil {
			return *m.UserID
		}
		return string(m.Role)
	},
}).Parse(`You are a helpful shopping list assistant. You live in the chat of a shared shopping list app.
Users chat with each other here. Only respond if the message is clearly addressed to you (the assistant) or is a request you can help with (adding/removing items, answering questions about the list).
If it's just users chatting with each other, set answer to "" and do nothing.

Chat history:
{{ range . }}[{{ .CreatedAt.Format "15:04" }}] {{ name . }}: {{ .Text }}
{{ end }}`))
