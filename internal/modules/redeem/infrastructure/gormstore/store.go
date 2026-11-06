package gormstore

import (
	"context"
	"errors"
	"strings"
	"time"

	cardsecretdomain "github.com/dujiao-next/internal/modules/cardsecret/domain"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
	redeemdomain "github.com/dujiao-next/internal/modules/redeem/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) with(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx)
}

func (s *Store) Create(ctx context.Context, order *redeemdomain.Order) error {
	return s.with(ctx).Create(order).Error
}

func (s *Store) Update(ctx context.Context, order *redeemdomain.Order) error {
	return s.with(ctx).Save(order).Error
}

func (s *Store) GetByID(ctx context.Context, id uint) (*redeemdomain.Order, error) {
	var o redeemdomain.Order
	if err := s.with(ctx).First(&o, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, redeemcontract.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (s *Store) GetLatestByCardSecretID(ctx context.Context, cardSecretID uint) (*redeemdomain.Order, error) {
	var o redeemdomain.Order
	err := s.with(ctx).Where("card_secret_id = ?", cardSecretID).Order("id DESC").First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, redeemcontract.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (s *Store) GetSucceededByCardSecretID(ctx context.Context, cardSecretID uint) (*redeemdomain.Order, error) {
	var o redeemdomain.Order
	err := s.with(ctx).Where("card_secret_id = ? AND status = ?", cardSecretID, redeemdomain.StatusSucceeded).Order("id DESC").First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, redeemcontract.ErrNotFound
		}
		return nil, err
	}
	return &o, nil
}

func (s *Store) Transaction(ctx context.Context, fn func(tx redeemcontract.OrderRepository) error) error {
	return s.with(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&Store{db: tx})
	})
}

func (s *Store) GetByProductID(ctx context.Context, productID uint) (*redeemdomain.ProductMap, error) {
	var m redeemdomain.ProductMap
	err := s.with(ctx).Where("product_id = ?", productID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, redeemcontract.ErrMapMissing
		}
		return nil, err
	}
	return &m, nil
}

func (s *Store) Upsert(ctx context.Context, m *redeemdomain.ProductMap) error {
	return s.with(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "product_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"service_code", "plan_type", "payment_region", "max_service_fee", "enabled", "updated_at"}),
	}).Create(m).Error
}

func (s *Store) List(ctx context.Context) ([]redeemdomain.ProductMap, error) {
	var items []redeemdomain.ProductMap
	if err := s.with(ctx).Order("product_id ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) FindByContent(ctx context.Context, content string) (*redeemcontract.CardSecretView, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, redeemcontract.ErrCodeNotFound
	}
	var sec cardsecretdomain.Secret
	err := s.with(ctx).Where("secret = ? AND deleted_at IS NULL", content).First(&sec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 兼容大小写不敏感查找
			err = s.with(ctx).Where("lower(secret) = lower(?) AND deleted_at IS NULL", content).First(&sec).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, redeemcontract.ErrCodeNotFound
			}
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &redeemcontract.CardSecretView{
		ID:        sec.ID,
		ProductID: sec.ProductID,
		Secret:    sec.Secret,
		Status:    sec.Status,
	}, nil
}

func (s *Store) MarkUsedIfAvailable(ctx context.Context, id uint, usedAt time.Time) error {
	return s.with(ctx).Model(&cardsecretdomain.Secret{}).
		Where("id = ? AND status = ?", id, cardsecretdomain.StatusAvailable).
		Updates(map[string]any{
			"status":     cardsecretdomain.StatusUsed,
			"used_at":    usedAt,
			"updated_at": usedAt,
		}).Error
}

func (s *Store) GetProductByID(ctx context.Context, id uint) (*redeemcontract.ProductView, error) {
	var p productdomain.Product
	if err := s.with(ctx).First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, redeemcontract.ErrNotFound
		}
		return nil, err
	}
	zh, en := localizeTitle(map[string]interface{}(p.TitleJSON))
	return &redeemcontract.ProductView{
		ID:       p.ID,
		Slug:     p.Slug,
		TitleZH:  zh,
		TitleEN:  en,
		IsActive: p.IsActive,
	}, nil
}

func localizeTitle(raw map[string]interface{}) (zh, en string) {
	if raw == nil {
		return "", ""
	}
	as := func(k string) string {
		v, ok := raw[k]
		if !ok || v == nil {
			return ""
		}
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	}
	zh = first(as("zh-CN"), as("zh"), as("zh_CN"))
	en = first(as("en-US"), as("en"), as("en_US"))
	if zh == "" {
		zh = en
	}
	if en == "" {
		en = zh
	}
	return zh, en
}

func first(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Ensure Store 同时满足多个端口。
var (
	_ redeemcontract.OrderRepository      = (*Store)(nil)
	_ redeemcontract.ProductMapRepository = (*Store)(nil)
	_ redeemcontract.CardSecretLookup     = (*Store)(nil)
)

// ProductLookupAdapter wraps store for product lookup port.
type ProductLookupAdapter struct{ *Store }

func (a ProductLookupAdapter) GetByID(ctx context.Context, id uint) (*redeemcontract.ProductView, error) {
	return a.Store.GetProductByID(ctx, id)
}
