package data

import (
	"context"
	"errors"
	"strconv"

	"micro-one-api/internal/billing/biz"

	"gorm.io/gorm"
)

type accountRepo struct {
	data *Data
}

func NewAccountRepo(data *Data) biz.AccountRepo {
	return &accountRepo{data: data}
}

func (r *accountRepo) GetAccountSnapshot(ctx context.Context, userID string) (*biz.Account, error) {
	var user struct {
		ID           int64  `gorm:"column:id"`
		Username     string `gorm:"column:username"`
		DisplayName  string `gorm:"column:display_name"`
		Group        string `gorm:"column:group"`
		Balance      int64  `gorm:"column:balance"`
		UsedAmount   int64  `gorm:"column:used_amount"`
		RequestCount int64  `gorm:"column:request_count"`
		FrozenAmount int64  `gorm:"column:frozen_amount"`
		Status       int32  `gorm:"column:status"`
	}

	if err := r.data.db.WithContext(ctx).Table("users").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, biz.ErrAccountNotFound
		}
		return nil, err
	}

	return &biz.Account{
		UserID:       userID,
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		Group:        user.Group,
		Balance:      user.Balance,
		UsedAmount:   user.UsedAmount,
		RequestCount: user.RequestCount,
		FrozenAmount: user.FrozenAmount,
		Status:       user.Status,
	}, nil
}

func (r *accountRepo) UpdateBalance(ctx context.Context, userID string, delta int64, operationType string) (int64, error) {
	var account struct {
		Balance int64 `gorm:"column:balance"`
	}

	tx := r.data.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Table("users").Where("id = ?", userID).First(&account).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, biz.ErrAccountNotFound
		}
		return 0, err
	}

	newBalance := account.Balance + delta
	if newBalance < 0 {
		tx.Rollback()
		return 0, biz.ErrInsufficientQuota
	}

	if err := tx.Table("users").Where("id = ?", userID).Update("balance", newBalance).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (r *accountRepo) UpdateUsage(ctx context.Context, userID string, usedAmountDelta, requestCountDelta int64) error {
	return r.data.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"used_amount":   gorm.Expr("used_amount + ?", usedAmountDelta),
			"request_count": gorm.Expr("request_count + ?", requestCountDelta),
		}).Error
}

func (r *accountRepo) UpdateFrozenAmount(ctx context.Context, userID string, delta int64) error {
	var account struct {
		FrozenAmount int64 `gorm:"column:frozen_amount"`
	}
	if err := r.data.db.WithContext(ctx).Table("users").Where("id = ?", userID).First(&account).Error; err != nil {
		return err
	}

	newFrozenAmount := account.FrozenAmount + delta
	return r.data.db.WithContext(ctx).Table("users").Where("id = ?", userID).Update("frozen_amount", newFrozenAmount).Error
}

func (r *accountRepo) BatchGetAccountSnapshots(ctx context.Context, userIDs []string) (map[string]*biz.Account, error) {
	if len(userIDs) == 0 {
		return map[string]*biz.Account{}, nil
	}

	var users []struct {
		ID           int64  `gorm:"column:id"`
		Username     string `gorm:"column:username"`
		DisplayName  string `gorm:"column:display_name"`
		Group        string `gorm:"column:group"`
		Balance      int64  `gorm:"column:balance"`
		UsedAmount   int64  `gorm:"column:used_amount"`
		RequestCount int64  `gorm:"column:request_count"`
		FrozenAmount int64  `gorm:"column:frozen_amount"`
		Status       int32  `gorm:"column:status"`
	}

	if err := r.data.db.WithContext(ctx).Table("users").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	result := make(map[string]*biz.Account, len(users))
	for _, user := range users {
		userID := strconv.FormatInt(user.ID, 10)
		result[userID] = &biz.Account{
			UserID:       userID,
			Username:     user.Username,
			DisplayName:  user.DisplayName,
			Group:        user.Group,
			Balance:      user.Balance,
			UsedAmount:   user.UsedAmount,
			RequestCount: user.RequestCount,
			FrozenAmount: user.FrozenAmount,
			Status:       user.Status,
		}
	}

	return result, nil
}
