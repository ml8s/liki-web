package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ml8s/liki-web/internal/account"
	"github.com/ml8s/liki-web/internal/aipay"
	"github.com/ml8s/liki-web/internal/auth"
	"github.com/ml8s/liki-web/internal/birthprofile"
	"github.com/ml8s/liki-web/internal/payment"
)

func startErasureWorker(wg *sync.WaitGroup, ctx context.Context, queue *account.ErasureStore, handler *account.Handler) {
	wg.Go(func() {
		worker := &account.ErasureWorker{Queue: queue, Handler: handler, Interval: 30 * time.Second}
		worker.Start(ctx)
	})
}

func startAuthCleanupWorker(wg *sync.WaitGroup, ctx context.Context, store *auth.Store, retentionDays int) {
	wg.Go(func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := store.Cleanup(ctx, retentionDays); err != nil {
					slog.Error("auth cleanup", "err", err)
				}
			}
		}
	})
}

// startProfileCleanupWorker physically removes profiles only after their soft
// delete recovery window expires.
func startProfileCleanupWorker(wg *sync.WaitGroup, ctx context.Context, store *birthprofile.Store, retentionDays int) {
	wg.Go(func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
				purged, err := store.CleanupExpired(ctx, cutoff)
				if err != nil {
					slog.Error("profile cleanup", "err", err)
					continue
				}
				if purged > 0 {
					slog.Info("profile cleanup complete", "purged", purged)
				}
			}
		}
	})
}

func startPaymentWorkers(
	wg *sync.WaitGroup,
	ctx context.Context,
	runtime *payment.Runtime,
	aipayStore *aipay.Store,
	config Config,
) {
	wg.Go(func() { runtime.StartNotificationWorker(ctx, 30*time.Second) })
	wg.Go(func() { runtime.StartPaymentCompletionWorker(ctx, 5*time.Second) })
	wg.Go(func() { runtime.StartReconciliationWorker(ctx, time.Minute) })
	wg.Go(func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				expiredIntents, deletedReceipts, err := runtime.RunRetention(
					ctx,
					time.Duration(config.PaymentIntentExpiryDays)*24*time.Hour,
					time.Duration(config.PaymentNotificationRetentionDays)*24*time.Hour,
				)
				if err != nil {
					slog.Error("payment retention", "err", err)
					continue
				}
				if expiredIntents > 0 || deletedReceipts > 0 {
					slog.Info("payment retention complete",
						"expired_intents", expiredIntents,
						"deleted_notifications", deletedReceipts,
					)
				}
				purged, err := aipayStore.CleanupExpiredBills(ctx, time.Now().UTC())
				if err != nil {
					slog.Error("aipay bill cleanup", "err", err)
					continue
				}
				if purged > 0 {
					slog.Info("aipay expired bills cleaned", "count", purged)
				}
			}
		}
	})
}
