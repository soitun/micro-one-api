package biz

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"micro-one-api/internal/pkg/metrics"
)

type mockReconRepo struct {
	accounts             []*Account
	ledgerSums           map[string]int64
	reservations         []*Reservation
	channels             []*ChannelUsageSnapshot
	channelLedgerUsage   []*ChannelLedgerUsage
	ledgerConsumeSummary *ConsumeSummary
	logConsumeSummary    *ConsumeSummary
}

func (m *mockReconRepo) ListAllAccounts(ctx context.Context) ([]*Account, error) {
	return m.accounts, nil
}

func (m *mockReconRepo) SumLedgerAmounts(ctx context.Context, userID string) (int64, error) {
	if sum, ok := m.ledgerSums[userID]; ok {
		return sum, nil
	}
	return 0, nil
}

func (m *mockReconRepo) ListChannelUsage(ctx context.Context) ([]*ChannelUsageSnapshot, error) {
	return m.channels, nil
}

func (m *mockReconRepo) SumConsumeLedgerUsageByChannel(ctx context.Context) ([]*ChannelLedgerUsage, error) {
	return m.channelLedgerUsage, nil
}

func (m *mockReconRepo) GetLedgerConsumeSummary(ctx context.Context) (*ConsumeSummary, error) {
	if m.ledgerConsumeSummary != nil {
		return m.ledgerConsumeSummary, nil
	}
	return &ConsumeSummary{}, nil
}

func (m *mockReconRepo) GetLogConsumeSummary(ctx context.Context) (*ConsumeSummary, error) {
	if m.logConsumeSummary != nil {
		return m.logConsumeSummary, nil
	}
	return &ConsumeSummary{}, nil
}

func (m *mockReconRepo) ListReservationsByStatus(ctx context.Context, status string) ([]*Reservation, error) {
	var result []*Reservation
	for _, r := range m.reservations {
		if r.Status == status {
			result = append(result, r)
		}
	}
	return result, nil
}

func TestRunReconciliation_ExpiredReservations(t *testing.T) {
	account := &Account{UserID: "user1", Quota: 900, FrozenQuota: 100, Group: "default"}
	expiredRes := &Reservation{
		ReservationID: "res1",
		UserID:        "user1",
		Amount:        100,
		Status:        ReservationStatusReserved,
		ExpiredAt:     time.Now().Add(-10 * time.Minute),
	}

	accountRepo := &mockAccountRepo{account: account}
	reservationRepo := &mockReservationRepo{reservations: map[string]*Reservation{"res1": expiredRes}}
	reconRepo := &mockReconRepo{
		accounts:     []*Account{account},
		ledgerSums:   map[string]int64{"user1": 900},
		reservations: []*Reservation{expiredRes},
	}

	uc := NewReconciliationUsecase(accountRepo, reservationRepo, reconRepo, nil)
	result, err := uc.RunReconciliation(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.ExpiredCleaned)
	assert.Equal(t, ReservationStatusExpired, expiredRes.Status)
	assert.Equal(t, int64(0), account.FrozenQuota)
}

func TestRunReconciliation_QuotaConsistency(t *testing.T) {
	account := &Account{UserID: "user1", Quota: 1000, FrozenQuota: 0, Group: "default"}

	accountRepo := &mockAccountRepo{account: account}
	reservationRepo := &mockReservationRepo{reservations: make(map[string]*Reservation)}
	reconRepo := &mockReconRepo{
		accounts:   []*Account{account},
		ledgerSums: map[string]int64{"user1": 500}, // mismatch: quota=1000, ledger=500
	}

	uc := NewReconciliationUsecase(accountRepo, reservationRepo, reconRepo, nil)
	result, err := uc.RunReconciliation(context.Background())

	require.NoError(t, err)
	assert.Len(t, result.AccountInconsistencies, 1)
	assert.Equal(t, "user1", result.AccountInconsistencies[0].UserID)
	assert.Equal(t, int64(500), result.AccountInconsistencies[0].ExpectedQuota)
	assert.Equal(t, int64(1000), result.AccountInconsistencies[0].ActualQuota)
}

func TestRunReconciliation_NoInconsistencies(t *testing.T) {
	account := &Account{UserID: "user1", Quota: 1000, FrozenQuota: 0, Group: "default"}

	accountRepo := &mockAccountRepo{account: account}
	reservationRepo := &mockReservationRepo{reservations: make(map[string]*Reservation)}
	reconRepo := &mockReconRepo{
		accounts:   []*Account{account},
		ledgerSums: map[string]int64{"user1": 1000},
	}

	uc := NewReconciliationUsecase(accountRepo, reservationRepo, reconRepo, nil)
	result, err := uc.RunReconciliation(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result.AccountInconsistencies)
	assert.Equal(t, 1, result.TotalAccounts)
}

func TestRunReconciliation_RecordsMetrics(t *testing.T) {
	account := &Account{UserID: "user1", Quota: 1000, FrozenQuota: 0, Group: "default"}
	reconRepo := &mockReconRepo{
		accounts:   []*Account{account},
		ledgerSums: map[string]int64{"user1": 500},
	}
	uc := NewReconciliationUsecase(
		&mockAccountRepo{account: account},
		&mockReservationRepo{reservations: make(map[string]*Reservation)},
		reconRepo,
		nil,
	)
	runBefore := testutil.ToFloat64(metrics.ReconciliationRunsTotal.WithLabelValues("discrepancy"))
	accountDiffBefore := testutil.ToFloat64(metrics.ReconciliationDiscrepanciesTotal.WithLabelValues(ReconciliationDiscrepancyTypeAccount))

	_, err := uc.RunReconciliation(context.Background())

	require.NoError(t, err)
	assert.InEpsilon(t, 1, testutil.ToFloat64(metrics.ReconciliationRunsTotal.WithLabelValues("discrepancy"))-runBefore, 0.000001)
	assert.InEpsilon(t, 1, testutil.ToFloat64(metrics.ReconciliationDiscrepanciesTotal.WithLabelValues(ReconciliationDiscrepancyTypeAccount))-accountDiffBefore, 0.000001)
}

func TestRunReconciliation_ChannelUsageConsistency(t *testing.T) {
	account := &Account{UserID: "user1", Quota: 1000, FrozenQuota: 0, Group: "default"}
	reconRepo := &mockReconRepo{
		accounts:   []*Account{account},
		ledgerSums: map[string]int64{"user1": 1000},
		channels: []*ChannelUsageSnapshot{
			{ChannelID: 10, UsedQuota: 250},
		},
		channelLedgerUsage: []*ChannelLedgerUsage{
			{ChannelID: 10, Quota: 500, UpstreamCost: 123},
		},
	}

	uc := NewReconciliationUsecase(
		&mockAccountRepo{account: account},
		&mockReservationRepo{reservations: make(map[string]*Reservation)},
		reconRepo,
		nil,
	)
	result, err := uc.RunReconciliation(context.Background())

	require.NoError(t, err)
	require.Len(t, result.ChannelInconsistencies, 1)
	assert.Equal(t, int64(10), result.ChannelInconsistencies[0].ChannelID)
	assert.Equal(t, int64(500), result.ChannelInconsistencies[0].ExpectedUsedQuota)
	assert.Equal(t, int64(250), result.ChannelInconsistencies[0].ActualUsedQuota)
	assert.Equal(t, int64(123), result.ChannelInconsistencies[0].UpstreamCost)
	assert.Equal(t, 1, result.DiscrepancyCount())
}

func TestRunReconciliation_LogLedgerConsistency(t *testing.T) {
	account := &Account{UserID: "user1", Quota: 1000, FrozenQuota: 0, Group: "default"}
	reconRepo := &mockReconRepo{
		accounts:             []*Account{account},
		ledgerSums:           map[string]int64{"user1": 1000},
		ledgerConsumeSummary: &ConsumeSummary{Count: 2, Quota: 700},
		logConsumeSummary:    &ConsumeSummary{Count: 1, Quota: 400},
	}

	uc := NewReconciliationUsecase(
		&mockAccountRepo{account: account},
		&mockReservationRepo{reservations: make(map[string]*Reservation)},
		reconRepo,
		nil,
	)
	result, err := uc.RunReconciliation(context.Background())

	require.NoError(t, err)
	require.Len(t, result.LogInconsistencies, 1)
	assert.Equal(t, int64(2), result.LogInconsistencies[0].LedgerCount)
	assert.Equal(t, int64(1), result.LogInconsistencies[0].LogCount)
	assert.Equal(t, int64(300), result.LogInconsistencies[0].QuotaDiff)
	assert.Equal(t, 1, result.DiscrepancyCount())
}
