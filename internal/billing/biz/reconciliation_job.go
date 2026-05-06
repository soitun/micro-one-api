package biz

import (
	"context"
	"log"
	"time"
)

// ReconciliationJob runs periodic billing reconciliation tasks.
type ReconciliationJob struct {
	uc            *ReconciliationUsecase
	schedule      string // cron expression (not used in simple ticker mode)
	checkInterval time.Duration
	stopChan      chan struct{}
}

// NewReconciliationJob creates a new reconciliation job.
func NewReconciliationJob(uc *ReconciliationUsecase, checkInterval time.Duration) *ReconciliationJob {
	if checkInterval <= 0 {
		checkInterval = 1 * time.Hour
	}
	return &ReconciliationJob{
		uc:            uc,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the reconciliation loop. Call Stop() to terminate.
func (j *ReconciliationJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.checkInterval)
	defer ticker.Stop()

	// Run once on startup
	j.runReconciliation(ctx)

	for {
		select {
		case <-ticker.C:
			j.runReconciliation(ctx)
		case <-ctx.Done():
			log.Println("reconciliation job stopped")
			return
		case <-j.stopChan:
			log.Println("reconciliation job stopped")
			return
		}
	}
}

// Stop terminates the reconciliation job.
func (j *ReconciliationJob) Stop() {
	close(j.stopChan)
}

func (j *ReconciliationJob) runReconciliation(ctx context.Context) {
	result, err := j.uc.RunReconciliation(ctx)
	if err != nil {
		log.Printf("reconciliation failed: %v", err)
		return
	}

	log.Printf("reconciliation completed: run_at=%s, expired_cleaned=%d, total_accounts=%d, inconsistencies=%d",
		result.RunAt.Format(time.RFC3339),
		result.ExpiredCleaned,
		result.TotalAccounts,
		len(result.AccountInconsistencies),
	)

	for _, inc := range result.AccountInconsistencies {
		log.Printf("account inconsistency: user_id=%s, expected=%d, actual=%d, frozen=%d",
			inc.UserID, inc.ExpectedQuota, inc.ActualQuota, inc.FrozenQuota)
	}
}
