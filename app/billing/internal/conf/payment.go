package conf

import "micro-one-api/app/billing/internal/biz"

// ToPaymentConfig converts the proto Payment to the biz PaymentConfig.
func (p *Payment) ToPaymentConfig() biz.PaymentConfig {
	cfg := biz.PaymentConfig{
		AmountPerUnit: p.AmountPerUnit,
	}

	if p.Alipay != nil {
		cfg.Alipay = biz.AlipayConfig{
			Enabled:              p.Alipay.Enabled,
			FormURL:              p.Alipay.FormUrl,
			AppID:                p.Alipay.AppId,
			PrivateKey:           p.Alipay.PrivateKey,
			PrivateKeyPath:       p.Alipay.PrivateKeyPath,
			PublicKey:            p.Alipay.PublicKey,
			PublicKeyPath:        p.Alipay.PublicKeyPath,
			NotifyURL:            p.Alipay.NotifyUrl,
			ReturnURL:            p.Alipay.ReturnUrl,
			AppCertPath:          p.Alipay.AppCertPath,
			RootCertPath:         p.Alipay.RootCertPath,
			AlipayPublicCertPath: p.Alipay.AlipayPublicCertPath,
		}
	}

	return cfg
}
