package data

import (
	"context"
	"fmt"

	"micro-one-api/internal/billing/biz"
)

type reconciliationRepo struct {
	data *Data
}

// NewReconciliationRepo creates a new ReconciliationRepo.
func NewReconciliationRepo(data *Data) biz.ReconciliationRepo {
	return &reconciliationRepo{data: data}
}

func (r *reconciliationRepo) ListAllAccounts(ctx context.Context) ([]*biz.Account, error) {
	var models []accountModel
	if err := r.data.db.WithContext(ctx).Find(&models).Error; err != nil {
		return nil, err
	}

	accounts := make([]*biz.Account, len(models))
	for i, m := range models {
		accounts[i] = &biz.Account{
			UserID:       fmt.Sprintf("%d", m.ID),
			Username:     m.Username,
			DisplayName:  m.DisplayName,
			Group:        m.Group,
			Quota:        m.Quota,
			UsedQuota:    m.UsedQuota,
			RequestCount: m.RequestCount,
			FrozenQuota:  m.FrozenQuota,
			Status:       m.Status,
		}
	}
	return accounts, nil
}

func (r *reconciliationRepo) SumLedgerAmounts(ctx context.Context, userID string) (int64, error) {
	var sum int64
	if err := r.data.db.WithContext(ctx).
		Model(&ledgerModel{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error; err != nil {
		return 0, err
	}
	return sum, nil
}

type reconciliationChannelUsageModel struct {
	ID        int64 `gorm:"column:id"`
	UsedQuota int64 `gorm:"column:used_quota"`
}

func (reconciliationChannelUsageModel) TableName() string { return "channels" }

func (r *reconciliationRepo) ListChannelUsage(ctx context.Context) ([]*biz.ChannelUsageSnapshot, error) {
	var models []reconciliationChannelUsageModel
	if err := r.data.db.WithContext(ctx).
		Select("id, used_quota").
		Find(&models).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ChannelUsageSnapshot, len(models))
	for i, m := range models {
		out[i] = &biz.ChannelUsageSnapshot{
			ChannelID: m.ID,
			UsedQuota: m.UsedQuota,
		}
	}
	return out, nil
}

func (r *reconciliationRepo) SumConsumeLedgerUsageByChannel(ctx context.Context) ([]*biz.ChannelLedgerUsage, error) {
	type row struct {
		ChannelID    int64 `gorm:"column:channel_id"`
		Quota        int64 `gorm:"column:quota"`
		UpstreamCost int64 `gorm:"column:upstream_cost"`
	}
	var rows []row
	if err := r.data.db.WithContext(ctx).
		Model(&ledgerModel{}).
		Select("channel_id, COALESCE(SUM(quota), 0) AS quota, COALESCE(SUM(upstream_cost), 0) AS upstream_cost").
		Where("type = ? AND channel_id > 0", biz.LedgerTypeConsume).
		Group("channel_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*biz.ChannelLedgerUsage, len(rows))
	for i, row := range rows {
		out[i] = &biz.ChannelLedgerUsage{
			ChannelID:    row.ChannelID,
			Quota:        row.Quota,
			UpstreamCost: row.UpstreamCost,
		}
	}
	return out, nil
}

func (r *reconciliationRepo) GetLedgerConsumeSummary(ctx context.Context) (*biz.ConsumeSummary, error) {
	var summary biz.ConsumeSummary
	if err := r.data.db.WithContext(ctx).
		Model(&ledgerModel{}).
		Select("COUNT(*) AS count, COALESCE(SUM(quota), 0) AS quota").
		Where("type = ?", biz.LedgerTypeConsume).
		Scan(&summary).Error; err != nil {
		return nil, err
	}
	return &summary, nil
}

type reconciliationLogModel struct {
	ID    int64 `gorm:"column:id"`
	Quota int64 `gorm:"column:quota"`
}

func (reconciliationLogModel) TableName() string { return "logs" }

func (r *reconciliationRepo) GetLogConsumeSummary(ctx context.Context) (*biz.ConsumeSummary, error) {
	var summary biz.ConsumeSummary
	if err := r.data.db.WithContext(ctx).
		Model(&reconciliationLogModel{}).
		Select("COUNT(*) AS count, COALESCE(SUM(quota), 0) AS quota").
		Where("level = ?", biz.LedgerTypeConsume).
		Scan(&summary).Error; err != nil {
		return nil, err
	}
	return &summary, nil
}

func (r *reconciliationRepo) ListReservationsByStatus(ctx context.Context, status string) ([]*biz.Reservation, error) {
	var models []reservationModel
	if err := r.data.db.WithContext(ctx).
		Where("status = ?", status).
		Find(&models).Error; err != nil {
		return nil, err
	}

	reservations := make([]*biz.Reservation, len(models))
	for i, m := range models {
		reservations[i] = &biz.Reservation{
			ReservationID: m.ReservationID,
			UserID:        m.UserID,
			RequestID:     m.RequestID,
			Amount:        m.Amount,
			Status:        m.Status,
			Model:         stringFromPtr(m.Model),
			ChannelID:     stringFromPtr(m.ChannelID),
			CreatedAt:     m.CreatedAt,
			UpdatedAt:     m.UpdatedAt,
			ExpiredAt:     timeFromPtr(m.ExpiredAt),
		}
	}
	return reservations, nil
}
