package service

import (
	"context"
	"fmt"
	"time"

	billingv1 "micro-one-api/api/billing/v1"
	"micro-one-api/internal/billing/biz"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type BillingService struct {
	billingv1.UnimplementedBillingServiceServer
	uc *biz.BillingUsecase
}

func NewBillingService(uc *biz.BillingUsecase) *BillingService {
	return &BillingService{uc: uc}
}

func (s *BillingService) ReserveQuota(ctx context.Context, req *billingv1.ReserveQuotaRequest) (*billingv1.ReserveQuotaResponse, error) {
	reservation, err := s.uc.ReserveQuota(ctx, req.UserId, req.RequestId, req.EstimatedTokens, req.Model, req.ChannelId)
	if err != nil {
		return &billingv1.ReserveQuotaResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.ReserveQuotaResponse{
		Success:        true,
		ReservationId:   reservation.ReservationID,
		ReservedAmount: reservation.Amount,
	}, nil
}

func (s *BillingService) CommitQuota(ctx context.Context, req *billingv1.CommitQuotaRequest) (*billingv1.CommitQuotaResponse, error) {
	committedAmount, refundAmount, err := s.uc.CommitQuota(ctx, req.ReservationId, req.ActualTokens, req.Success)
	if err != nil {
		return &billingv1.CommitQuotaResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.CommitQuotaResponse{
		Success:        true,
		CommittedAmount: committedAmount,
		RefundAmount:   refundAmount,
	}, nil
}

func (s *BillingService) ReleaseQuota(ctx context.Context, req *billingv1.ReleaseQuotaRequest) (*billingv1.ReleaseQuotaResponse, error) {
	err := s.uc.ReleaseQuota(ctx, req.ReservationId, req.Reason)
	if err != nil {
		return &billingv1.ReleaseQuotaResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.ReleaseQuotaResponse{
		Success: true,
	}, nil
}

func (s *BillingService) GetAccountSnapshot(ctx context.Context, req *billingv1.GetAccountSnapshotRequest) (*billingv1.GetAccountSnapshotResponse, error) {
	account, err := s.uc.GetAccountSnapshot(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	return &billingv1.GetAccountSnapshotResponse{
		Snapshot: &billingv1.AccountSnapshot{
			UserId:       account.UserID,
			Quota:        account.Quota,
			UsedQuota:    account.UsedQuota,
			RequestCount: account.RequestCount,
			Group:        account.Group,
			GroupRatio:   account.GroupRatio(),
			FrozenQuota:  account.FrozenQuota,
		},
	}, nil
}

func (s *BillingService) TopUpQuota(ctx context.Context, req *billingv1.TopUpQuotaRequest) (*billingv1.TopUpQuotaResponse, error) {
	newQuota, err := s.uc.TopUpQuota(ctx, req.UserId, req.OperatorId, req.Amount, req.Remark)
	if err != nil {
		return &billingv1.TopUpQuotaResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.TopUpQuotaResponse{
		Success:  true,
		NewQuota: newQuota,
	}, nil
}

func (s *BillingService) CreateRedeemCode(ctx context.Context, req *billingv1.CreateRedeemCodeRequest) (*billingv1.CreateRedeemCodeResponse, error) {
	err := s.uc.CreateRedeemCode(ctx, req.Code, req.Name, req.Amount, req.Count, req.OperatorId)
	if err != nil {
		return &billingv1.CreateRedeemCodeResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.CreateRedeemCodeResponse{
		Success: true,
	}, nil
}

func (s *BillingService) CreateRedeemCodesBatch(ctx context.Context, req *billingv1.CreateRedeemCodesBatchRequest) (*billingv1.CreateRedeemCodesBatchResponse, error) {
	codes, err := s.uc.CreateRedeemCodesBatch(ctx, req.Name, req.Amount, req.Count, req.BatchSize, req.OperatorId)
	if err != nil {
		return &billingv1.CreateRedeemCodesBatchResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.CreateRedeemCodesBatchResponse{
		Success: true,
		Codes:   codes,
	}, nil
}

func (s *BillingService) GetRedeemCode(ctx context.Context, req *billingv1.GetRedeemCodeRequest) (*billingv1.GetRedeemCodeResponse, error) {
	code, err := s.uc.GetRedeemCode(ctx, req.Code)
	if err != nil {
		return &billingv1.GetRedeemCodeResponse{
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.GetRedeemCodeResponse{
		RedeemCode: &billingv1.RedeemCode{
			Code:      code.Code,
			Name:      code.Name,
			Amount:    code.Amount,
			Count:     code.Count,
			Status:    code.Status,
			CreatedBy: code.CreatedBy,
			CreatedAt: toProtoTimestamp(code.CreatedAt),
			UpdatedAt: toProtoTimestamp(code.UpdatedAt),
		},
	}, nil
}

func (s *BillingService) ListRedeemCodes(ctx context.Context, req *billingv1.ListRedeemCodesRequest) (*billingv1.ListRedeemCodesResponse, error) {
	codes, total, err := s.uc.ListRedeemCodes(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	redeemCodes := make([]*billingv1.RedeemCode, len(codes))
	for i, code := range codes {
		redeemCodes[i] = &billingv1.RedeemCode{
			Code:      code.Code,
			Name:      code.Name,
			Amount:    code.Amount,
			Count:     code.Count,
			Status:    code.Status,
			CreatedBy: code.CreatedBy,
			CreatedAt: toProtoTimestamp(code.CreatedAt),
			UpdatedAt: toProtoTimestamp(code.UpdatedAt),
		}
	}

	return &billingv1.ListRedeemCodesResponse{
		Codes: redeemCodes,
		Total: total,
	}, nil
}

func (s *BillingService) SearchRedeemCodes(ctx context.Context, req *billingv1.SearchRedeemCodesRequest) (*billingv1.SearchRedeemCodesResponse, error) {
	codes, err := s.uc.SearchRedeemCodes(ctx, req.Keyword)
	if err != nil {
		return nil, err
	}

	redeemCodes := make([]*billingv1.RedeemCode, len(codes))
	for i, code := range codes {
		redeemCodes[i] = &billingv1.RedeemCode{
			Code:      code.Code,
			Name:      code.Name,
			Amount:    code.Amount,
			Count:     code.Count,
			Status:    code.Status,
			CreatedBy: code.CreatedBy,
			CreatedAt: toProtoTimestamp(code.CreatedAt),
			UpdatedAt: toProtoTimestamp(code.UpdatedAt),
		}
	}

	return &billingv1.SearchRedeemCodesResponse{
		Codes: redeemCodes,
	}, nil
}

func (s *BillingService) UpdateRedeemCode(ctx context.Context, req *billingv1.UpdateRedeemCodeRequest) (*billingv1.UpdateRedeemCodeResponse, error) {
	err := s.uc.UpdateRedeemCode(ctx, req.Code, req.Name, req.Amount, req.Status)
	if err != nil {
		return &billingv1.UpdateRedeemCodeResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.UpdateRedeemCodeResponse{
		Success: true,
	}, nil
}

func (s *BillingService) DeleteRedeemCode(ctx context.Context, req *billingv1.DeleteRedeemCodeRequest) (*billingv1.DeleteRedeemCodeResponse, error) {
	err := s.uc.DeleteRedeemCode(ctx, req.Code)
	if err != nil {
		return &billingv1.DeleteRedeemCodeResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.DeleteRedeemCodeResponse{
		Success: true,
	}, nil
}

func (s *BillingService) RedeemCode(ctx context.Context, req *billingv1.RedeemCodeRequest) (*billingv1.RedeemCodeResponse, error) {
	amount, newQuota, err := s.uc.RedeemCode(ctx, req.UserId, req.Code)
	if err != nil {
		return &billingv1.RedeemCodeResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &billingv1.RedeemCodeResponse{
		Success:  true,
		Amount:   amount,
		NewQuota: newQuota,
	}, nil
}

func (s *BillingService) ListLedger(ctx context.Context, req *billingv1.ListLedgerRequest) (*billingv1.ListLedgerResponse, error) {
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	ledgers, total, err := s.uc.ListLedgers(ctx, req.UserId, page, pageSize)
	if err != nil {
		return nil, err
	}

	entries := make([]*billingv1.LedgerEntry, len(ledgers))
	for i, ledger := range ledgers {
		entries[i] = &billingv1.LedgerEntry{
			Id:           fmt.Sprintf("%d", ledger.ID),
			UserId:       ledger.UserID,
			Amount:       ledger.Amount,
			BalanceAfter: ledger.BalanceAfter,
			Type:         ledger.Type,
			ReferenceId:  ledger.ReferenceID,
			Remark:       ledger.Remark,
			CreatedAt:    toProtoTimestamp(ledger.CreatedAt),
		}
	}

	return &billingv1.ListLedgerResponse{
		Entries: entries,
		Total:   total,
	}, nil
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return &timestamppb.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}
