package biz

type Account struct {
	UserID       string
	Username     string
	DisplayName  string
	Group        string
	Quota        int64
	UsedQuota    int64
	RequestCount int64
	FrozenQuota  int64
	Status       int32
}

func (a *Account) AvailableQuota() int64 {
	return a.Quota - a.FrozenQuota
}

func (a *Account) GroupRatio() float64 {
	ratios := map[string]float64{
		"default": 1.0,
		"vip":     0.5,  // VIP 用户享受 5 折优惠
		"svip":    0.3,  // SVIP 用户享受 3 折优惠
	}

	if ratio, ok := ratios[a.Group]; ok {
		return ratio
	}
	return 1.0
}
