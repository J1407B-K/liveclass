package domain

import "time"

const (
	OrderPending     = "pending"
	OrderConfirmed   = "confirmed"
	OrderCompensated = "compensated"

	ReservationReserved  = "reserved"
	ReservationConfirmed = "confirmed"
	ReservationReleased  = "released"
)

type Product struct {
	ID          int64  `gorm:"primaryKey;autoIncrement:false"`
	Name        string `gorm:"type:varchar(128);not null"`
	Description string `gorm:"type:varchar(512);not null"`
	PointsPrice int64  `gorm:"not null"`
	Active      bool   `gorm:"not null;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Product) TableName() string { return "mall_products" }

type Order struct {
	ID          string `gorm:"primaryKey;type:varchar(64)"`
	UserID      int64  `gorm:"not null;uniqueIndex:uk_mall_order_request,priority:1;index"`
	RequestID   string `gorm:"type:varchar(64);not null;uniqueIndex:uk_mall_order_request,priority:2"`
	ProductID   int64  `gorm:"not null;index"`
	Quantity    int32  `gorm:"not null"`
	TotalPoints int64  `gorm:"not null"`
	Status      string `gorm:"type:varchar(24);not null;index"`
	SagaGID     string `gorm:"column:saga_gid;type:varchar(128);not null;uniqueIndex:uk_mall_order_saga_gid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Order) TableName() string { return "mall_orders" }

type Inventory struct {
	ProductID int64 `gorm:"primaryKey;autoIncrement:false"`
	Available int64 `gorm:"not null"`
	Reserved  int64 `gorm:"not null"`
	Sold      int64 `gorm:"not null"`
	Version   int64 `gorm:"not null"`
	UpdatedAt time.Time
}

func (Inventory) TableName() string { return "mall_inventories" }

type InventoryReservation struct {
	OrderID   string `gorm:"primaryKey;type:varchar(64)"`
	ProductID int64  `gorm:"not null;index"`
	Quantity  int64  `gorm:"not null"`
	Status    string `gorm:"type:varchar(24);not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (InventoryReservation) TableName() string { return "mall_inventory_reservations" }

type PointsAccount struct {
	UserID    int64 `gorm:"primaryKey;autoIncrement:false"`
	Balance   int64 `gorm:"not null"`
	Version   int64 `gorm:"not null"`
	UpdatedAt time.Time
}

func (PointsAccount) TableName() string { return "mall_points_accounts" }

type PointsLedger struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	OrderID   string `gorm:"type:varchar(64);not null;uniqueIndex:uk_mall_points_operation,priority:1"`
	Operation string `gorm:"type:varchar(24);not null;uniqueIndex:uk_mall_points_operation,priority:2"`
	UserID    int64  `gorm:"not null;index"`
	Delta     int64  `gorm:"not null"`
	CreatedAt time.Time
}

func (PointsLedger) TableName() string { return "mall_points_ledgers" }

type SagaPayload struct {
	OrderID     string `json:"order_id"`
	UserID      int64  `json:"user_id"`
	RequestID   string `json:"request_id"`
	ProductID   int64  `json:"product_id"`
	Quantity    int64  `json:"quantity"`
	PointsPrice int64  `json:"points_price"`
	TotalPoints int64  `json:"total_points"`
	SagaGID     string `json:"saga_gid"`
	FailAt      string `json:"fail_at,omitempty"`
}
