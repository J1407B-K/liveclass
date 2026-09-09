package domain

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseRole string

const (
	MallDatabase      DatabaseRole = "mall"
	OrderDatabase     DatabaseRole = "order"
	InventoryDatabase DatabaseRole = "inventory"
	PointsDatabase    DatabaseRole = "points"
)

var databaseNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func DSNForRole(role DatabaseRole) (string, error) {
	var envKey, defaultDB string
	switch role {
	case MallDatabase:
		envKey, defaultDB = "LIVECLASS_MALL_MYSQL_DSN", "liveclass_mall"
	case OrderDatabase:
		envKey, defaultDB = "LIVECLASS_MALL_ORDER_MYSQL_DSN", "liveclass_mall_order"
	case InventoryDatabase:
		envKey, defaultDB = "LIVECLASS_MALL_INVENTORY_MYSQL_DSN", "liveclass_mall_inventory"
	case PointsDatabase:
		envKey, defaultDB = "LIVECLASS_MALL_POINTS_MYSQL_DSN", "liveclass_mall_points"
	default:
		return "", fmt.Errorf("unknown mall database role %q", role)
	}
	if dsn := os.Getenv(envKey); dsn != "" {
		return dsn, nil
	}
	return fmt.Sprintf("root:123456@tcp(127.0.0.1:3306)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local&timeout=2s&readTimeout=2s&writeTimeout=2s", defaultDB), nil
}

// OpenMySQL opens exactly one service-owned database. For local development it
// creates a missing database; production databases should normally be provisioned separately.
func OpenMySQL(role DatabaseRole) (*gorm.DB, *sql.DB, error) {
	dsn, err := DSNForRole(role)
	if err != nil {
		return nil, nil, err
	}
	if err = ensureDatabaseExists(dsn); err != nil {
		return nil, nil, fmt.Errorf("prepare %s database: %w", role, err)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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

func ensureDatabaseExists(dsn string) error {
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return err
	}
	if cfg.DBName == "" || !databaseNamePattern.MatchString(cfg.DBName) {
		return errors.New("mall DSN requires a safe explicit database name")
	}
	probe, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	probeErr := probe.PingContext(probeCtx)
	cancel()
	_ = probe.Close()
	if probeErr == nil {
		return nil
	}
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(probeErr, &mysqlErr) || mysqlErr.Number != 1049 {
		return probeErr
	}
	databaseName := cfg.DBName
	cfg.DBName = ""
	bootstrap, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer bootstrap.Close()
	createCtx, createCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer createCancel()
	_, err = bootstrap.ExecContext(createCtx, "CREATE DATABASE IF NOT EXISTS `"+databaseName+"` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	return err
}

// EnsureBarrierTable creates a service-local barrier in the database selected
// by that service's DSN. It deliberately does not use a shared schema.
func EnsureBarrierTable(raw *sql.DB) error {
	if raw == nil {
		return fmt.Errorf("nil sql db")
	}
	_, err := raw.Exec(`CREATE TABLE IF NOT EXISTS dtm_barrier (
id BIGINT PRIMARY KEY AUTO_INCREMENT,
trans_type VARCHAR(45) DEFAULT '', gid VARCHAR(128) DEFAULT '', branch_id VARCHAR(128) DEFAULT '',
op VARCHAR(45) DEFAULT '', barrier_id VARCHAR(45) DEFAULT '', reason VARCHAR(45) DEFAULT '',
create_time DATETIME DEFAULT CURRENT_TIMESTAMP, update_time DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
KEY idx_barrier_create_time(create_time), KEY idx_barrier_update_time(update_time),
UNIQUE KEY uk_barrier(gid, branch_id, op, barrier_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func MigrateMall(db *gorm.DB) error {
	return db.AutoMigrate(&Product{})
}

func MigrateOrder(db *gorm.DB) error {
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
	return db.AutoMigrate(&Order{})
}

func MigrateInventory(db *gorm.DB) error {
	return db.AutoMigrate(&Inventory{}, &InventoryReservation{})
}

func MigratePoints(db *gorm.DB) error {
	return db.AutoMigrate(&PointsAccount{}, &PointsLedger{})
}
