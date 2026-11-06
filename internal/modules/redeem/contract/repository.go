package contract

import (
	"context"
	"time"

	redeemdomain "github.com/dujiao-next/internal/modules/redeem/domain"
)

// OrderRepository 兑换订单持久化。
type OrderRepository interface {
	Create(ctx context.Context, order *redeemdomain.Order) error
	Update(ctx context.Context, order *redeemdomain.Order) error
	GetByID(ctx context.Context, id uint) (*redeemdomain.Order, error)
	GetLatestByCardSecretID(ctx context.Context, cardSecretID uint) (*redeemdomain.Order, error)
	GetSucceededByCardSecretID(ctx context.Context, cardSecretID uint) (*redeemdomain.Order, error)
	Transaction(ctx context.Context, fn func(tx OrderRepository) error) error
}

// ProductMapRepository 商品映射持久化。
type ProductMapRepository interface {
	GetByProductID(ctx context.Context, productID uint) (*redeemdomain.ProductMap, error)
	Upsert(ctx context.Context, m *redeemdomain.ProductMap) error
	List(ctx context.Context) ([]redeemdomain.ProductMap, error)
}

// CardSecretView 卡密只读视图（避免依赖 cardsecret 完整领域）。
type CardSecretView struct {
	ID        uint
	ProductID uint
	Secret    string
	Status    string
}

// CardSecretLookup 按内容查找卡密。
type CardSecretLookup interface {
	FindByContent(ctx context.Context, content string) (*CardSecretView, error)
	MarkUsedIfAvailable(ctx context.Context, id uint, usedAt time.Time) error
}

// ProductView 商品只读视图。
type ProductView struct {
	ID       uint
	Slug     string
	TitleZH  string
	TitleEN  string
	IsActive bool
}

// ProductLookup 读取商品展示信息。
type ProductLookup interface {
	GetByID(ctx context.Context, id uint) (*ProductView, error)
}
