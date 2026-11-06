package application

import (
	"context"
	"errors"
	"strings"
	"time"

	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
	redeemdomain "github.com/dujiao-next/internal/modules/redeem/domain"
)

// VerifyResult 校验兑换码结果。
type VerifyResult struct {
	Valid                bool   `json:"valid"`
	ProductID            uint   `json:"product_id,omitempty"`
	ProductName          string `json:"product_name,omitempty"`
	PlanType             string `json:"plan_type,omitempty"`
	ServiceCode          string `json:"service_code,omitempty"`
	PaymentRegion        string `json:"payment_region,omitempty"`
	Status               string `json:"status"`
	CooldownRemainingSec int    `json:"cooldown_remaining_sec"`
	MaskedCode           string `json:"masked_code,omitempty"`
	Message              string `json:"message,omitempty"`
	RedeemOrderID        uint   `json:"redeem_order_id,omitempty"`
}

func (s *Service) Verify(ctx context.Context, code string) (*VerifyResult, error) {
	if !s.cfg.Enabled {
		return nil, redeemcontract.ErrDisabled
	}
	code = normalizeCode(code)
	if code == "" {
		return nil, redeemcontract.ErrInvalidCode
	}
	secret, err := s.secrets.FindByContent(ctx, code)
	if err != nil {
		if errors.Is(err, redeemcontract.ErrCodeNotFound) {
			return &VerifyResult{Valid: false, Status: "invalid", Message: "兑换码无效或不存在"}, nil
		}
		return nil, err
	}

	productName := ""
	if s.products != nil {
		if p, perr := s.products.GetByID(ctx, secret.ProductID); perr == nil && p != nil {
			productName = firstNonEmpty(p.TitleZH, p.TitleEN, p.Slug)
		}
	}

	pm, mapErr := s.resolveMap(ctx, secret.ProductID)
	planType, serviceCode, region := "", "", ""
	if mapErr == nil && pm != nil {
		planType, serviceCode, region = pm.PlanType, pm.ServiceCode, pm.PaymentRegion
	}

	out := &VerifyResult{
		Valid:         true,
		ProductID:     secret.ProductID,
		ProductName:   productName,
		PlanType:      planType,
		ServiceCode:   serviceCode,
		PaymentRegion: region,
		MaskedCode:    maskCode(code),
		Status:        "unused",
	}

	if succeeded, err := s.orders.GetSucceededByCardSecretID(ctx, secret.ID); err == nil && succeeded != nil {
		out.Status = redeemdomain.StatusSucceeded
		out.RedeemOrderID = succeeded.ID
		out.Message = "该卡密已兑换成功"
		return out, nil
	}

	latest, err := s.orders.GetLatestByCardSecretID(ctx, secret.ID)
	if err != nil {
		if errors.Is(err, redeemcontract.ErrNotFound) {
			if mapErr != nil {
				out.Valid = false
				out.Status = "unmapped"
				out.Message = "该商品暂未开通兑换通道，请联系客服"
				return out, nil
			}
			return out, nil
		}
		return nil, err
	}
	out.RedeemOrderID = latest.ID
	out.Status = latest.Status

	switch latest.Status {
	case redeemdomain.StatusCooldown:
		remain := 0
		if latest.CooldownUntil != nil {
			remain = int(latest.CooldownUntil.Sub(s.now()).Seconds())
			if remain < 0 {
				remain = 0
				out.Status = "unused"
			}
		}
		out.CooldownRemainingSec = remain
		if remain > 0 {
			out.Message = "冷却中，请稍后再试"
		}
	case redeemdomain.StatusLocked, redeemdomain.StatusProcessing:
		out.Message = "开通处理中"
	case redeemdomain.StatusPendingConfirm:
		out.Message = "结果确认中，请勿重复提交"
	case redeemdomain.StatusFailed:
		out.Status = "unused"
		out.Message = "上次开通未成功，可重新填写"
	case redeemdomain.StatusSucceeded:
		out.Message = "该卡密已兑换成功"
	}
	return out, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (s *Service) cooldownRemaining(until *time.Time) int {
	if until == nil {
		return 0
	}
	remain := int(until.Sub(s.now()).Seconds())
	if remain < 0 {
		return 0
	}
	return remain
}
