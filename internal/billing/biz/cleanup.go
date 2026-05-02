package biz

import (
	"context"
	"log"
	"time"
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
				log.Printf("cleanup expired reservations failed: %v", err)
			}
		case <-ctx.Done():
			log.Println("cleanup job stopped")
			return
		case <-j.stopChan:
			log.Println("cleanup job stopped")
			return
		}
	}
}

func (j *CleanupJob) Stop() {
	close(j.stopChan)
}

func (j *CleanupJob) cleanupExpiredReservations(ctx context.Context) error {
	reservations, err := j.uc.reservationRepo.GetExpiredReservations(ctx)
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if err := j.uc.ReleaseQuota(ctx, reservation.ReservationID, "reservation expired"); err != nil {
			log.Printf("failed to release expired reservation %s: %v", reservation.ReservationID, err)
		} else {
			log.Printf("released expired reservation %s", reservation.ReservationID)
		}

		if err := j.uc.reservationRepo.UpdateReservationStatus(ctx, reservation.ReservationID, ReservationStatusExpired); err != nil {
			log.Printf("failed to update reservation status to expired: %v", err)
		}
	}

	return nil
}
