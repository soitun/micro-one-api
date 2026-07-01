package passthrough

import (
	"bytes"
	"net/http"
)

type Kind string

const (
	KindRetryable              Kind = "Retryable"
	KindRetryableOnSameAccount Kind = "RetryableOnSameAccount"
	KindNonRetryable           Kind = "NonRetryable"
	KindCyberBlocked           Kind = "CyberBlocked"
	KindPassthrough            Kind = "Passthrough"
)

type UpstreamError struct {
	StatusCode int
	Body       []byte
	Kind       Kind
}

func Classify(statusCode int, body []byte) UpstreamError {
	err := UpstreamError{StatusCode: statusCode, Body: body, Kind: KindNonRetryable}
	lowerBody := bytes.ToLower(body)
	if bytes.Contains(lowerBody, []byte("cyber_policy")) || bytes.Contains(lowerBody, []byte("cyber safety")) {
		err.Kind = KindCyberBlocked
		return err
	}
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden || statusCode == http.StatusTooManyRequests:
		err.Kind = KindPassthrough
	case statusCode >= 500:
		err.Kind = KindRetryable
	case statusCode == http.StatusConflict || statusCode == http.StatusLocked:
		err.Kind = KindRetryableOnSameAccount
	default:
		err.Kind = KindNonRetryable
	}
	return err
}

func (e UpstreamError) RetryableAcrossAccounts() bool {
	return e.Kind == KindRetryable
}

func (e UpstreamError) ShouldPassthrough() bool {
	return e.Kind == KindPassthrough || e.Kind == KindCyberBlocked
}
