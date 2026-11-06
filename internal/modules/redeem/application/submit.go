package application

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
	redeemdomain "github.com/dujiao-next/internal/modules/redeem/domain"
)

// SubmitInput 提交兑换。
type SubmitInput struct {
	Code       string
	Contact    string
	Credential string // 原始凭据文本；不落库
	Authorized bool
}

// SubmitResult 提交结果。
type SubmitResult struct {
	RedeemOrderID uint   `json:"redeem_order_id"`
	Status        string `json:"status"`
	Message       string `json:"message,omitempty"`
	CooldownSec   int    `json:"cooldown_remaining_sec,omitempty"`
}

func (s *Service) Submit(ctx context.Context, in SubmitInput) (*SubmitResult, error) {
	if !s.cfg.Enabled {
		return nil, redeemcontract.ErrDisabled
	}
	if !s.OrbitConfigured() {
		return nil, redeemcontract.ErrMaintenance
	}
	if !in.Authorized {
		return nil, redeemcontract.ErrNotAuthorized
	}
	code := normalizeCode(in.Code)
	contact := strings.TrimSpace(in.Contact)
	if code == "" {
		return nil, redeemcontract.ErrInvalidCode
	}
	if !validContact(contact) {
		return nil, redeemcontract.ErrInvalidContact
	}
	cred := strings.TrimSpace(in.Credential)
	if cred == "" || utf8.RuneCountInString(cred) > 64*1024 {
		return nil, redeemcontract.ErrInvalidCredential
	}

	secret, err := s.secrets.FindByContent(ctx, code)
	if err != nil {
		return nil, err
	}
	if _, err := s.orders.GetSucceededByCardSecretID(ctx, secret.ID); err == nil {
		return nil, redeemcontract.ErrAlreadySucceeded
	}

	pm, err := s.resolveMap(ctx, secret.ProductID)
	if err != nil {
		return nil, redeemcontract.ErrMapMissing
	}

	sessionJSON, maskedFromCred, err := prepareSessionJSON(pm.ServiceCode, cred)
	if err != nil {
		return nil, redeemcontract.ErrInvalidCredential
	}

	now := s.now()
	var order *redeemdomain.Order

	err = s.orders.Transaction(ctx, func(orderStore redeemcontract.OrderRepository) error {
		latest, lerr := orderStore.GetLatestByCardSecretID(ctx, secret.ID)
		attempt := 1
		if lerr == nil && latest != nil {
			switch latest.Status {
			case redeemdomain.StatusSucceeded:
				return redeemcontract.ErrAlreadySucceeded
			case redeemdomain.StatusLocked, redeemdomain.StatusProcessing:
				return redeemcontract.ErrBusy
			case redeemdomain.StatusPendingConfirm:
				return redeemcontract.ErrPendingConfirm
			case redeemdomain.StatusCooldown:
				if latest.CooldownUntil != nil && latest.CooldownUntil.After(now) {
					return redeemcontract.ErrInCooldown
				}
				attempt = latest.Attempt + 1
			case redeemdomain.StatusFailed:
				attempt = latest.Attempt + 1
			default:
				attempt = latest.Attempt + 1
			}
		} else if lerr != nil && !errors.Is(lerr, redeemcontract.ErrNotFound) {
			return lerr
		}

		order = &redeemdomain.Order{
			CardSecretID:   secret.ID,
			CodeSnapshot:   hashCode(code),
			ProductID:      secret.ProductID,
			Contact:        contact,
			Status:         redeemdomain.StatusLocked,
			Attempt:        attempt,
			ClientOrderNo:  s.clientOrderNo(secret.ID, attempt),
			IdempotencyKey: s.idempotencyKey(secret.ID, attempt),
			ServiceCode:    pm.ServiceCode,
			PlanType:       pm.PlanType,
			PaymentRegion:  pm.PaymentRegion,
			MaskedEmail:    maskedFromCred,
			SubmittedAt:    &now,
		}
		return orderStore.Create(ctx, order)
	})
	if err != nil {
		return nil, err
	}

	createRes, cerr := s.orbit.CreateSubscription(ctx, redeemcontract.CreateSubscriptionInput{
		ClientOrderNo:  order.ClientOrderNo,
		IdempotencyKey: order.IdempotencyKey,
		ServiceCode:    pm.ServiceCode,
		PlanType:       pm.PlanType,
		PaymentRegion:  pm.PaymentRegion,
		SessionJSON:    sessionJSON,
		Authorized:     true,
		MaxServiceFee:  pm.MaxServiceFee,
	})
	// credential 变量在此结束，不再引用

	if cerr != nil {
		if errors.Is(cerr, redeemcontract.ErrMaintenance) {
			_ = s.failWithCooldown(ctx, order, "maintenance", "兑换通道维护中，请稍后再试", true)
			return nil, redeemcontract.ErrMaintenance
		}
		_ = s.markPendingConfirm(ctx, order, "request_error", "结果确认中，请勿重复提交")
		return &SubmitResult{RedeemOrderID: order.ID, Status: redeemdomain.StatusPendingConfirm, Message: "结果确认中，请勿重复提交"}, nil
	}

	switch createRes.SubmissionOutcome {
	case "accepted":
		order.Status = redeemdomain.StatusProcessing
		order.OrbitTaskNo = createRes.TaskNo
		order.ResultJSON = encodeResultSnapshot(createRes.RawSafeSummary)
		order.LastErrorCode = ""
		order.LastErrorMsg = ""
		delay := nextPollDelay(0)
		next := now.Add(delay)
		order.PollStep = 0
		order.NextPollAt = &next
		_ = s.orders.Update(ctx, order)
		return &SubmitResult{RedeemOrderID: order.ID, Status: order.Status, Message: "已受理，正在开通"}, nil

	case "rejected":
		errCode := createRes.ErrorCode
		if errCode == "" {
			errCode = createRes.UpstreamCode
		}
		if errCode == "" {
			errCode = itoa(createRes.BusinessCode)
		}
		mapped := MapUpstreamError(errCode)
		_ = s.failWithCooldown(ctx, order, mapped.Code, mapped.Message, mapped.Retryable)
		return &SubmitResult{
			RedeemOrderID: order.ID,
			Status:        redeemdomain.StatusCooldown,
			Message:       mapped.Message,
			CooldownSec:   s.cfg.CooldownSeconds,
		}, nil

	default: // unknown
		if createRes.TaskNo != "" {
			order.OrbitTaskNo = createRes.TaskNo
		}
		ec := createRes.ErrorCode
		if ec == "" {
			ec = "59018"
		}
		_ = s.markPendingConfirm(ctx, order, ec, "结果确认中，请勿重复提交")
		return &SubmitResult{RedeemOrderID: order.ID, Status: redeemdomain.StatusPendingConfirm, Message: "结果确认中，请勿重复提交"}, nil
	}
}

func (s *Service) failWithCooldown(ctx context.Context, order *redeemdomain.Order, code, msg string, retryable bool) error {
	now := s.now()
	order.LastErrorCode = code
	order.LastErrorMsg = msg
	if retryable {
		until := now.Add(time.Duration(s.cfg.CooldownSeconds) * time.Second)
		order.Status = redeemdomain.StatusCooldown
		order.CooldownUntil = &until
	} else {
		order.Status = redeemdomain.StatusFailed
	}
	return s.orders.Update(ctx, order)
}

func (s *Service) markPendingConfirm(ctx context.Context, order *redeemdomain.Order, code, msg string) error {
	order.Status = redeemdomain.StatusPendingConfirm
	order.LastErrorCode = code
	order.LastErrorMsg = msg
	return s.orders.Update(ctx, order)
}

func validContact(contact string) bool {
	if contact == "" || utf8.RuneCountInString(contact) > 200 {
		return false
	}
	if strings.Contains(contact, "@") {
		parts := strings.SplitN(contact, "@", 2)
		return len(parts[0]) > 0 && strings.Contains(parts[1], ".")
	}
	// Telegram: @name or t.me/ or plain handle
	if strings.HasPrefix(contact, "@") && len(contact) >= 4 {
		return true
	}
	if strings.Contains(strings.ToLower(contact), "t.me/") {
		return true
	}
	return utf8.RuneCountInString(contact) >= 3
}

// prepareSessionJSON 返回可作为 JSON 值嵌入的原始字节，以及可能解析出的打码邮箱。
func prepareSessionJSON(serviceCode, cred string) ([]byte, string, error) {
	cred = strings.TrimSpace(cred)
	serviceCode = strings.ToLower(strings.TrimSpace(serviceCode))

	if serviceCode == "grok" {
		// 若是 JSON 对象则原样使用；否则作为 JSON 字符串。
		if json.Valid([]byte(cred)) && (strings.HasPrefix(cred, "{") || strings.HasPrefix(cred, "[")) {
			return []byte(cred), "", nil
		}
		b, err := json.Marshal(cred)
		if err != nil {
			return nil, "", err
		}
		return b, "", nil
	}

	// ChatGPT: 必须是 JSON 对象
	if !json.Valid([]byte(cred)) || !strings.HasPrefix(cred, "{") {
		return nil, "", redeemcontract.ErrInvalidCredential
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(cred), &obj); err != nil {
		return nil, "", err
	}
	if _, ok := obj["accessToken"]; !ok {
		return nil, "", redeemcontract.ErrInvalidCredential
	}
	masked := ""
	if user, ok := obj["user"].(map[string]any); ok {
		if email, ok := user["email"].(string); ok {
			masked = maskEmail(email)
		}
	}
	// 使用原始字节，避免重排字段
	return []byte(cred), masked, nil
}

func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}
