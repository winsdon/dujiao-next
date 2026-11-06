package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
	redeemdomain "github.com/dujiao-next/internal/modules/redeem/domain"
)

// Config 兑换模块运行时配置。
type Config struct {
	Enabled         bool
	CooldownSeconds int
	ProductMaps     map[uint]ProductMapConfig
}

// ProductMapConfig 来自 config.yml 的商品映射。
type ProductMapConfig struct {
	ServiceCode   string
	PlanType      string
	PaymentRegion string
	MaxServiceFee string
	Enabled       bool
	Disabled      bool
}

// Service 兑换应用服务。
type Service struct {
	orders   redeemcontract.OrderRepository
	maps     redeemcontract.ProductMapRepository
	secrets  redeemcontract.CardSecretLookup
	products redeemcontract.ProductLookup
	orbit    redeemcontract.OrbitcardClient
	cfg      Config
	now      func() time.Time
}

func NewService(
	orders redeemcontract.OrderRepository,
	maps redeemcontract.ProductMapRepository,
	secrets redeemcontract.CardSecretLookup,
	products redeemcontract.ProductLookup,
	orbit redeemcontract.OrbitcardClient,
	cfg Config,
) *Service {
	if cfg.CooldownSeconds <= 0 {
		cfg.CooldownSeconds = 60
	}
	if cfg.ProductMaps == nil {
		cfg.ProductMaps = map[uint]ProductMapConfig{}
	}
	return &Service{
		orders:   orders,
		maps:     maps,
		secrets:  secrets,
		products: products,
		orbit:    orbit,
		cfg:      cfg,
		now:      time.Now,
	}
}

func (s *Service) Enabled() bool {
	return s.cfg.Enabled
}

func (s *Service) OrbitConfigured() bool {
	return s.orbit != nil && s.orbit.Configured()
}

func normalizeCode(code string) string {
	return strings.TrimSpace(code)
}

func maskContact(contact string) string {
	contact = strings.TrimSpace(contact)
	if contact == "" {
		return ""
	}
	if strings.Contains(contact, "@") {
		parts := strings.SplitN(contact, "@", 2)
		local := parts[0]
		if len(local) <= 1 {
			return "*@" + parts[1]
		}
		return string(local[0]) + "***@" + parts[1]
	}
	if len(contact) <= 3 {
		return "***"
	}
	return contact[:2] + "***" + contact[len(contact)-1:]
}

func maskEmail(email string) string {
	return maskContact(email)
}

func maskCode(code string) string {
	code = strings.TrimSpace(code)
	if len(code) <= 8 {
		return "****"
	}
	return code[:4] + "…" + code[len(code)-4:]
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func (s *Service) resolveMap(ctx context.Context, productID uint) (*redeemdomain.ProductMap, error) {
	if m, ok := s.cfg.ProductMaps[productID]; ok {
		if strings.TrimSpace(m.PlanType) == "" {
			return nil, redeemcontract.ErrMapMissing
		}
		enabled := m.Enabled
		// yaml 布尔零值默认为 false；只要配置了 plan_type 且未显式 Disabled 则启用
		if !m.Disabled {
			enabled = true
		} else {
			enabled = false
		}
		if !enabled {
			return nil, redeemcontract.ErrMapMissing
		}
		return &redeemdomain.ProductMap{
			ProductID:     productID,
			ServiceCode:   defaultStr(m.ServiceCode, "chatgpt"),
			PlanType:      m.PlanType,
			PaymentRegion: m.PaymentRegion,
			MaxServiceFee: defaultStr(m.MaxServiceFee, "1.00"),
			Enabled:       true,
		}, nil
	}
	if s.maps == nil {
		return nil, redeemcontract.ErrMapMissing
	}
	m, err := s.maps.GetByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if m == nil || !m.Enabled || m.PlanType == "" {
		return nil, redeemcontract.ErrMapMissing
	}
	return m, nil
}

func defaultStr(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func encodeResultSnapshot(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *Service) clientOrderNo(secretID uint, attempt int) string {
	return fmt.Sprintf("RD-%d-%d", secretID, attempt)
}

func (s *Service) idempotencyKey(secretID uint, attempt int) string {
	return fmt.Sprintf("redeem:%d:%d", secretID, attempt)
}

var pollIntervals = []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second}

func nextPollDelay(step int) time.Duration {
	if step < 0 {
		step = 0
	}
	if step >= len(pollIntervals) {
		return pollIntervals[len(pollIntervals)-1]
	}
	return pollIntervals[step]
}
