package data

import (
	"context"
	"time"

	channelv1 "micro-one-api/api/channel/v1"
	relaycredential "micro-one-api/internal/relay/credential"
)

// ChannelSubscriptionAccountStore adapts channel-service subscription-account
// RPCs to the relay credential and server resolver interfaces.
type ChannelSubscriptionAccountStore struct {
	client channelv1.ChannelServiceClient
}

func NewChannelSubscriptionAccountStore(client channelv1.ChannelServiceClient) *ChannelSubscriptionAccountStore {
	return &ChannelSubscriptionAccountStore{client: client}
}

func (s *ChannelSubscriptionAccountStore) Lookup(ctx context.Context, accountID int64) (*relaycredential.AccountCredentials, error) {
	if s == nil || s.client == nil {
		return nil, relaycredential.ErrNotConfigured
	}
	reply, err := s.client.GetSubscriptionAccount(ctx, &channelv1.GetSubscriptionAccountRequest{AccountId: accountID})
	if err != nil {
		return nil, err
	}
	account := reply.GetAccount()
	if account == nil {
		return nil, relaycredential.ErrAccountNotFound
	}
	return &relaycredential.AccountCredentials{
		AccountID:    account.GetAccountId(),
		AccessToken:  account.GetAccessToken(),
		RefreshToken: account.GetRefreshToken(),
		ExpiresAt:    time.Unix(account.GetExpiresAt(), 0),
	}, nil
}

func (s *ChannelSubscriptionAccountStore) Store(ctx context.Context, accountID int64, creds *relaycredential.AccountCredentials) error {
	if s == nil || s.client == nil {
		return relaycredential.ErrNotConfigured
	}
	req := &channelv1.UpdateSubscriptionAccountRequest{Id: accountID}
	if creds != nil {
		req.AccessToken = creds.AccessToken
		req.RefreshToken = creds.RefreshToken
		req.ExpiresAt = creds.ExpiresAt.Unix()
		req.AccountId = creds.AccountID
	}
	reply, err := s.client.UpdateSubscriptionAccount(ctx, req)
	if err != nil {
		return err
	}
	if reply != nil && !reply.GetSuccess() {
		return relaycredential.ErrRefreshFailed
	}
	return nil
}

func (s *ChannelSubscriptionAccountStore) Resolve(ctx context.Context, channelID int64) (*relaycredential.SubscriptionAccountMetadata, error) {
	if s == nil || s.client == nil {
		return nil, relaycredential.ErrNotConfigured
	}
	reply, err := s.client.GetSubscriptionAccount(ctx, &channelv1.GetSubscriptionAccountRequest{AccountId: channelID})
	if err != nil {
		return nil, err
	}
	account := reply.GetAccount()
	if account == nil {
		return nil, relaycredential.ErrAccountNotFound
	}
	return &relaycredential.SubscriptionAccountMetadata{
		ID:          account.GetId(),
		AccessToken: account.GetAccessToken(),
		AccountID:   account.GetAccountId(),
		Platform:    relaycredential.Platform(account.GetPlatform()),
		Fingerprint: account.GetFingerprint(),
		GroupID:     account.GetGroup(),
	}, nil
}

func (s *ChannelSubscriptionAccountStore) PlatformOf(ctx context.Context, accountID int64) relaycredential.Platform {
	if s == nil || s.client == nil {
		return ""
	}
	reply, err := s.client.GetSubscriptionAccount(ctx, &channelv1.GetSubscriptionAccountRequest{AccountId: accountID})
	if err != nil || reply.GetAccount() == nil {
		return ""
	}
	return relaycredential.Platform(reply.GetAccount().GetPlatform())
}

var (
	_ relaycredential.AccountLookup               = (*ChannelSubscriptionAccountStore)(nil)
	_ relaycredential.SubscriptionAccountResolver = (*ChannelSubscriptionAccountStore)(nil)
)
