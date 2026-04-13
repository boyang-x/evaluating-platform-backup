package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
)

// ToolCallInfo 工具调用信息（用于计费，与 agent 包解耦）
type ToolCallInfo struct {
	ToolName   string
	TokensUsed int
}

// Service 计费服务：负责余额检查、计费记录和扣费
type Service struct {
	userRepo    *repository.UserRepository
	billingRepo *repository.BillingRepository
}

// NewService 创建计费服务
func NewService(
	userRepo *repository.UserRepository,
	billingRepo *repository.BillingRepository,
) *Service {
	return &Service{
		userRepo:    userRepo,
		billingRepo: billingRepo,
	}
}

// CheckSufficientBalance 检查用户余额是否满足发起评估的最低要求
// 若余额不足返回 ErrInsufficientBalance
func (s *Service) CheckSufficientBalance(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user.Balance < MinAssessmentBalance {
		return fmt.Errorf(
			"insufficient balance: current %.4f 元, required %.2f 元",
			user.Balance, MinAssessmentBalance,
		)
	}
	return nil
}

// GetBalance 获取用户当前余额
func (s *Service) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get user balance: %w", err)
	}
	return user.Balance, nil
}

// ProcessAssessmentBilling 计算并扣除本次评估费用，同时持久化计费记录。
// 若评估使用了专家工作流资产（assetID / expertID 非 nil），
// 按 ExpertShareRatio 计算每条工具记录的分成，并在结算结束后一次性打款给专家。
func (s *Service) ProcessAssessmentBilling(
	ctx context.Context,
	userID uuid.UUID,
	assessmentID uuid.UUID,
	logs []ToolCallInfo,
	totalTokensUsed int,
	assetID *uuid.UUID,
	expertID *uuid.UUID,
) error {
	totalAmount := 0.0
	totalExpertShare := 0.0

	// 为每个工具调用创建计费记录
	for _, tc := range logs {
		amount := ToolPrice(tc.ToolName)
		totalAmount += amount

		rec := &model.BillingRecord{
			ID:           uuid.New(),
			UserID:       userID,
			AssessmentID: &assessmentID,
			ToolName:     tc.ToolName,
			ToolCategory: ToolCategory(tc.ToolName),
			CallCount:    1,
			TokensUsed:   tc.TokensUsed,
			Amount:       amount,
		}

		// 工作流资产调用：按比例计算专家分成
		if assetID != nil && expertID != nil {
			share := amount * ExpertShareRatio
			rec.AssetID = assetID
			rec.ExpertID = expertID
			rec.ExpertShare = share
			totalExpertShare += share
		}

		if err := s.billingRepo.CreateRecord(ctx, rec); err != nil {
			return fmt.Errorf("create billing record for %s: %w", tc.ToolName, err)
		}
	}

	// 单独记录 token 费用（工作流执行无 LLM 调用，此块通常不触发）
	if totalTokensUsed > 0 {
		tokenAmount := CalculateTokenCost(totalTokensUsed)
		totalAmount += tokenAmount

		tokenRec := &model.BillingRecord{
			ID:           uuid.New(),
			UserID:       userID,
			AssessmentID: &assessmentID,
			ToolName:     "token_usage",
			ToolCategory: "infrastructure",
			CallCount:    1,
			TokensUsed:   totalTokensUsed,
			Amount:       tokenAmount,
		}
		if err := s.billingRepo.CreateRecord(ctx, tokenRec); err != nil {
			return fmt.Errorf("create token billing record: %w", err)
		}
	}

	// 从用户余额扣款
	if totalAmount > 0 {
		description := fmt.Sprintf("评估任务 %s 费用扣除（%d 次工具调用，%d tokens）",
			assessmentID, len(logs), totalTokensUsed)
		if err := s.userRepo.UpdateBalance(ctx, userID, -totalAmount, description); err != nil {
			return fmt.Errorf("deduct balance: %w", err)
		}
	}

	// 一次性结算专家分成
	if totalExpertShare > 0 && expertID != nil {
		description := fmt.Sprintf("工作流资产收益分成 - 评估 %s（%d 次工具调用）",
			assessmentID, len(logs))
		if err := s.userRepo.UpdateBalance(ctx, *expertID, totalExpertShare, description); err != nil {
			return fmt.Errorf("credit expert share: %w", err)
		}
	}

	return nil
}

// GetBillingRecords 分页获取用户计费记录
func (s *Service) GetBillingRecords(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.BillingRecord, int, error) {
	return s.billingRepo.ListByUser(ctx, userID, limit, offset)
}

// GetTransactionHistory 分页获取用户余额变动记录
func (s *Service) GetTransactionHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.BalanceTransaction, int, error) {
	return s.billingRepo.ListTransactions(ctx, userID, limit, offset)
}

// GetAssessmentCost 查询某次评估的总费用
func (s *Service) GetAssessmentCost(ctx context.Context, assessmentID uuid.UUID) (float64, error) {
	return s.billingRepo.GetAssessmentCost(ctx, assessmentID)
}

// Recharge 给用户账户充值（正向余额变动）
func (s *Service) Recharge(ctx context.Context, userID uuid.UUID, amount float64, description string) error {
	if amount <= 0 {
		return fmt.Errorf("recharge amount must be positive")
	}
	return s.userRepo.UpdateBalance(ctx, userID, amount, description)
}

// GetExpertEarnings 分页查询专家的工具调用收益明细，同时返回累计总收益
func (s *Service) GetExpertEarnings(ctx context.Context, expertID uuid.UUID, limit, offset int) ([]model.BillingRecord, int, float64, error) {
	records, total, err := s.billingRepo.ListEarnings(ctx, expertID, limit, offset)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("list earnings: %w", err)
	}
	totalEarnings, err := s.billingRepo.GetTotalEarnings(ctx, expertID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("sum earnings: %w", err)
	}
	return records, total, totalEarnings, nil
}
