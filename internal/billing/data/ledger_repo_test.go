package data

import (
	"context"
	"testing"
	"time"

	"micro-one-api/internal/billing/biz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLedgerTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE billing_ledgers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT,
			amount INTEGER,
			balance_after INTEGER,
			type TEXT,
			reference_id TEXT,
			remark TEXT,
			created_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestLedgerRepo_CreateLedger(t *testing.T) {
	db := setupLedgerTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	data := &Data{db: db}
	repo := NewLedgerRepo(data)

	ctx := context.Background()
	ledger := &biz.Ledger{
		UserID:       "user1",
		Amount:       -100,
		BalanceAfter: 900,
		Type:         "consume",
		ReferenceID:  "res_test_001",
		Remark:       "test consume",
	}

	err := repo.CreateLedger(ctx, ledger)
	require.NoError(t, err)

	// 验证数据已插入
	var model ledgerModel
	err = db.Where("user_id = ?", "user1").First(&model).Error
	require.NoError(t, err)
	assert.Equal(t, "user1", model.UserID)
	assert.Equal(t, int64(-100), model.Amount)
	assert.Equal(t, int64(900), model.BalanceAfter)
	assert.Equal(t, "consume", model.Type)
}

func TestLedgerRepo_ListLedgers(t *testing.T) {
	db := setupLedgerTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 插入测试数据
	now := time.Now()
	err := db.Exec(`
		INSERT INTO billing_ledgers (user_id, amount, balance_after, type, reference_id, remark, created_at)
		VALUES
			('user1', -100, 900, 'consume', 'res_001', 'consume 1', ?),
			('user1', 500, 1400, 'recharge', NULL, 'topup', ?),
			('user1', 50, 1450, 'refund', 'res_001', 'refund', ?),
			('user2', -200, 800, 'consume', 'res_002', 'consume 2', ?)
	`, now, now.Add(-time.Hour), now.Add(-30*time.Minute), now).Error
	require.NoError(t, err)

	data := &Data{db: db}
	repo := NewLedgerRepo(data)

	ctx := context.Background()
	ledgers, total, err := repo.ListLedgers(ctx, "user1", 1, 10)

	require.NoError(t, err)
	assert.Len(t, ledgers, 3)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, "user1", ledgers[0].UserID)
}

func TestLedgerRepo_ListLedgers_Pagination(t *testing.T) {
	db := setupLedgerTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 插入测试数据
	now := time.Now()
	for i := 0; i < 15; i++ {
		err := db.Exec(`
			INSERT INTO billing_ledgers (user_id, amount, balance_after, type, reference_id, remark, created_at)
			VALUES (?, -100, ?, 'consume', ?, ?, ?)
		`, "user1", 1000-i*100, "res_"+string(rune('0'+i)), "consume "+string(rune('0'+i)), now.Add(time.Duration(-i)*time.Minute)).Error
		require.NoError(t, err)
	}

	data := &Data{db: db}
	repo := NewLedgerRepo(data)

	ctx := context.Background()

	// 第一页
	ledgers1, total, err := repo.ListLedgers(ctx, "user1", 1, 10)
	require.NoError(t, err)
	assert.Len(t, ledgers1, 10)
	assert.Equal(t, int64(15), total)

	// 第二页
	ledgers2, _, err := repo.ListLedgers(ctx, "user1", 2, 10)
	require.NoError(t, err)
	assert.Len(t, ledgers2, 5)
}

func TestLedgerRepo_ListLedgers_Empty(t *testing.T) {
	db := setupLedgerTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	data := &Data{db: db}
	repo := NewLedgerRepo(data)

	ctx := context.Background()
	ledgers, total, err := repo.ListLedgers(ctx, "user1", 1, 10)

	require.NoError(t, err)
	assert.Len(t, ledgers, 0)
	assert.Equal(t, int64(0), total)
}