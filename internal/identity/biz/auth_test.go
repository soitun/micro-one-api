package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockIdentityRepo struct {
	tokens map[string]*Token
	users  map[int64]*User
}

func (m *mockIdentityRepo) FindTokenByKey(ctx context.Context, key string) (*Token, error) {
	token, ok := m.tokens[key]
	if !ok {
		return nil, ErrTokenNotFound
	}
	return token, nil
}

func (m *mockIdentityRepo) FindUserByID(ctx context.Context, userID int64) (*User, error) {
	user, ok := m.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func TestIdentityUsecase_ValidateToken_ValidToken(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"valid-token": {
				ID:             1,
				UserID:         1,
				Key:            "valid-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
				Models:         []string{"gpt-4o-mini"},
			},
		},
		users: map[int64]*User{
			1: {
				ID:       1,
				Username: "test-user",
				Status:   UserStatusEnabled,
			},
		},
	}

	uc := NewIdentityUsecase(repo)
	token, err := uc.ValidateToken(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if token.ID != 1 {
		t.Fatalf("unexpected token ID: %d", token.ID)
	}
	if token.UserID != 1 {
		t.Fatalf("unexpected token user ID: %d", token.UserID)
	}
}

func TestIdentityUsecase_ValidateToken_EmptyToken(t *testing.T) {
	repo := &mockIdentityRepo{tokens: make(map[string]*Token), users: make(map[int64]*User)}
	uc := NewIdentityUsecase(repo)

	_, err := uc.ValidateToken(context.Background(), "")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
	}

	_, err = uc.ValidateToken(context.Background(), "   ")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for whitespace token, got: %v", err)
	}
}

func TestIdentityUsecase_ValidateToken_TokenNotFound(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: make(map[string]*Token),
		users:  make(map[int64]*User),
	}
	uc := NewIdentityUsecase(repo)

	_, err := uc.ValidateToken(context.Background(), "nonexistent-token")
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got: %v", err)
	}
}

func TestIdentityUsecase_ValidateToken_TokenExpired(t *testing.T) {
	expiredTime := time.Now().Add(-time.Hour).Unix()
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"expired-token": {
				ID:             1,
				UserID:         1,
				Key:            "expired-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      expiredTime,
				RemainQuota:    1000,
				UnlimitedQuota: false,
			},
		},
		users: map[int64]*User{
			1: {ID: 1, Username: "test-user", Status: UserStatusEnabled},
		},
	}

	uc := NewIdentityUsecase(repo)
	_, err := uc.ValidateToken(context.Background(), "expired-token")
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got: %v", err)
	}
}

func TestIdentityUsecase_ValidateToken_TokenStatusExpired(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"expired-status-token": {
				ID:             1,
				UserID:         1,
				Key:            "expired-status-token",
				Status:         TokenStatusExpired,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
			},
		},
		users: map[int64]*User{
			1: {ID: 1, Username: "test-user", Status: UserStatusEnabled},
		},
	}

	uc := NewIdentityUsecase(repo)
	_, err := uc.ValidateToken(context.Background(), "expired-status-token")
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got: %v", err)
	}
}

func TestIdentityUsecase_ValidateToken_TokenExhausted(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"exhausted-token": {
				ID:             1,
				UserID:         1,
				Key:            "exhausted-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    0,
				UnlimitedQuota: false,
			},
		},
		users: map[int64]*User{
			1: {ID: 1, Username: "test-user", Status: UserStatusEnabled},
		},
	}

	uc := NewIdentityUsecase(repo)
	_, err := uc.ValidateToken(context.Background(), "exhausted-token")
	if !errors.Is(err, ErrTokenExhausted) {
		t.Fatalf("expected ErrTokenExhausted, got: %v", err)
	}
}

func TestIdentityUsecase_ValidateToken_TokenStatusExhausted(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"exhausted-status-token": {
				ID:             1,
				UserID:         1,
				Key:            "exhausted-status-token",
				Status:         TokenStatusExhausted,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
			},
		},
		users: map[int64]*User{
			1: {ID: 1, Username: "test-user", Status: UserStatusEnabled},
		},
	}

	uc := NewIdentityUsecase(repo)
	_, err := uc.ValidateToken(context.Background(), "exhausted-status-token")
	if !errors.Is(err, ErrTokenExhausted) {
		t.Fatalf("expected ErrTokenExhausted, got: %v", err)
	}
}

func TestIdentityUsecase_ValidateToken_TokenDisabled(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"disabled-token": {
				ID:             1,
				UserID:         1,
				Key:            "disabled-token",
				Status:         TokenStatusDisabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
			},
		},
		users: map[int64]*User{
			1: {ID: 1, Username: "test-user", Status: UserStatusEnabled},
		},
	}

	uc := NewIdentityUsecase(repo)
	_, err := uc.ValidateToken(context.Background(), "disabled-token")
	if !errors.Is(err, ErrTokenDisabled) {
		t.Fatalf("expected ErrTokenDisabled, got: %v", err)
	}
}

func TestIdentityUsecase_ValidateToken_UnlimitedQuota(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"unlimited-token": {
				ID:             1,
				UserID:         1,
				Key:            "unlimited-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    0,
				UnlimitedQuota: true,
			},
		},
		users: map[int64]*User{
			1: {ID: 1, Username: "test-user", Status: UserStatusEnabled},
		},
	}

	uc := NewIdentityUsecase(repo)
	token, err := uc.ValidateToken(context.Background(), "unlimited-token")
	if err != nil {
		t.Fatalf("ValidateToken() unexpected error: %v", err)
	}
	if !token.UnlimitedQuota {
		t.Fatal("expected unlimited quota")
	}
}

func TestIdentityUsecase_GetAuthSnapshot_Valid(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"valid-token": {
				ID:             1,
				UserID:         1,
				Key:            "valid-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
				Models:         []string{"gpt-4o-mini", "gpt-4o"},
			},
		},
		users: map[int64]*User{
			1: {
				ID:          1,
				Username:    "test-user",
				DisplayName: "Test User",
				Group:       "default",
				Status:      UserStatusEnabled,
			},
		},
	}

	uc := NewIdentityUsecase(repo)
	snapshot, err := uc.GetAuthSnapshot(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("GetAuthSnapshot() error = %v", err)
	}
	if snapshot.UserID != 1 {
		t.Fatalf("unexpected user ID: %d", snapshot.UserID)
	}
	if snapshot.TokenID != 1 {
		t.Fatalf("unexpected token ID: %d", snapshot.TokenID)
	}
	if snapshot.Group != "default" {
		t.Fatalf("unexpected group: %s", snapshot.Group)
	}
	if len(snapshot.AllowedModels) != 2 {
		t.Fatalf("unexpected models count: %d", len(snapshot.AllowedModels))
	}
	if !snapshot.UserEnabled {
		t.Fatal("expected user enabled")
	}
	if !snapshot.TokenEnabled {
		t.Fatal("expected token enabled")
	}
}

func TestIdentityUsecase_GetAuthSnapshot_UserDisabled(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"valid-token": {
				ID:             1,
				UserID:         1,
				Key:            "valid-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
			},
		},
		users: map[int64]*User{
			1: {
				ID:       1,
				Username: "test-user",
				Group:    "default",
				Status:   UserStatusDisabled,
			},
		},
	}

	uc := NewIdentityUsecase(repo)
	_, err := uc.GetAuthSnapshot(context.Background(), "valid-token")
	if !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("expected ErrUserDisabled, got: %v", err)
	}
}

func TestIdentityUsecase_GetAuthSnapshot_ModelWhitelist(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"restricted-token": {
				ID:             1,
				UserID:         1,
				Key:            "restricted-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
				Models:         []string{"gpt-4o-mini"},
			},
		},
		users: map[int64]*User{
			1: {
				ID:       1,
				Username: "test-user",
				Group:    "default",
				Status:   UserStatusEnabled,
			},
		},
	}

	uc := NewIdentityUsecase(repo)
	snapshot, err := uc.GetAuthSnapshot(context.Background(), "restricted-token")
	if err != nil {
		t.Fatalf("GetAuthSnapshot() error = %v", err)
	}
	if len(snapshot.AllowedModels) != 1 {
		t.Fatalf("expected 1 model, got: %d", len(snapshot.AllowedModels))
	}
	if snapshot.AllowedModels[0] != "gpt-4o-mini" {
		t.Fatalf("unexpected model: %s", snapshot.AllowedModels[0])
	}
}

func TestIdentityUsecase_GetAuthSnapshot_EmptyModelWhitelist(t *testing.T) {
	repo := &mockIdentityRepo{
		tokens: map[string]*Token{
			"all-models-token": {
				ID:             1,
				UserID:         1,
				Key:            "all-models-token",
				Status:         TokenStatusEnabled,
				ExpiredAt:      time.Now().Add(time.Hour).Unix(),
				RemainQuota:    1000,
				UnlimitedQuota: false,
				Models:         []string{},
			},
		},
		users: map[int64]*User{
			1: {
				ID:       1,
				Username: "test-user",
				Group:    "default",
				Status:   UserStatusEnabled,
			},
		},
	}

	uc := NewIdentityUsecase(repo)
	snapshot, err := uc.GetAuthSnapshot(context.Background(), "all-models-token")
	if err != nil {
		t.Fatalf("GetAuthSnapshot() error = %v", err)
	}
	if len(snapshot.AllowedModels) != 0 {
		t.Fatalf("expected empty models (no restrictions), got: %v", snapshot.AllowedModels)
	}
}

func TestSplitCSVPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected []string
	}{
		{"nil input", nil, nil},
		{"empty string", strPtr(""), []string{}},
		{"single item", strPtr("gpt-4o-mini"), []string{"gpt-4o-mini"}},
		{"multiple items", strPtr("gpt-4o-mini,gpt-4o"), []string{"gpt-4o-mini", "gpt-4o"}},
		{"items with spaces", strPtr("gpt-4o-mini, gpt-4o, gpt-3.5-turbo"), []string{"gpt-4o-mini", "gpt-4o", "gpt-3.5-turbo"}},
		{"items with extra spaces", strPtr("  gpt-4o-mini  ,  gpt-4o  "), []string{"gpt-4o-mini", "gpt-4o"}},
		{"empty items", strPtr("gpt-4o-mini,,gpt-4o"), []string{"gpt-4o-mini", "gpt-4o"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitCSVPtr(tt.input)
			if !equalStringSlices(result, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty string", "", []string{}},
		{"single item", "gpt-4o-mini", []string{"gpt-4o-mini"}},
		{"multiple items", "gpt-4o-mini,gpt-4o", []string{"gpt-4o-mini", "gpt-4o"}},
		{"items with spaces", "gpt-4o-mini, gpt-4o, gpt-3.5-turbo", []string{"gpt-4o-mini", "gpt-4o", "gpt-3.5-turbo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCSV(tt.input)
			if !equalStringSlices(result, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
