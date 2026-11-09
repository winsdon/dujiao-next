package application

import (
	"context"
	"errors"
	"testing"

	redeemcontract "github.com/dujiao-next/internal/modules/redeem/contract"
	redeemdomain "github.com/dujiao-next/internal/modules/redeem/domain"
)

type pendingOrderStore struct {
	redeemcontract.OrderRepository
	order *redeemdomain.Order
}

func (s *pendingOrderStore) GetSucceededByCardSecretID(context.Context, uint) (*redeemdomain.Order, error) {
	return nil, redeemcontract.ErrNotFound
}

func (s *pendingOrderStore) GetLatestByCardSecretID(context.Context, uint) (*redeemdomain.Order, error) {
	if s.order == nil {
		return nil, redeemcontract.ErrNotFound
	}
	copy := *s.order
	return &copy, nil
}

func (s *pendingOrderStore) Create(_ context.Context, order *redeemdomain.Order) error {
	order.ID = 1
	copy := *order
	s.order = &copy
	return nil
}

func (s *pendingOrderStore) Update(_ context.Context, order *redeemdomain.Order) error {
	copy := *order
	s.order = &copy
	return nil
}

func (s *pendingOrderStore) Transaction(_ context.Context, fn func(redeemcontract.OrderRepository) error) error {
	return fn(s)
}

type pendingSecretLookup struct {
	redeemcontract.CardSecretLookup
}

func (pendingSecretLookup) FindByContent(context.Context, string) (*redeemcontract.CardSecretView, error) {
	return &redeemcontract.CardSecretView{ID: 1, ProductID: 1}, nil
}

type unknownSubmissionClient struct {
	redeemcontract.OrbitcardClient
	calls int
}

func (*unknownSubmissionClient) Configured() bool { return true }

func (c *unknownSubmissionClient) CreateSubscription(context.Context, redeemcontract.CreateSubscriptionInput) (*redeemcontract.CreateSubscriptionResult, error) {
	c.calls++
	return &redeemcontract.CreateSubscriptionResult{
		BusinessCode:      15000,
		SubmissionOutcome: "unknown",
		TaskNo:            "task-1",
	}, nil
}

func TestSubmitUnknownResultPreventsNewOrder(t *testing.T) {
	orders := &pendingOrderStore{}
	client := &unknownSubmissionClient{}
	svc := NewService(orders, nil, pendingSecretLookup{}, nil, client, Config{
		Enabled: true,
		ProductMaps: map[uint]ProductMapConfig{
			1: {ServiceCode: "chatgpt", PlanType: "gpt-plus"},
		},
	})
	in := SubmitInput{
		Code:       "test-code",
		Contact:    "test@example.test",
		Credential: `{"accessToken":"test-token"}`,
		Authorized: true,
	}
	result, err := svc.Submit(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != redeemdomain.StatusPendingConfirm || orders.order.Status != redeemdomain.StatusPendingConfirm {
		t.Fatalf("unknown submission must stay pending: %+v", result)
	}
	if orders.order.LastErrorCode != "15000" || orders.order.OrbitTaskNo != "task-1" || orders.order.CooldownUntil != nil {
		t.Fatalf("unknown submission must retain diagnostics and order identity without cooldown: %+v", orders.order)
	}
	_, err = svc.Submit(context.Background(), in)
	if !errors.Is(err, redeemcontract.ErrPendingConfirm) || client.calls != 1 {
		t.Fatalf("second submission must not reach upstream: err=%v calls=%d", err, client.calls)
	}
}
