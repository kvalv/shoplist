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
