package application

import (
	"context"
	"errors"
	"strings"
	"time"

	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
	redeemdomain "github.com/dujiao-next/internal/modules/redeem/domain"
)

// StatusResult 查询兑换进度（captcha 仅响应返回）。
type StatusResult struct {
	RedeemOrderID         uint                        `json:"redeem_order_id,omitempty"`
	Status                string                      `json:"status"`
	Stage                 string                      `json:"stage,omitempty"`
	ProductName           string                      `json:"product_name,omitempty"`
	PlanType              string                      `json:"plan_type,omitempty"`
	ServiceCode           string                      `json:"service_code,omitempty"`
	PaymentRegion         string                      `json:"payment_region,omitempty"`
	MaskedCode            string                      `json:"masked_code,omitempty"`
	MaskedContact         string                      `json:"masked_contact,omitempty"`
	MaskedEmail           string                      `json:"masked_email,omitempty"`
	CooldownRemainingSec  int                         `json:"cooldown_remaining_sec,omitempty"`
	Message               string                      `json:"message,omitempty"`
	ErrorCode             string                      `json:"error_code,omitempty"`
	SubscriptionCancelled *bool                       `json:"subscription_cancelled,omitempty"`
	CompletedAt           string                      `json:"completed_at,omitempty"`
	Captcha               *redeemcontract.CaptchaInfo `json:"captcha,omitempty"`
	SubmittedAt           string                      `json:"submitted_at,omitempty"`
}

func (s *Service) Status(ctx context.Context, code string) (*StatusResult, error) {
	if !s.cfg.Enabled {
		return nil, redeemcontract.ErrDisabled
	}
	code = normalizeCode(code)
	if code == "" {
		return nil, redeemcontract.ErrInvalidCode
	}
	secret, err := s.secrets.FindByContent(ctx, code)
	if err != nil {
		return nil, err
	}

	productName := ""
	if s.products != nil {
		if p, perr := s.products.GetByID(ctx, secret.ProductID); perr == nil && p != nil {
			productName = firstNonEmpty(p.TitleZH, p.TitleEN, p.Slug)
		}
	}

	latest, err := s.orders.GetLatestByCardSecretID(ctx, secret.ID)
	if err != nil {
		if errors.Is(err, redeemcontract.ErrNotFound) {
			return &StatusResult{
				Status:      "unused",
				ProductName: productName,
				MaskedCode:  maskCode(code),
				Message:     "尚未提交兑换",
			}, nil
		}
		return nil, err
	}

	out := &StatusResult{
		RedeemOrderID: latest.ID,
		Status:        latest.Status,
		ProductName:   productName,
		PlanType:      latest.PlanType,
		ServiceCode:   latest.ServiceCode,
		PaymentRegion: latest.PaymentRegion,
		MaskedCode:    maskCode(code),
		MaskedContact: maskContact(latest.Contact),
		MaskedEmail:   latest.MaskedEmail,
		ErrorCode:     latest.LastErrorCode,
		Message:       latest.LastErrorMsg,
	}
	if latest.SubmittedAt != nil {
		out.SubmittedAt = latest.SubmittedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
	}
	if latest.CompletedAt != nil {
		out.CompletedAt = latest.CompletedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
	}
	if latest.Status == redeemdomain.StatusCooldown {
		out.CooldownRemainingSec = s.cooldownRemaining(latest.CooldownUntil)
		if out.CooldownRemainingSec == 0 {
			out.Status = "unused"
		}
	}

	// 终态直接返回
	if latest.Status == redeemdomain.StatusSucceeded || latest.Status == redeemdomain.StatusFailed {
		if latest.Status == redeemdomain.StatusSucceeded {
			var cancelled *bool
			// 从 ResultJSON 粗解析 subscriptionCancelled
			if strings.Contains(latest.ResultJSON, `"subscription_cancelled":true`) || strings.Contains(latest.ResultJSON, `"subscriptionCancelled":true`) {
				t := true
				cancelled = &t
			} else if strings.Contains(latest.ResultJSON, `"subscription_cancelled":false`) || strings.Contains(latest.ResultJSON, `"subscriptionCancelled":false`) {
				f := false
				cancelled = &f
			}
			out.SubscriptionCancelled = cancelled
			out.Stage = "done"
			out.Message = "开通成功"
		}
		return out, nil
	}

	if latest.Status == redeemdomain.StatusCooldown {
		return out, nil
	}

	// processing / pending_confirm / locked → 尝试向上游查询
	if latest.Status == redeemdomain.StatusProcessing || latest.Status == redeemdomain.StatusPendingConfirm || latest.Status == redeemdomain.StatusLocked {
		if !s.OrbitConfigured() {
			out.Message = "兑换通道维护中，请稍后再试"
			return out, nil
		}
		captcha, polled, err := s.pollIfDue(ctx, latest)
		if err == nil && polled != nil {
			out.Status = polled.Status
			out.PlanType = firstNonEmpty(polled.PlanType, out.PlanType)
			out.PaymentRegion = firstNonEmpty(polled.PaymentRegion, out.PaymentRegion)
			out.MaskedEmail = firstNonEmpty(polled.MaskedEmail, out.MaskedEmail)
			out.ErrorCode = polled.LastErrorCode
			out.Message = polled.LastErrorMsg
			if polled.CompletedAt != nil {
				out.CompletedAt = polled.CompletedAt.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
			}
			if polled.Status == redeemdomain.StatusSucceeded {
				out.Message = "开通成功"
				out.Stage = "done"
				if strings.Contains(polled.ResultJSON, `"subscriptionCancelled":true`) {
					t := true
					out.SubscriptionCancelled = &t
				} else if strings.Contains(polled.ResultJSON, `"subscriptionCancelled":false`) {
					f := false
					out.SubscriptionCancelled = &f
				}
			}
		}
		if captcha != nil {
			out.Captcha = captcha
			out.Stage = "awaiting_captcha"
			if out.Message == "" {
				out.Message = "需要完成人机验证"
			}
		} else if out.Status == redeemdomain.StatusProcessing {
			out.Stage = "processing"
			if out.Message == "" {
				out.Message = "正在处理，请稍候"
			}
		} else if out.Status == redeemdomain.StatusPendingConfirm {
			out.Stage = "pending_confirm"
			if out.Message == "" {
				out.Message = "结果确认中，请勿重复提交"
			}
		}
	}
	return out, nil
}

func (s *Service) pollIfDue(ctx context.Context, order *redeemdomain.Order) (*redeemcontract.CaptchaInfo, *redeemdomain.Order, error) {
	now := s.now()
	if order.SubmittedAt != nil && now.Sub(*order.SubmittedAt) > 15*time.Minute {
		// 超过 15 分钟仍无终态 → pending_confirm
		if order.Status == redeemdomain.StatusProcessing || order.Status == redeemdomain.StatusLocked {
			order.Status = redeemdomain.StatusPendingConfirm
			order.LastErrorMsg = "处理超时，结果确认中，请稍后查询或联系客服"
			_ = s.orders.Update(ctx, order)
		}
		return nil, order, nil
	}
	if order.NextPollAt != nil && now.Before(*order.NextPollAt) {
		return nil, order, nil
	}
	if order.ClientOrderNo == "" && order.OrbitTaskNo == "" {
		return nil, order, nil
	}

	res, err := s.orbit.GetSubscription(ctx, redeemcontract.GetSubscriptionInput{
		TaskNo:        order.OrbitTaskNo,
		ClientOrderNo: order.ClientOrderNo,
	})
	if err != nil {
		return nil, order, err
	}

	now = s.now()
	order.LastPolledAt = &now
	step := order.PollStep + 1
	order.PollStep = step
	delay := nextPollDelay(step)
	if res.RetryAfterSeconds > 0 {
		d := time.Duration(res.RetryAfterSeconds) * time.Second
		if d > delay {
			delay = d
		}
	}
	next := now.Add(delay)
	order.NextPollAt = &next

	if res.TaskNo != "" {
		order.OrbitTaskNo = res.TaskNo
	}
	if res.PaymentRegion != "" {
		order.PaymentRegion = res.PaymentRegion
	}
	if res.PlanType != "" {
		order.PlanType = res.PlanType
	}
	if res.TargetEmail != "" {
		order.MaskedEmail = maskEmail(res.TargetEmail)
	}

	var captcha *redeemcontract.CaptchaInfo
	if res.Captcha != nil && res.Captcha.Status == "pending" && res.Captcha.URL != "" {
		captcha = &redeemcontract.CaptchaInfo{
			ID:        res.Captcha.ID,
			Status:    res.Captcha.Status,
			URL:       res.Captcha.URL,
			ExpiresAt: res.Captcha.ExpiresAt,
		}
	}

	if res.BusinessCode != 0 && res.SubscriptionSucceeded == nil {
		// 查询失败不改变订单，仅推进轮询
		_ = s.orders.Update(ctx, order)
		return captcha, order, nil
	}

	if res.SubscriptionSucceeded != nil && *res.SubscriptionSucceeded {
		order.Status = redeemdomain.StatusSucceeded
		completed := now
		if res.CompletedAt != "" {
			if t, err := time.Parse(time.RFC3339, res.CompletedAt); err == nil {
				completed = t
			}
		}
		order.CompletedAt = &completed
		order.LastErrorCode = ""
		order.LastErrorMsg = ""
		snap := map[string]any{
			"task_no":                order.OrbitTaskNo,
			"status":                 "succeeded",
			"payment_region":         order.PaymentRegion,
			"plan_type":              order.PlanType,
			"subscriptionCancelled":  res.SubscriptionCancelled,
			"subscription_succeeded": true,
			"masked_email":           order.MaskedEmail,
		}
		order.ResultJSON = encodeResultSnapshot(snap)
		_ = s.orders.Update(ctx, order)
		_ = s.secrets.MarkUsedIfAvailable(ctx, order.CardSecretID, completed)
		return nil, order, nil
	}

	if res.SubscriptionSucceeded != nil && !*res.SubscriptionSucceeded {
		mapped := MapUpstreamError(firstNonEmpty(res.FailureCode, res.FailureMessage))
		order.LastErrorCode = mapped.Code
		order.LastErrorMsg = mapped.Message
		if mapped.Retryable {
			until := now.Add(time.Duration(s.cfg.CooldownSeconds) * time.Second)
			order.Status = redeemdomain.StatusCooldown
			order.CooldownUntil = &until
		} else {
			order.Status = redeemdomain.StatusFailed
		}
		_ = s.orders.Update(ctx, order)
		return nil, order, nil
	}

	// 仍在处理
	if order.Status == redeemdomain.StatusPendingConfirm && res.TaskNo != "" {
		order.Status = redeemdomain.StatusProcessing
	}
	if order.Status == redeemdomain.StatusLocked && res.TaskNo != "" {
		order.Status = redeemdomain.StatusProcessing
	}
	_ = s.orders.Update(ctx, order)
	return captcha, order, nil
}
