package handler

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"payment-service/internal/repository"
	paymentv1 "simple-marketplace-project/gen/payment/v1"
)

type PaymentHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	repo   repository.PaymentRepository
	logger *slog.Logger
}

func NewPaymentHandler(repo repository.PaymentRepository, logger *slog.Logger) *PaymentHandler {
	return &PaymentHandler{repo: repo, logger: logger}
}

func (h *PaymentHandler) GetBalance(ctx context.Context, req *paymentv1.BalanceRequest) (*paymentv1.BalanceResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "userId is required")
	}

	balance, err := h.repo.GetBalance(req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get balance: %v", err))
	}

	return &paymentv1.BalanceResponse{
		UserBalance: balance,
	}, nil
}

func (h *PaymentHandler) CreateBalanceNote(ctx context.Context, req *paymentv1.BalanceNoteRequest) (*emptypb.Empty, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "userId is required")
	}
	if req.GetValue() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "value must be positive")
	}

	err := h.repo.CreatePayment(req.GetUserId(), req.GetIsDebit(), req.GetValue())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create balance note: %v", err))
	}

	return &emptypb.Empty{}, nil
}
