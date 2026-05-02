package service

import (
	"context"
	"fmt"
	"time"

	adminv1 "micro-one-api/api/admin/v1"
	billingv1 "micro-one-api/api/billing/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdminService is the transport layer entry for admin-api.
type AdminService struct {
	billingClient billingv1.BillingServiceClient
}

// NewAdminService creates a new admin service
func NewAdminService(billingClient billingv1.BillingServiceClient) *AdminService {
	return &AdminService{
		billingClient: billingClient,
	}
}

// TopUpQuota 充值
func (s *AdminService) TopUpQuota(ctx context.Context, req *adminv1.TopUpQuotaRequest) (*adminv1.TopUpQuotaResponse, error) {
	billingReq := &billingv1.TopUpQuotaRequest{
		UserId:      req.UserId,
		Amount:      req.Amount,
		OperatorId:  req.OperatorId,
		Remark:      req.Remark,
	}

	resp, err := s.billingClient.TopUpQuota(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			return &adminv1.TopUpQuotaResponse{
				Success:      false,
				ErrorMessage: st.Message(),
			}, nil
		}
		return &adminv1.TopUpQuotaResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &adminv1.TopUpQuotaResponse{
		Success:    true,
		NewBalance: resp.NewQuota,
	}, nil
}

// CreateRedeemCode 创建兑换码
func (s *AdminService) CreateRedeemCode(ctx context.Context, req *adminv1.CreateRedeemCodeRequest) (*adminv1.CreateRedeemCodeResponse, error) {
	billingReq := &billingv1.CreateRedeemCodeRequest{
		Code:       req.Code,
		Name:       req.Name,
		Amount:     req.Amount,
		Count:      req.Count,
		OperatorId: req.OperatorId,
	}

	_, err := s.billingClient.CreateRedeemCode(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			return &adminv1.CreateRedeemCodeResponse{
				Success:      false,
				ErrorMessage: st.Message(),
			}, nil
		}
		return &adminv1.CreateRedeemCodeResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &adminv1.CreateRedeemCodeResponse{
		Success: true,
	}, nil
}

// CreateRedeemCodesBatch 批量创建兑换码
func (s *AdminService) CreateRedeemCodesBatch(ctx context.Context, req *adminv1.CreateRedeemCodesBatchRequest) (*adminv1.CreateRedeemCodesBatchResponse, error) {
	billingReq := &billingv1.CreateRedeemCodesBatchRequest{
		Name:       req.Name,
		Amount:     req.Amount,
		Count:      req.Count,
		BatchSize:  req.BatchSize,
		OperatorId: req.OperatorId,
	}

	resp, err := s.billingClient.CreateRedeemCodesBatch(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			return &adminv1.CreateRedeemCodesBatchResponse{
				Success:      false,
				ErrorMessage: st.Message(),
			}, nil
		}
		return &adminv1.CreateRedeemCodesBatchResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &adminv1.CreateRedeemCodesBatchResponse{
		Success: resp.Success,
		Codes:   resp.Codes,
	}, nil
}

// GetRedeemCode 获取兑换码
func (s *AdminService) GetRedeemCode(ctx context.Context, req *adminv1.GetRedeemCodeRequest) (*adminv1.GetRedeemCodeResponse, error) {
	billingReq := &billingv1.GetRedeemCodeRequest{
		Code: req.Code,
	}

	resp, err := s.billingClient.GetRedeemCode(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.NotFound {
				return &adminv1.GetRedeemCodeResponse{
					ErrorMessage: "redeem code not found",
				}, nil
			}
			return nil, err
		}
		return nil, err
	}

	redeemCodeInfo := &adminv1.RedeemCodeInfo{
		Code:      resp.RedeemCode.Code,
		Name:      resp.RedeemCode.Name,
		Amount:    resp.RedeemCode.Amount,
		Count:     resp.RedeemCode.Count,
		Status:    resp.RedeemCode.Status,
		CreatedBy: resp.RedeemCode.CreatedBy,
		CreatedAt: resp.RedeemCode.CreatedAt.AsTime().Unix(),
	}

	return &adminv1.GetRedeemCodeResponse{
		RedeemCode: redeemCodeInfo,
	}, nil
}

// ListRedeemCodes 获取兑换码列表
func (s *AdminService) ListRedeemCodes(ctx context.Context, req *adminv1.ListRedeemCodesRequest) (*adminv1.ListRedeemCodesResponse, error) {
	billingReq := &billingv1.ListRedeemCodesRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	resp, err := s.billingClient.ListRedeemCodes(ctx, billingReq)
	if err != nil {
		return &adminv1.ListRedeemCodesResponse{
			Codes: []*adminv1.RedeemCodeInfo{},
			Total: 0,
		}, nil
	}

	codes := make([]*adminv1.RedeemCodeInfo, len(resp.Codes))
	for i, code := range resp.Codes {
		codes[i] = &adminv1.RedeemCodeInfo{
			Code:      code.Code,
			Name:      code.Name,
			Amount:    code.Amount,
			Count:     code.Count,
			Status:    code.Status,
			CreatedBy: code.CreatedBy,
			CreatedAt: code.CreatedAt.AsTime().Unix(),
		}
	}

	return &adminv1.ListRedeemCodesResponse{
		Codes: codes,
		Total: resp.Total,
	}, nil
}

// SearchRedeemCodes 搜索兑换码
func (s *AdminService) SearchRedeemCodes(ctx context.Context, req *adminv1.SearchRedeemCodesRequest) (*adminv1.SearchRedeemCodesResponse, error) {
	billingReq := &billingv1.SearchRedeemCodesRequest{
		Keyword: req.Keyword,
	}

	resp, err := s.billingClient.SearchRedeemCodes(ctx, billingReq)
	if err != nil {
		return &adminv1.SearchRedeemCodesResponse{
			Codes: []*adminv1.RedeemCodeInfo{},
		}, nil
	}

	codes := make([]*adminv1.RedeemCodeInfo, len(resp.Codes))
	for i, code := range resp.Codes {
		codes[i] = &adminv1.RedeemCodeInfo{
			Code:      code.Code,
			Name:      code.Name,
			Amount:    code.Amount,
			Count:     code.Count,
			Status:    code.Status,
			CreatedBy: code.CreatedBy,
			CreatedAt: code.CreatedAt.AsTime().Unix(),
		}
	}

	return &adminv1.SearchRedeemCodesResponse{
		Codes: codes,
	}, nil
}

// UpdateRedeemCode 更新兑换码
func (s *AdminService) UpdateRedeemCode(ctx context.Context, req *adminv1.UpdateRedeemCodeRequest) (*adminv1.UpdateRedeemCodeResponse, error) {
	billingReq := &billingv1.UpdateRedeemCodeRequest{
		Code:   req.Code,
		Name:   req.Name,
		Amount: req.Amount,
		Status: req.Status,
	}

	_, err := s.billingClient.UpdateRedeemCode(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			return &adminv1.UpdateRedeemCodeResponse{
				Success:      false,
				ErrorMessage: st.Message(),
			}, nil
		}
		return &adminv1.UpdateRedeemCodeResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &adminv1.UpdateRedeemCodeResponse{
		Success: true,
	}, nil
}

// DeleteRedeemCode 删除兑换码
func (s *AdminService) DeleteRedeemCode(ctx context.Context, req *adminv1.DeleteRedeemCodeRequest) (*adminv1.DeleteRedeemCodeResponse, error) {
	billingReq := &billingv1.DeleteRedeemCodeRequest{
		Code: req.Code,
	}

	_, err := s.billingClient.DeleteRedeemCode(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			return &adminv1.DeleteRedeemCodeResponse{
				Success:      false,
				ErrorMessage: st.Message(),
			}, nil
		}
		return &adminv1.DeleteRedeemCodeResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &adminv1.DeleteRedeemCodeResponse{
		Success: true,
	}, nil
}

// ListUserLedger 查询用户流水
func (s *AdminService) ListUserLedger(ctx context.Context, req *adminv1.ListUserLedgerRequest) (*adminv1.ListUserLedgerResponse, error) {
	billingReq := &billingv1.ListLedgerRequest{
		UserId:   req.UserId,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	resp, err := s.billingClient.ListLedger(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.NotFound {
				return &adminv1.ListUserLedgerResponse{
					Ledgers: []*adminv1.LedgerInfo{},
					Total:   0,
				}, nil
			}
			return nil, err
		}
		return nil, err
	}

	ledgers := make([]*adminv1.LedgerInfo, len(resp.Entries))
	for i, ledger := range resp.Entries {
		ledgers[i] = &adminv1.LedgerInfo{
			Id:           0, // LedgerEntry 的 id 是 string 类型，需要转换
			UserId:       ledger.UserId,
			Amount:       ledger.Amount,
			BalanceAfter: ledger.BalanceAfter,
			Type:         ledger.Type,
			ReferenceId:  ledger.ReferenceId,
			Remark:       ledger.Remark,
			CreatedAt:    ledger.CreatedAt.AsTime().Unix(),
		}
	}

	return &adminv1.ListUserLedgerResponse{
		Ledgers: ledgers,
		Total:   resp.Total,
	}, nil
}

// GetAccountSnapshot 获取账户快照
func (s *AdminService) GetAccountSnapshot(ctx context.Context, req *adminv1.GetAccountSnapshotRequest) (*adminv1.GetAccountSnapshotResponse, error) {
	billingReq := &billingv1.GetAccountSnapshotRequest{
		UserId: req.UserId,
	}

	resp, err := s.billingClient.GetAccountSnapshot(ctx, billingReq)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.NotFound {
				return nil, status.Error(codes.NotFound, "account not found")
			}
			return nil, err
		}
		return nil, err
	}

	return &adminv1.GetAccountSnapshotResponse{
		Account: &adminv1.AccountInfo{
			UserId:       resp.Snapshot.UserId,
			Username:     "", // AccountSnapshot 没有 username 字段
			DisplayName:  "", // AccountSnapshot 没有 display_name 字段
			Group:        resp.Snapshot.Group,
			Quota:        resp.Snapshot.Quota,
			UsedQuota:    resp.Snapshot.UsedQuota,
			RequestCount: resp.Snapshot.RequestCount,
			FrozenQuota:  resp.Snapshot.FrozenQuota,
			Status:       0, // AccountSnapshot 没有 status 字段
		},
	}, nil
}

// 辅助函数：将 time.Time 转换为 Unix 时间戳
func toUnixTimestamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// 辅助函数：将 Unix 时间戳转换为 time.Time
func fromUnixTimestamp(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// 辅助函数：创建错误响应
func errorResponse(message string) error {
	return status.Error(codes.Internal, fmt.Sprintf("internal error: %s", message))
}