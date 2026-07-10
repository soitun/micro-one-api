package biz

type Account struct {
	UserID       string
	Username     string
	DisplayName  string
	Group        string
	Balance      int64
	UsedAmount   int64
	RequestCount int64
	FrozenAmount int64
	Status       int32
}

func (a *Account) AvailableBalance() int64 {
	return a.Balance - a.FrozenAmount
}

// GroupRatio returns the ratio for this account's group using default ratios.
// Prefer using BillingUsecase.getGroupRatio() which supports externalized config.
func (a *Account) GroupRatio() float64 {
	ratios := DefaultGroupRatios()
	if ratio, ok := ratios[a.Group]; ok {
		return ratio
	}
	return 1.0
}
