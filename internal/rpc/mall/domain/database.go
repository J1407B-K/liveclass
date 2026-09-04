package domain

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func DSNFromEnv() string {
	if dsn := os.Getenv("LIVECLASS_MALL_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return "root:123456@tcp(127.0.0.1:3306)/liveclass?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local&timeout=2s&readTimeout=2s&writeTimeout=2s"
}

func OpenMySQL() (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(mysql.Open(DSNFromEnv()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, nil, err
	}
	raw, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	raw.SetMaxOpenConns(64)
	raw.SetMaxIdleConns(16)
	raw.SetConnMaxLifetime(30 * time.Minute)
	return db, raw, nil
}

func EnsureBarrierTable(raw *sql.DB) error {
	if raw == nil {
		return fmt.Errorf("nil sql db")
	}
	statements := []string{
		"CREATE DATABASE IF NOT EXISTS dtm_barrier DEFAULT CHARACTER SET utf8mb4",
		`CREATE TABLE IF NOT EXISTS dtm_barrier.barrier (
id BIGINT PRIMARY KEY AUTO_INCREMENT,
trans_type VARCHAR(45) DEFAULT '', gid VARCHAR(128) DEFAULT '', branch_id VARCHAR(128) DEFAULT '',
op VARCHAR(45) DEFAULT '', barrier_id VARCHAR(45) DEFAULT '', reason VARCHAR(45) DEFAULT '',
create_time DATETIME DEFAULT CURRENT_TIMESTAMP, update_time DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
KEY idx_barrier_create_time(create_time), KEY idx_barrier_update_time(update_time),
UNIQUE KEY uk_barrier(gid, branch_id, op, barrier_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, statement := range statements {
		if _, err := raw.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func MigrateOrder(db *gorm.DB) error {
	// Early development versions let GORM infer SagaGID as saga_g_id.
	// Remove only that mall-specific compatibility column before enforcing the
	// explicit saga_gid mapping.
	if db.Migrator().HasColumn("mall_orders", "saga_g_id") {
		if db.Migrator().HasIndex("mall_orders", "idx_mall_orders_saga_g_id") {
			if err := db.Migrator().DropIndex("mall_orders", "idx_mall_orders_saga_g_id"); err != nil {
				return err
			}
		}
		if err := db.Migrator().DropColumn("mall_orders", "saga_g_id"); err != nil {
			return err
		}
	}
	return db.AutoMigrate(&Product{}, &Order{})
}

func MigrateInventory(db *gorm.DB) error {
	return db.AutoMigrate(&Inventory{}, &InventoryReservation{})
}

func MigratePoints(db *gorm.DB) error {
	return db.AutoMigrate(&PointsAccount{}, &PointsLedger{})
}
