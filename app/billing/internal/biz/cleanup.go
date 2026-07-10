package biz

import (
	"context"
	"time"

	"go.uber.org/zap"

	applogger "micro-one-api/platform/logging"
)

type CleanupJob struct {
	uc            *BillingUsecase
	checkInterval time.Duration
	stopChan      chan struct{}
}

func NewCleanupJob(uc *BillingUsecase, checkInterval time.Duration) *CleanupJob {
	return &CleanupJob{
		uc:            uc,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

func (j *CleanupJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := j.cleanupExpiredReservations(ctx); err != nil {
				applogger.Log.Warn("cleanup expired reservations failed", zap.Error(err))
			}
		case <-ctx.Done():
			applogger.Log.Info("cleanup job stopped", zap.String("reason", "context canceled"))
			return
		case <-j.stopChan:
			applogger.Log.Info("cleanup job stopped", zap.String("reason", "stop requested"))
			return
		}
	}
}

func (j *CleanupJob) Stop() {
	close(j.stopChan)
}

// cleanupExpiredReservations releases every reservation whose
// expired_at has passed. The release goes through the unified
// releaseReservation path so the wallet refund, the ledger entry,
// and the status transition are all in one transaction; expired
// reservations get the `expired` final status instead of `released`
// so reconciliation can distinguish "user gave up" from "system
// cleanup".
func (j *CleanupJob) cleanupExpiredReservations(ctx context.Context) error {
	reservations, err := j.uc.reservationRepo.GetExpiredReservations(ctx)
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if err := j.uc.releaseReservation(ctx, reservation.ReservationID, "reservation expired", ReservationStatusExpired); err != nil {
			applogger.Log.Warn("failed to release expired reservation", zap.String("reservation_id", reservation.ReservationID), zap.Error(err))
			continue
		}
		applogger.Log.Info("released expired reservation", zap.String("reservation_id", reservation.ReservationID))
	}

	return nil
}
