package data

import "time"

type reservationModel struct {
	ID           uint      `gorm:"primaryKey;column:id"`
	ReservationID string   `gorm:"uniqueIndex;column:reservation_id"`
	UserID       string   `gorm:"index;column:user_id"`
	RequestID    string   `gorm:"index;column:request_id"`
	Amount       int64    `gorm:"column:amount"`
	Status       string   `gorm:"column:status"`
	Model        *string  `gorm:"column:model"`
	ChannelID    *string  `gorm:"column:channel_id"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
	ExpiredAt    *time.Time `gorm:"index;column:expired_at"`
}

func (reservationModel) TableName() string { return "billing_reservations" }

type ledgerModel struct {
	ID           uint      `gorm:"primaryKey;column:id"`
	UserID       string    `gorm:"index;column:user_id"`
	Amount       int64     `gorm:"column:amount"`
	BalanceAfter int64     `gorm:"column:balance_after"`
	Type         string    `gorm:"index;column:type"`
	ReferenceID  *string   `gorm:"index;column:reference_id"`
	Remark       *string   `gorm:"column:remark"`
	CreatedAt    time.Time `gorm:"index;column:created_at"`
}

func (ledgerModel) TableName() string { return "billing_ledgers" }

type redeemCodeModel struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Code      string    `gorm:"uniqueIndex;column:code"`
	Name      *string   `gorm:"column:name"`
	Amount    int64     `gorm:"column:amount"`
	Count     int       `gorm:"column:count"`
	Status    int8      `gorm:"column:status"`
	CreatedBy *string   `gorm:"column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (redeemCodeModel) TableName() string { return "billing_redeem_codes" }

type redeemRecordModel struct {
	ID          uint      `gorm:"primaryKey;column:id"`
	UserID      string    `gorm:"index;column:user_id"`
	Code        string    `gorm:"index;column:code"`
	Amount      int64     `gorm:"column:amount"`
	QuotaBefore int64     `gorm:"column:quota_before"`
	QuotaAfter  int64     `gorm:"column:quota_after"`
	CreatedAt   time.Time `gorm:"index;column:created_at"`
}

func (redeemRecordModel) TableName() string { return "billing_redeem_records" }
