package biz

import (
	"context"
	"fmt"
	"time"
)

// ReconciliationRepo provides data access for reconciliation tasks.
type ReconciliationRepo interface {
	// ListAllAccounts returns all accounts for consistency checks.
	ListAllAccounts(ctx context.Context) ([]*Account, error)
	// SumLedgerAmounts returns the net ledger amount for a user (sum of all amounts).
	SumLedgerAmounts(ctx context.Context, userID string) (int64, error)
	// ListChannelUsage returns the current channel usage counters.
	ListChannelUsage(ctx context.Context) ([]*ChannelUsageSnapshot, error)
	// SumConsumeLedgerUsageByChannel returns local consume ledger totals grouped by channel.
	SumConsumeLedgerUsageByChannel(ctx context.Context) ([]*ChannelLedgerUsage, error)
	// GetLedgerConsumeSummary returns the local consume ledger summary.
	GetLedgerConsumeSummary(ctx context.Context) (*ConsumeSummary, error)
	// GetLogConsumeSummary returns the duplicated consume log summary.
	GetLogConsumeSummary(ctx context.Context) (*ConsumeSummary, error)
	// ListReservationsByStatus returns reservations with the given status.
	ListReservationsByStatus(ctx context.Context, status string) ([]*Reservation, error)
}

// ReconciliationRunStore persists historical reconciliation runs so admins can review them.
type ReconciliationRunStore interface {
	SaveRun(ctx context.Context, result *ReconciliationResult) (int64, error)
	ListRuns(ctx context.Context, page, pageSize int32) ([]*ReconciliationResult, int64, error)
	GetRun(ctx context.Context, runID int64) (*ReconciliationResult, error)
}

// ReconciliationResult holds the outcome of a reconciliation run.
type ReconciliationResult struct {
	RunID                  int64                  `json:"run_id,omitempty"`
	RunAt                  time.Time              `json:"run_at"`
	ExpiredCleaned         int                    `json:"expired_cleaned"`
	AccountInconsistencies []AccountInconsistency `json:"account_inconsistencies,omitempty"`
	ChannelInconsistencies []ChannelInconsistency `json:"channel_inconsistencies,omitempty"`
	LogInconsistencies     []LogInconsistency     `json:"log_inconsistencies,omitempty"`
	TotalAccounts          int                    `json:"total_accounts"`
	TotalChannels          int                    `json:"total_channels"`
	TotalReservations      int                    `json:"total_reservations"`
}

const (
	ReconciliationDiscrepancyTypeAccount = "account_quota"
	ReconciliationDiscrepancyTypeChannel = "channel_usage"
	ReconciliationDiscrepancyTypeLog     = "ledger_log_consume"
)

func (r *ReconciliationResult) DiscrepancyCount() int {
	if r == nil {
		return 0
	}
	return len(r.AccountInconsistencies) + len(r.ChannelInconsistencies) + len(r.LogInconsistencies)
}

// AccountInconsistency describes a quota mismatch for a single account.
type AccountInconsistency struct {
	UserID          string `json:"user_id"`
	ExpectedQuota   int64  `json:"expected_quota"`
	ActualQuota     int64  `json:"actual_quota"`
	LedgerNetAmount int64  `json:"ledger_net_amount"`
	FrozenQuota     int64  `json:"frozen_quota"`
}

// ChannelUsageSnapshot is the current usage counter stored on a channel.
type ChannelUsageSnapshot struct {
	ChannelID int64
	UsedQuota int64
}

// ChannelLedgerUsage is the local ledger usage/cost summary for a channel.
type ChannelLedgerUsage struct {
	ChannelID    int64
	Quota        int64
	UpstreamCost int64
}

// ChannelInconsistency describes a mismatch between channel usage counters and local ledgers.
type ChannelInconsistency struct {
	ChannelID         int64 `json:"channel_id"`
	ExpectedUsedQuota int64 `json:"expected_used_quota"`
	ActualUsedQuota   int64 `json:"actual_used_quota"`
	LedgerQuota       int64 `json:"ledger_quota"`
	UpstreamCost      int64 `json:"upstream_cost"`
	Difference        int64 `json:"difference"`
}

// ConsumeSummary is a compact summary of consume records in a duplicated write path.
type ConsumeSummary struct {
	Count int64
	Quota int64
}

// LogInconsistency describes drift between billing_ledgers and logs consume records.
type LogInconsistency struct {
	LedgerCount int64 `json:"ledger_count"`
	LogCount    int64 `json:"log_count"`
	LedgerQuota int64 `json:"ledger_quota"`
	LogQuota    int64 `json:"log_quota"`
	CountDiff   int64 `json:"count_diff"`
	QuotaDiff   int64 `json:"quota_diff"`
}

// ReconciliationUsecase runs billing reconciliation tasks.
type ReconciliationUsecase struct {
	accountRepo     AccountRepo
	reservationRepo ReservationRepo
	reconRepo       ReconciliationRepo
	runStore        ReconciliationRunStore
}

// NewReconciliationUsecase creates a new ReconciliationUsecase. runStore is optional —
// when nil the runs are not persisted (matches legacy behavior).
func NewReconciliationUsecase(
	accountRepo AccountRepo,
	reservationRepo ReservationRepo,
	reconRepo ReconciliationRepo,
	runStore ReconciliationRunStore,
) *ReconciliationUsecase {
	return &ReconciliationUsecase{
		accountRepo:     accountRepo,
		reservationRepo: reservationRepo,
		reconRepo:       reconRepo,
		runStore:        runStore,
	}
}

// RunReconciliation performs a full reconciliation: cleans expired reservations and checks quota consistency.
func (uc *ReconciliationUsecase) RunReconciliation(ctx context.Context) (*ReconciliationResult, error) {
	result := &ReconciliationResult{
		RunAt: time.Now(),
	}

	// Step 1: Clean up expired reservations
	expired, err := uc.reservationRepo.GetExpiredReservations(ctx)
	if err != nil {
		return nil, fmt.Errorf("get expired reservations: %w", err)
	}

	for _, res := range expired {
		if res.IsReserved() {
			// Release the expired reservation
			if err := uc.reservationRepo.UpdateReservationStatus(ctx, res.ReservationID, ReservationStatusExpired); err != nil {
				continue
			}
			// Unfreeze the quota
			_ = uc.accountRepo.UpdateFrozenQuota(ctx, res.UserID, -res.Amount)
			// Return the quota
			_, _ = uc.accountRepo.UpdateQuota(ctx, res.UserID, res.Amount, LedgerTypeRefund)
			result.ExpiredCleaned++
		}
	}

	// Step 2: Check account quota consistency
	accounts, err := uc.reconRepo.ListAllAccounts(ctx)
	if err != nil {
		return result, fmt.Errorf("list accounts: %w", err)
	}
	result.TotalAccounts = len(accounts)

	for _, account := range accounts {
		ledgerNet, err := uc.reconRepo.SumLedgerAmounts(ctx, account.UserID)
		if err != nil {
			continue
		}

		// The account's quota should roughly match the ledger net amount
		// Allow a tolerance of 100 units for rounding
		diff := account.Quota - ledgerNet
		if diff < 0 {
			diff = -diff
		}
		if diff > 100 {
			result.AccountInconsistencies = append(result.AccountInconsistencies, AccountInconsistency{
				UserID:          account.UserID,
				ExpectedQuota:   ledgerNet,
				ActualQuota:     account.Quota,
				LedgerNetAmount: ledgerNet,
				FrozenQuota:     account.FrozenQuota,
			})
		}
	}

	// Step 3: Check channel usage counters against local consume ledgers.
	channels, err := uc.reconRepo.ListChannelUsage(ctx)
	if err != nil {
		return result, fmt.Errorf("list channel usage: %w", err)
	}
	result.TotalChannels = len(channels)
	channelByID := make(map[int64]*ChannelUsageSnapshot, len(channels))
	for _, channel := range channels {
		channelByID[channel.ChannelID] = channel
	}
	ledgerUsage, err := uc.reconRepo.SumConsumeLedgerUsageByChannel(ctx)
	if err != nil {
		return result, fmt.Errorf("sum channel ledger usage: %w", err)
	}
	seenChannels := make(map[int64]bool, len(ledgerUsage))
	for _, usage := range ledgerUsage {
		if usage.ChannelID <= 0 {
			continue
		}
		seenChannels[usage.ChannelID] = true
		actual := int64(0)
		if channel := channelByID[usage.ChannelID]; channel != nil {
			actual = channel.UsedQuota
		}
		if diffAbs(actual-usage.Quota) > 100 {
			result.ChannelInconsistencies = append(result.ChannelInconsistencies, ChannelInconsistency{
				ChannelID:         usage.ChannelID,
				ExpectedUsedQuota: usage.Quota,
				ActualUsedQuota:   actual,
				LedgerQuota:       usage.Quota,
				UpstreamCost:      usage.UpstreamCost,
				Difference:        actual - usage.Quota,
			})
		}
	}
	for _, channel := range channels {
		if channel.ChannelID <= 0 || seenChannels[channel.ChannelID] || channel.UsedQuota == 0 {
			continue
		}
		result.ChannelInconsistencies = append(result.ChannelInconsistencies, ChannelInconsistency{
			ChannelID:         channel.ChannelID,
			ExpectedUsedQuota: 0,
			ActualUsedQuota:   channel.UsedQuota,
			Difference:        channel.UsedQuota,
		})
	}

	// Step 4: Check duplicated consume writes between billing_ledgers and logs.
	ledgerSummary, err := uc.reconRepo.GetLedgerConsumeSummary(ctx)
	if err != nil {
		return result, fmt.Errorf("get ledger consume summary: %w", err)
	}
	logSummary, err := uc.reconRepo.GetLogConsumeSummary(ctx)
	if err != nil {
		return result, fmt.Errorf("get log consume summary: %w", err)
	}
	if diffAbs(ledgerSummary.Count-logSummary.Count) > 0 || diffAbs(ledgerSummary.Quota-logSummary.Quota) > 100 {
		result.LogInconsistencies = append(result.LogInconsistencies, LogInconsistency{
			LedgerCount: ledgerSummary.Count,
			LogCount:    logSummary.Count,
			LedgerQuota: ledgerSummary.Quota,
			LogQuota:    logSummary.Quota,
			CountDiff:   ledgerSummary.Count - logSummary.Count,
			QuotaDiff:   ledgerSummary.Quota - logSummary.Quota,
		})
	}

	// Count reserved reservations
	reserved, _ := uc.reconRepo.ListReservationsByStatus(ctx, ReservationStatusReserved)
	result.TotalReservations = len(reserved)

	if uc.runStore != nil {
		if runID, err := uc.runStore.SaveRun(ctx, result); err == nil {
			result.RunID = runID
		}
	}

	return result, nil
}

// ListReconciliationRuns returns paginated historical runs, newest first.
// Returns an empty slice when no runStore is configured.
func (uc *ReconciliationUsecase) ListReconciliationRuns(ctx context.Context, page, pageSize int32) ([]*ReconciliationResult, int64, error) {
	if uc.runStore == nil {
		return nil, 0, nil
	}
	return uc.runStore.ListRuns(ctx, page, pageSize)
}

// GetReconciliationRun returns a single historical run by ID.
// Returns nil, nil when no runStore is configured.
func (uc *ReconciliationUsecase) GetReconciliationRun(ctx context.Context, runID int64) (*ReconciliationResult, error) {
	if uc.runStore == nil {
		return nil, nil
	}
	return uc.runStore.GetRun(ctx, runID)
}

func diffAbs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
