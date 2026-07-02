package biz

import "context"

type SubscriptionRepository interface {
	CreateSubscription(ctx context.Context, subscription *UserSubscription) error
	UpdateSubscription(ctx context.Context, subscription *UserSubscription) error
	DeleteSubscription(ctx context.Context, subscriptionID int64) error
	GetSubscriptionByID(ctx context.Context, subscriptionID int64) (*UserSubscription, error)
	ListSubscriptionsByUser(ctx context.Context, userID int64) ([]*UserSubscription, error)
	ListActiveSubscriptions(ctx context.Context) ([]*UserSubscription, error)
	// ListAllSubscriptions returns every subscription regardless of user or
	// status, newest first, so admins can browse without knowing a user id.
	ListAllSubscriptions(ctx context.Context) ([]*UserSubscription, error)
	GetActiveSubscriptionByUser(ctx context.Context, userID int64) (*UserSubscription, error)
	// AddUsage atomically rolls the active subscription's usage windows relative
	// to now (unix seconds) and adds costUSD to every window. Implementations
	// must perform the read-roll-increment as a single atomic unit so concurrent
	// callers cannot lose each other's increments.
	AddUsage(ctx context.Context, userID int64, costUSD float64, now int64) error
}

type GroupRepository interface {
	CreateGroup(ctx context.Context, group *SubscriptionGroup) error
	UpdateGroup(ctx context.Context, group *SubscriptionGroup) error
	DeleteGroup(ctx context.Context, groupID int64) error
	GetGroupByID(ctx context.Context, groupID int64) (*SubscriptionGroup, error)
	GetGroupByName(ctx context.Context, name string) (*SubscriptionGroup, error)
	ListGroups(ctx context.Context) ([]*SubscriptionGroup, error)
}
