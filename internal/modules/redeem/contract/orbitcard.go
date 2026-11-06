package contract

import "context"

// CreateSubscriptionInput 创建订阅入参（session 仅内存传递）。
type CreateSubscriptionInput struct {
	ClientOrderNo  string
	IdempotencyKey string
	ServiceCode    string
	PlanType       string
	PaymentRegion  string
	CardID         int64
	SessionJSON    []byte // 已是合法 JSON 值（对象或字符串）的原始字节
	Authorized     bool
	MaxServiceFee  string
}

// CreateSubscriptionResult 创建受理结果。
type CreateSubscriptionResult struct {
	HTTPStatus        int
	BusinessCode      int
	Message           string
	SubmissionOutcome string // accepted / rejected / unknown
	TaskNo            string
	ClientOrderNo     string
	Status            string
	UpstreamStatus    string
	SubscriptionOK    *bool
	ErrorCode         string
	UpstreamCode      string
	RawSafeSummary    map[string]any
}

// GetSubscriptionInput 查询入参。
type GetSubscriptionInput struct {
	TaskNo        string
	ClientOrderNo string
}

// CaptchaInfo 人机验证（仅响应，不落库）。
type CaptchaInfo struct {
	ID        string
	Status    string
	URL       string
	ExpiresAt string
}

// GetSubscriptionResult 查询结果（不含 session）。
type GetSubscriptionResult struct {
	HTTPStatus            int
	BusinessCode          int
	Message               string
	TaskNo                string
	ClientOrderNo         string
	Status                string
	TaskStatus            string
	Stage                 string
	UpstreamStatus        string
	SubscriptionSucceeded *bool
	SubscriptionCancelled *bool
	TargetEmail           string
	PaymentRegion         string
	PlanType              string
	CompletedAt           string
	FailureCode           string
	FailureMessage        string
	Captcha               *CaptchaInfo
	RetryAfterSeconds     int
}

// OrbitcardClient Orbitcard 开放 API。
type OrbitcardClient interface {
	Configured() bool
	CreateSubscription(ctx context.Context, in CreateSubscriptionInput) (*CreateSubscriptionResult, error)
	GetSubscription(ctx context.Context, in GetSubscriptionInput) (*GetSubscriptionResult, error)
}
