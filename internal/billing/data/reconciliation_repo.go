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
			UserID:       m.UserID,
			RequestID:    m.RequestID,
			Amount:       m.Amount,
			Status:       m.Status,
			Model:        stringFromPtr(m.Model),
			ChannelID:    stringFromPtr(m.ChannelID),
			CreatedAt:    m.CreatedAt,
			UpdatedAt:    m.UpdatedAt,
			ExpiredAt:    timeFromPtr(m.ExpiredAt),
		}
	}
	return reservations, nil
}
