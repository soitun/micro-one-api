package data

import (
	"os"

	"micro-one-api/internal/billing/biz"
	"micro-one-api/internal/pkg/xdb"

	"gorm.io/gorm"
)

type Data struct {
	db    *gorm.DB
	redis interface{}

	accountRepo     biz.AccountRepo
	reservationRepo biz.ReservationRepo
	ledgerRepo      biz.LedgerRepo
	redeemRepo      biz.RedeemRepo
}

func NewData() (*Data, error) {
	dsn := os.Getenv("BILLING_SQL_DSN")
	if dsn == "" {
		dsn = os.Getenv("SQL_DSN")
	}

	db, err := xdb.OpenMySQL(dsn)
	if err != nil {
		return nil, err
	}

	d := &Data{
		db:    db,
		redis: nil,
	}

	d.accountRepo = NewAccountRepo(d)
	d.reservationRepo = NewReservationRepo(d)
	d.ledgerRepo = NewLedgerRepo(d)
	d.redeemRepo = NewRedeemRepo(d)

	return d, nil
}

func (d *Data) AccountRepo() biz.AccountRepo {
	return d.accountRepo
}

func (d *Data) ReservationRepo() biz.ReservationRepo {
	return d.reservationRepo
}

func (d *Data) LedgerRepo() biz.LedgerRepo {
	return d.ledgerRepo
}

func (d *Data) RedeemRepo() biz.RedeemRepo {
	return d.redeemRepo
}
