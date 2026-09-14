package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"payment-service/internal/repository"
	paymentv1 "simple-marketplace-project/gen/payment/v1"
)

// mockPaymentRepository — мок интерфейса repository.PaymentRepository.
type mockPaymentRepository struct {
	mock.Mock
}

func (m *mockPaymentRepository) GetBalance(ctx context.Context, userId string) (float64, error) {
	args := m.Called(ctx, userId)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockPaymentRepository) CreatePayment(ctx context.Context, userId string, isDebit bool, value float64) error {
	args := m.Called(ctx, userId, isDebit, value)
	return args.Error(0)
}

func newTestHandler(repo *mockPaymentRepository) *PaymentHandler {
	logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 100})) // no-op logger
	return NewPaymentHandler(repo, logger)
}

func TestPaymentHandler_GetBalance(t *testing.T) {
	t.Run("empty userId returns InvalidArgument", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		resp, err := h.GetBalance(context.Background(), &paymentv1.BalanceRequest{UserId: ""})

		require.Nil(t, resp)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		repo.AssertNotCalled(t, "GetBalance", mock.Anything, mock.Anything)
	})

	t.Run("returns balance on success", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		repo.On("GetBalance", mock.Anything, "user-1").Return(123.45, nil)

		resp, err := h.GetBalance(context.Background(), &paymentv1.BalanceRequest{UserId: "user-1"})

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, 123.45, resp.GetUserBalance())
		repo.AssertExpectations(t)
	})

	t.Run("repo error maps to Internal", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		repo.On("GetBalance", mock.Anything, "user-1").Return(0.0, errors.New("db is down"))

		resp, err := h.GetBalance(context.Background(), &paymentv1.BalanceRequest{UserId: "user-1"})

		require.Nil(t, resp)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "db is down")
		repo.AssertExpectations(t)
	})
}

func TestPaymentHandler_CreateBalanceNote(t *testing.T) {
	t.Run("empty userId returns InvalidArgument", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		resp, err := h.CreateBalanceNote(context.Background(), &paymentv1.BalanceNoteRequest{
			UserId: "",
			Value:  10,
		})

		require.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		repo.AssertNotCalled(t, "CreatePayment", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("non-positive value returns InvalidArgument", func(t *testing.T) {
		cases := []float64{0, -1, -100.5}
		for _, v := range cases {
			repo := new(mockPaymentRepository)
			h := newTestHandler(repo)

			resp, err := h.CreateBalanceNote(context.Background(), &paymentv1.BalanceNoteRequest{
				UserId: "user-1",
				Value:  v,
			})

			require.Nil(t, resp)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code(), "value=%v", v)
			repo.AssertNotCalled(t, "CreatePayment", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		}
	})

	t.Run("success returns empty response", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		repo.On("CreatePayment", mock.Anything, "user-1", true, 50.0).Return(nil)

		resp, err := h.CreateBalanceNote(context.Background(), &paymentv1.BalanceNoteRequest{
			UserId:  "user-1",
			IsDebit: true,
			Value:   50,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		repo.AssertExpectations(t)
	})

	t.Run("generic repo error maps to Internal", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		repo.On("CreatePayment", mock.Anything, "user-1", false, 50.0).
			Return(errors.New("connection reset"))

		resp, err := h.CreateBalanceNote(context.Background(), &paymentv1.BalanceNoteRequest{
			UserId:  "user-1",
			IsDebit: false,
			Value:   50,
		})

		require.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		repo.AssertExpectations(t)
	})

	t.Run("insufficient balance maps to FailedPrecondition", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		repo.On("CreatePayment", mock.Anything, "user-1", false, 1000.0).
			Return(repository.ErrInsufficientBalance)

		resp, err := h.CreateBalanceNote(context.Background(), &paymentv1.BalanceNoteRequest{
			UserId:  "user-1",
			IsDebit: false,
			Value:   1000,
		})

		require.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "insufficient balance")
		repo.AssertExpectations(t)
	})

	t.Run("wrapped insufficient balance error still maps to FailedPrecondition", func(t *testing.T) {
		repo := new(mockPaymentRepository)
		h := newTestHandler(repo)

		wrapped := fmt.Errorf("create payment: %w", repository.ErrInsufficientBalance)
		repo.On("CreatePayment", mock.Anything, "user-1", false, 1000.0).
			Return(wrapped)

		resp, err := h.CreateBalanceNote(context.Background(), &paymentv1.BalanceNoteRequest{
			UserId:  "user-1",
			IsDebit: false,
			Value:   1000,
		})

		require.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		repo.AssertExpectations(t)
	})
}
