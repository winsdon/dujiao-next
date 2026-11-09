package orbitcard

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
)

// Config Orbitcard 客户端配置。
type Config struct {
	BaseURL   string
	APIKey    string
	APISecret string
	CardID    int64
	HTTP      *http.Client
}

// Client 实现 redeemcontract.OrbitcardClient。
type Client struct {
	cfg Config
}

func NewClient(cfg Config) *Client {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 45 * time.Second}
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://orbitcard.cc"
	}
	return &Client{cfg: cfg}
}

func (c *Client) Configured() bool {
	return c != nil &&
		strings.TrimSpace(c.cfg.APIKey) != "" &&
		strings.TrimSpace(c.cfg.APISecret) != "" &&
		c.cfg.CardID > 0
}

func (c *Client) CreateSubscription(ctx context.Context, in redeemcontract.CreateSubscriptionInput) (*redeemcontract.CreateSubscriptionResult, error) {
	if !c.Configured() {
		return nil, redeemcontract.ErrMaintenance
	}
	cardID := in.CardID
	if cardID <= 0 {
		cardID = c.cfg.CardID
	}

	// 用手写拼接保证 session_json 以原始字节嵌入，避免二次序列化。
	parts := []string{
		`"client_order_no":` + jsonString(in.ClientOrderNo),
		`"plan_type":` + jsonString(in.PlanType),
		`"card_id":` + strconv.FormatInt(cardID, 10),
		`"session_json":` + string(in.SessionJSON),
		`"authorized":` + strconv.FormatBool(in.Authorized),
	}
	if sc := strings.TrimSpace(in.ServiceCode); sc != "" && sc != "chatgpt" {
		parts = append([]string{`"service_code":` + jsonString(sc)}, parts...)
	}
	if r := strings.TrimSpace(in.PaymentRegion); r != "" {
		parts = append(parts, `"payment_region":`+jsonString(r))
	}
	if fee := strings.TrimSpace(in.MaxServiceFee); fee != "" {
		parts = append(parts, `"max_service_fee":`+jsonString(fee))
	}
	raw := []byte("{" + strings.Join(parts, ",") + "}")

	respBody, httpStatus, err := c.doSigned(ctx, "/api/open/v1/createSubscription", in.IdempotencyKey, raw)
	if err != nil {
		return &redeemcontract.CreateSubscriptionResult{
			HTTPStatus:        httpStatus,
			BusinessCode:      59018,
			Message:           "request failed",
			SubmissionOutcome: "unknown",
		}, nil
	}
	return parseCreateResult(httpStatus, respBody), nil
}

func (c *Client) GetSubscription(ctx context.Context, in redeemcontract.GetSubscriptionInput) (*redeemcontract.GetSubscriptionResult, error) {
	if !c.Configured() {
		return nil, redeemcontract.ErrMaintenance
	}
	var raw []byte
	switch {
	case strings.TrimSpace(in.TaskNo) != "":
		raw = []byte(`{"task_no":` + jsonString(in.TaskNo) + `}`)
	case strings.TrimSpace(in.ClientOrderNo) != "":
		raw = []byte(`{"client_order_no":` + jsonString(in.ClientOrderNo) + `}`)
	default:
		return nil, fmt.Errorf("task_no or client_order_no required")
	}
	respBody, httpStatus, err := c.doSigned(ctx, "/api/open/v1/getSubscription", "", raw)
	if err != nil {
		return &redeemcontract.GetSubscriptionResult{
			HTTPStatus:   httpStatus,
			BusinessCode: 59018,
			Message:      "query failed",
		}, nil
	}
	return parseGetResult(httpStatus, respBody), nil
}

func (c *Client) doSigned(ctx context.Context, path, idempotencyKey string, rawBody []byte) ([]byte, int, error) {
	u, err := url.Parse(c.cfg.BaseURL + path)
	if err != nil {
		return nil, 0, err
	}
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	nonce, err := randomNonce()
	if err != nil {
		return nil, 0, err
	}
	bodyHash := BodySHA256Hex(rawBody)
	sig := SignCanonical(c.cfg.APISecret, c.cfg.APIKey, timestamp, nonce, "POST", u.Host, path, "", bodyHash, idempotencyKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(rawBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", sig)
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.cfg.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return body, resp.StatusCode, nil
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func randomNonce() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type envelope struct {
	Code    int             `json:"code"`
	Msg     string          `json:"msg"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func parseCreateResult(httpStatus int, body []byte) *redeemcontract.CreateSubscriptionResult {
	out := &redeemcontract.CreateSubscriptionResult{HTTPStatus: httpStatus}
	if httpStatus == 401 || httpStatus == 403 {
		out.BusinessCode = httpStatus
		out.Message = "auth"
		out.SubmissionOutcome = "rejected"
		return out
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		out.BusinessCode = 59018
		out.SubmissionOutcome = "unknown"
		out.Message = "invalid response"
		return out
	}
	out.BusinessCode = env.Code
	out.Message = firstNonEmpty(env.Message, env.Msg)

	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	if data == nil {
		data = map[string]any{}
	}
	out.TaskNo = asString(data["task_no"])
	out.ClientOrderNo = asString(data["client_order_no"])
	out.Status = asString(data["status"])
	out.UpstreamStatus = asString(data["upstream_status"])
	out.ErrorCode = asString(data["error_code"])
	out.UpstreamCode = asString(data["upstream_code"])
	if v, ok := data["subscription_succeeded"].(bool); ok {
		out.SubscriptionOK = &v
	}
	out.RawSafeSummary = sanitizeCreateData(data)

	// 59018 + request_error.submission_outcome
	var reqErr struct {
		RequestError *struct {
			SubmissionOutcome string `json:"submission_outcome"`
			Code              string `json:"code"`
		} `json:"request_error"`
	}
	_ = json.Unmarshal(env.Data, &reqErr)
	if reqErr.RequestError != nil {
		// Only an explicit rejection proves that a new submission is safe.
		out.SubmissionOutcome = "unknown"
		if strings.EqualFold(strings.TrimSpace(reqErr.RequestError.SubmissionOutcome), "rejected") {
			out.SubmissionOutcome = "rejected"
		}
		if out.ErrorCode == "" {
			out.ErrorCode = reqErr.RequestError.Code
		}
		return out
	}
	if httpStatus >= 500 {
		out.SubmissionOutcome = "unknown"
		return out
	}
	if env.Code == 0 && out.TaskNo != "" {
		out.SubmissionOutcome = "accepted"
		return out
	}
	switch env.Code {
	case 40027, 59004, 59002, 59005, 59015, 59016, 59001, 59013, 59007:
		out.SubmissionOutcome = "rejected"
	default:
		out.SubmissionOutcome = "unknown"
	}
	return out
}

func parseGetResult(httpStatus int, body []byte) *redeemcontract.GetSubscriptionResult {
	out := &redeemcontract.GetSubscriptionResult{HTTPStatus: httpStatus}
	if httpStatus == 401 || httpStatus == 403 {
		out.BusinessCode = httpStatus
		out.Message = "auth"
		return out
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		out.BusinessCode = 59018
		out.Message = "invalid response"
		return out
	}
	out.BusinessCode = env.Code
	out.Message = firstNonEmpty(env.Message, env.Msg)

	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	if data == nil {
		return out
	}
	out.TaskNo = firstNonEmpty(asString(data["task_no"]), asString(data["orderId"]))
	out.ClientOrderNo = asString(data["client_order_no"])
	out.Status = asString(data["status"])
	out.TaskStatus = asString(data["task_status"])
	out.Stage = asString(data["stage"])
	out.UpstreamStatus = firstNonEmpty(asString(data["upstream_status"]), asString(data["status"]))
	out.TargetEmail = firstNonEmpty(asString(data["targetEmail"]), asString(data["target_email"]))
	out.PaymentRegion = firstNonEmpty(asString(data["paymentRegion"]), asString(data["payment_region"]))
	out.PlanType = asString(data["plan_type"])
	out.CompletedAt = asString(data["completedAt"])
	out.FailureCode = firstNonEmpty(asString(data["failureCode"]), asString(data["error_code"]))
	out.FailureMessage = asString(data["failureMessage"])
	if v, ok := data["subscription_succeeded"].(bool); ok {
		out.SubscriptionSucceeded = &v
	}
	if v, ok := data["subscriptionCancelled"].(bool); ok {
		out.SubscriptionCancelled = &v
	}
	if capRaw, ok := data["captcha"]; ok && capRaw != nil {
		if m, ok := capRaw.(map[string]any); ok {
			out.Captcha = &redeemcontract.CaptchaInfo{
				ID:        asString(m["id"]),
				Status:    asString(m["status"]),
				URL:       asString(m["url"]),
				ExpiresAt: asString(m["expires_at"]),
			}
		}
	}
	if re, ok := data["request_error"].(map[string]any); ok && re != nil {
		if v, ok := re["retry_after_seconds"].(float64); ok {
			out.RetryAfterSeconds = int(v)
		}
	}
	return out
}

func sanitizeCreateData(data map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"task_no", "client_order_no", "plan_type", "payment_region", "status", "upstream_status", "subscription_succeeded", "error_code", "fee_status", "billing_policy", "stage"} {
		if v, ok := data[k]; ok {
			out[k] = v
		}
	}
	return out
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
