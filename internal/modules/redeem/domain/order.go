package domain

import "time"

const (
	StatusCooldown       = "cooldown"
	StatusLocked         = "locked"
	StatusProcessing     = "processing"
	StatusPendingConfirm = "pending_confirm"
	StatusSucceeded      = "succeeded"
	StatusFailed         = "failed"
)

// Order 卡密兑换订单（Session 永不入库）。
type Order struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	CardSecretID   uint       `gorm:"not null;index:idx_redeem_orders_secret" json:"card_secret_id"`
	CodeSnapshot   string     `gorm:"type:text;not null" json:"-"`
	ProductID      uint       `gorm:"not null;index" json:"product_id"`
	Contact        string     `gorm:"type:varchar(255);not null" json:"contact"`
	Status         string     `gorm:"type:varchar(32);not null;index" json:"status"`
	Attempt        int        `gorm:"not null;default:1" json:"attempt"`
	ClientOrderNo  string     `gorm:"type:varchar(128);index" json:"client_order_no"`
	IdempotencyKey string     `gorm:"type:varchar(128)" json:"idempotency_key"`
	OrbitTaskNo    string     `gorm:"type:varchar(128);index" json:"orbit_task_no"`
	CooldownUntil  *time.Time `gorm:"index" json:"cooldown_until"`
	LastErrorCode  string     `gorm:"type:varchar(64)" json:"last_error_code"`
	LastErrorMsg   string     `gorm:"type:varchar(512)" json:"last_error_msg"`
	ResultJSON     string     `gorm:"type:text" json:"-"` // 安全快照，不含 session
	ServiceCode    string     `gorm:"type:varchar(32)" json:"service_code"`
	PlanType       string     `gorm:"type:varchar(64)" json:"plan_type"`
	PaymentRegion  string     `gorm:"type:varchar(8)" json:"payment_region"`
	MaskedEmail    string     `gorm:"type:varchar(255)" json:"masked_email"`
	LastPolledAt   *time.Time `json:"last_polled_at"`
	NextPollAt     *time.Time `json:"next_poll_at"`
	PollStep       int        `gorm:"not null;default:0" json:"poll_step"`
	SubmittedAt    *time.Time `json:"submitted_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (Order) TableName() string { return "redeem_orders" }
