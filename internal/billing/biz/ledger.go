package biz

import "time"

const (
	LedgerTypeConsume  = "consume"
	LedgerTypeRecharge = "recharge"
	LedgerTypeRefund   = "refund"
	LedgerTypeRedeem   = "redeem"
)

type Ledger struct {
	ID           uint
	UserID       string
	Amount       int64
	BalanceAfter int64
	Type         string
	ReferenceID  string
	Remark       string
	CreatedAt    time.Time
}
