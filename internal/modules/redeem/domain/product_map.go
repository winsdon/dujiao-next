package domain

import "time"

// ProductMap 商品到 Orbitcard 套餐的映射（也可由 config 提供）。
type ProductMap struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	ProductID     uint      `gorm:"uniqueIndex;not null" json:"product_id"`
	ServiceCode   string    `gorm:"type:varchar(32);not null" json:"service_code"`
	PlanType      string    `gorm:"type:varchar(64);not null" json:"plan_type"`
	PaymentRegion string    `gorm:"type:varchar(8)" json:"payment_region"`
	MaxServiceFee string    `gorm:"type:varchar(32);not null;default:'1.00'" json:"max_service_fee"`
	Enabled       bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ProductMap) TableName() string { return "redeem_product_maps" }
