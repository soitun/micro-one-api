package biz

import (
	"context"
	"errors"
	"testing"
)

type testIdentityClient struct{}

func (testIdentityClient) GetAuthSnapshot(_ context.Context, _ string) (*AuthSnapshot, error) {
	return &AuthSnapshot{
		UserID:        1,
		TokenID:       1,
		Group:         "default",
		AllowedModels: []string{"gpt-4o-mini"},
		UserEnabled:   true,
		TokenEnabled:  true,
	}, nil
}

type testChannelClient struct{}

func (testChannelClient) SelectChannel(_ context.Context, group, model string, _ bool) (*Channel, error) {
	return &Channel{
		ID:      1,
		Name:    group + ":" + model,
		BaseURL: "https://api.openai.com/v1",
	}, nil
}

func TestRelayUsecasePlan(t *testing.T) {
	uc := NewRelayUsecase(testIdentityClient{}, testChannelClient{})
	plan, err := uc.Plan(context.Background(), RelayRequest{
		Token: "demo-token",
		Model: "gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Auth.Group != "default" {
		t.Fatalf("unexpected auth group: %s", plan.Auth.Group)
	}
	if plan.Channel.Name != "default:gpt-4o-mini" {
		t.Fatalf("unexpected channel name: %s", plan.Channel.Name)
	}
}

type testIdentityClientError struct {
	err error
}

func (c testIdentityClientError) GetAuthSnapshot(_ context.Context, _ string) (*AuthSnapshot, error) {
	return nil, c.err
}

func TestRelayUsecasePlan_IdentityError(t *testing.T) {
	wantErr := errors.New("token not found")
	uc := NewRelayUsecase(testIdentityClientError{err: wantErr}, testChannelClient{})
	_, err := uc.Plan(context.Background(), RelayRequest{Token: "bad-token", Model: "gpt-4o-mini"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

type testChannelClientError struct {
	err error
}

func (c testChannelClientError) SelectChannel(_ context.Context, _, _ string, _ bool) (*Channel, error) {
	return nil, c.err
}

func TestRelayUsecasePlan_ChannelError(t *testing.T) {
	wantErr := errors.New("no channel available")
	uc := NewRelayUsecase(testIdentityClient{}, testChannelClientError{err: wantErr})
	_, err := uc.Plan(context.Background(), RelayRequest{Token: "demo-token", Model: "gpt-4o-mini"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}
