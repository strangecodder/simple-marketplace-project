package handler

// ВАЖНО — предположения о типах, которые пришлось восстановить (proto-файлов
// и model.go у меня нет):
//
//   model.OrderProduct  { ProductId uuid.UUID; Count int64 }
//   model.ProductCount  { ProductId uuid.UUID; Count int64 }
//   model.Order         { OrderID uuid.UUID; UserID uuid.UUID }
//
//   listingv1.ProductItemInfo { ProductId, Name, Description string; Price float64; Count int64 }
//   listingv1.ShortProductItem{ ProductId string; Price float64; Count int64 }
//   listingv1.ItemInfoRequest { ItemId string }
//   listingv1.NewItemRequest  { SellerId, Name, Description string; Price float64 }
//   listingv1.AllItemsResponse{ Items []*ShortProductItem }
//
//   orderv1.ProductItem  { OrderId, Name, Description string; Count int64; PricePerItem, TotalPrice float64 }
//   orderv1.OrderResponse{ Products []*ProductItem }
//
//   paymentv1.BalanceNoteRequest{ UserId string; IsDebit bool; Value float64 }
//
// ListingServiceClient и PaymentServiceClient — сгенерированные gRPC-клиенты.
// Я реализовал в моках только те методы, что видел в предыдущих хендлерах
// (Listing: GetAllItems, GetItemInfo, CreateNewItem, UpdateProduct,
// GetAllItemInfo; Payment: CreateBalanceNote). Если в реальных .proto есть
// другие rpc — компилятор укажет на недостающие методы, допишите их в мок
// по аналогии.
//
// ==================== ОБНАРУЖЕННЫЕ БАГИ В HANDLER.GO ====================
//
// 1. PayOrder: `errors.Is(err, errors.New("order not found"))` — ВСЕГДА false.
//    errors.New создаёт новый уникальный объект при каждом вызове, поэтому
//    сравнение с ошибкой, пришедшей из репозитория, никогда не совпадёт.
//    Ветка codes.NotFound фактически недостижима — любая ошибка GetOrderById,
//    включая "order not found", улетает в codes.Internal.
//    Исправление: завести sentinel-ошибку в repository (var ErrOrderNotFound
//    = errors.New("order not found")) и сравнивать errors.Is(err, repository.ErrOrderNotFound).
//
// 2. convertProductsModel: если хотя бы один product_id невалиден, функция
//    молча возвращает пустой []model.OrderProduct{} и ПРОГЛАТЫВАЕТ ошибку —
//    CreateNewOrder узнает об этом, только что придёт createOrder с 0
//    товаров, а не получит явную ошибку InvalidArgument.
//
// 3. CreateNewOrder: при ошибке репозитория возвращается НЕ nil, а
//    &orderv1.CreateOrderResponse{} вместе с err — необычно для gRPC-хендлера
//    (обычно на error response должен быть nil).
//
// Тесты ниже фиксируют РЕАЛЬНОЕ (текущее) поведение с пометкой "БАГ" —
// когда почините логику, эти тесты нужно будет переписать под новое поведение.

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"order-service/internal/model"
	"order-service/internal/repository"

	listingv1 "simple-marketplace-project/gen/listing/v1"
	orderv1 "simple-marketplace-project/gen/order/v1"
	paymentv1 "simple-marketplace-project/gen/payment/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ==================== Моки ====================

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) GetOrderProducts(orderId uuid.UUID) ([]model.OrderProduct, error) {
	args := m.Called(orderId)
	var products []model.OrderProduct
	if args.Get(0) != nil {
		products = args.Get(0).([]model.OrderProduct)
	}
	return products, args.Error(1)
}

func (m *MockOrderRepository) GetOrderState(orderId uuid.UUID) (string, error) {
	args := m.Called(orderId)
	return args.String(0), args.Error(1)
}

func (m *MockOrderRepository) CreateOrder(products []model.OrderProduct) (uuid.UUID, error) {
	args := m.Called(products)
	var id uuid.UUID
	if args.Get(0) != nil {
		id = args.Get(0).(uuid.UUID)
	}
	return id, args.Error(1)
}

func (m *MockOrderRepository) RejectOrder(orderId uuid.UUID) error {
	args := m.Called(orderId)
	return args.Error(0)
}

func (m *MockOrderRepository) GetUnpaidProductCounts(ctx context.Context, userId uuid.UUID) ([]model.ProductCount, error) {
	args := m.Called(ctx, userId)
	var counts []model.ProductCount
	if args.Get(0) != nil {
		counts = args.Get(0).([]model.ProductCount)
	}
	return counts, args.Error(1)
}

func (m *MockOrderRepository) GetOrderById(ctx context.Context, orderId uuid.UUID) (*model.Order, error) {
	args := m.Called(ctx, orderId)
	var order *model.Order
	if args.Get(0) != nil {
		order = args.Get(0).(*model.Order)
	}
	return order, args.Error(1)
}

func (m *MockOrderRepository) MarkOrdersPaid(ctx context.Context, orderId uuid.UUID) error {
	args := m.Called(ctx, orderId)
	return args.Error(0)
}

var _ repository.OrderRepository = (*MockOrderRepository)(nil)

type MockListingClient struct {
	mock.Mock
}

func (m *MockListingClient) DeleteItem(ctx context.Context, in *listingv1.ItemInfoRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockListingClient) GetAllItems(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*listingv1.AllItemsResponse, error) {
	args := m.Called(ctx, in)
	var resp *listingv1.AllItemsResponse
	if args.Get(0) != nil {
		resp = args.Get(0).(*listingv1.AllItemsResponse)
	}
	return resp, args.Error(1)
}

func (m *MockListingClient) GetItemInfo(ctx context.Context, in *listingv1.ItemInfoRequest, opts ...grpc.CallOption) (*listingv1.ProductItemInfo, error) {
	args := m.Called(ctx, in)
	var resp *listingv1.ProductItemInfo
	if args.Get(0) != nil {
		resp = args.Get(0).(*listingv1.ProductItemInfo)
	}
	return resp, args.Error(1)
}

func (m *MockListingClient) CreateNewItem(ctx context.Context, in *listingv1.NewItemRequest, opts ...grpc.CallOption) (*listingv1.ItemInfoRequest, error) {
	args := m.Called(ctx, in)
	var resp *listingv1.ItemInfoRequest
	if args.Get(0) != nil {
		resp = args.Get(0).(*listingv1.ItemInfoRequest)
	}
	return resp, args.Error(1)
}

func (m *MockListingClient) UpdateProduct(ctx context.Context, in *listingv1.NewItemRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	var resp *emptypb.Empty
	if args.Get(0) != nil {
		resp = args.Get(0).(*emptypb.Empty)
	}
	return resp, args.Error(1)
}

func (m *MockListingClient) GetAllItemInfo(ctx context.Context, in *listingv1.ItemInfoRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	var resp *emptypb.Empty
	if args.Get(0) != nil {
		resp = args.Get(0).(*emptypb.Empty)
	}
	return resp, args.Error(1)
}

var _ listingv1.ListingServiceClient = (*MockListingClient)(nil)

type MockPaymentClient struct {
	mock.Mock
}

func (m *MockPaymentClient) CreateBalanceNote(ctx context.Context, in *paymentv1.BalanceNoteRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	args := m.Called(ctx, in)
	var resp *emptypb.Empty
	if args.Get(0) != nil {
		resp = args.Get(0).(*emptypb.Empty)
	}
	return resp, args.Error(1)
}

func (m *MockPaymentClient) GetBalance(ctx context.Context, in *paymentv1.BalanceRequest, opts ...grpc.CallOption) (*paymentv1.BalanceResponse, error) {
	args := m.Called(ctx, in)
	var resp *paymentv1.BalanceResponse
	if args.Get(0) != nil {
		resp = args.Get(0).(*paymentv1.BalanceResponse)
	}
	return resp, args.Error(1)
}

var _ paymentv1.PaymentServiceClient = (*MockPaymentClient)(nil)

// ==================== Хелперы ====================

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestHandler создаёт OrderHandler напрямую через литерал структуры (а не
// NewHandler), т.к. NewHandler не принимает listingClient/paymentClient —
// поля неэкспортированы, но тест лежит в том же пакете handler, так что
// прямой доступ работает.
func newTestHandler(repo repository.OrderRepository, listingClient listingv1.ListingServiceClient, paymentClient paymentv1.PaymentServiceClient) *OrderHandler {
	return &OrderHandler{
		repo:          repo,
		listingClient: listingClient,
		paymentClient: paymentClient,
		logger:        silentLogger(),
	}
}

// ==================== GetOrderProducts ====================

func TestGetOrderProducts_Success(t *testing.T) {
	orderId := uuid.New()
	productId := uuid.New()

	repo := new(MockOrderRepository)
	listingClient := new(MockListingClient)
	h := newTestHandler(repo, listingClient, nil)

	repo.On("GetOrderProducts", orderId).Return([]model.OrderProduct{
		{ProductId: productId, Count: 2},
	}, nil)
	listingClient.On("GetItemInfo", mock.Anything, &listingv1.ItemInfoRequest{ItemId: productId.String()}).
		Return(&listingv1.ProductItemInfo{
			ProductId:   productId.String(),
			Name:        "Item",
			Description: "desc",
			Price:       50,
			Count:       10,
		}, nil)

	resp, err := h.GetOrderProducts(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Products, 1)
	assert.Equal(t, orderId.String(), resp.Products[0].OrderId)
	assert.Equal(t, "Item", resp.Products[0].Name)
	assert.Equal(t, int64(2), resp.Products[0].Count)
	assert.Equal(t, float64(50), resp.Products[0].PricePerItem)
	assert.Equal(t, float64(100), resp.Products[0].TotalPrice)
	repo.AssertExpectations(t)
	listingClient.AssertExpectations(t)
}

func TestGetOrderProducts_InvalidUUID(t *testing.T) {
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	resp, err := h.GetOrderProducts(context.Background(), &orderv1.OrderRequest{OrderId: "not-a-uuid"})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertNotCalled(t, "GetOrderProducts", mock.Anything)
}

func TestGetOrderProducts_RepoError(t *testing.T) {
	orderId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderProducts", orderId).Return(nil, errors.New("db error"))

	resp, err := h.GetOrderProducts(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

func TestGetOrderProducts_ListingClientError(t *testing.T) {
	orderId := uuid.New()
	productId := uuid.New()
	repo := new(MockOrderRepository)
	listingClient := new(MockListingClient)
	h := newTestHandler(repo, listingClient, nil)

	repo.On("GetOrderProducts", orderId).Return([]model.OrderProduct{{ProductId: productId, Count: 1}}, nil)
	listingClient.On("GetItemInfo", mock.Anything, mock.Anything).Return(nil, errors.New("listing unavailable"))

	resp, err := h.GetOrderProducts(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ==================== mapOrderState ====================

func TestMapOrderState(t *testing.T) {
	h := newTestHandler(nil, nil, nil)

	tests := []struct {
		input    string
		expected orderv1.OrderState
		wantErr  bool
	}{
		{"CREATED", orderv1.OrderState_CREATED, false},
		{"WAIT_PAID", orderv1.OrderState_WAIT_PAID, false},
		{"PAID", orderv1.OrderState_PAID, false},
		{"REJECTED", orderv1.OrderState_REJECTED, false},
		{"UNKNOWN_STATE", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := h.mapOrderState(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, got)
			}
		})
	}
}

// ==================== GetOrderState ====================

func TestGetOrderState_Success(t *testing.T) {
	orderId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderState", orderId).Return("PAID", nil)

	resp, err := h.GetOrderState(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, orderv1.OrderState_PAID, resp.State)
	repo.AssertExpectations(t)
}

func TestGetOrderState_InvalidUUID(t *testing.T) {
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	resp, err := h.GetOrderState(context.Background(), &orderv1.OrderRequest{OrderId: "bad-id"})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertNotCalled(t, "GetOrderState", mock.Anything)
}

func TestGetOrderState_RepoError(t *testing.T) {
	orderId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderState", orderId).Return("", errors.New("db error"))

	resp, err := h.GetOrderState(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetOrderState_UnknownStatusFromRepo(t *testing.T) {
	orderId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderState", orderId).Return("SOME_GARBAGE_STATUS", nil)

	resp, err := h.GetOrderState(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

// ==================== CreateNewOrder ====================

func TestCreateNewOrder_Success(t *testing.T) {
	productId := uuid.New()
	createdId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("CreateOrder", []model.OrderProduct{{ProductId: productId, Count: 3}}).
		Return(createdId, nil)

	resp, err := h.CreateNewOrder(context.Background(), &orderv1.CreateOrderRequest{
		Products: []*listingv1.ShortProductItem{
			{ProductId: productId.String(), Count: 3, Price: 10},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, createdId.String(), resp.OrderId)
	repo.AssertExpectations(t)
}

func TestCreateNewOrder_RepoError(t *testing.T) {
	productId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("CreateOrder", []model.OrderProduct{{ProductId: productId, Count: 1}}).
		Return(uuid.Nil, errors.New("insert failed"))

	resp, err := h.CreateNewOrder(context.Background(), &orderv1.CreateOrderRequest{
		Products: []*listingv1.ShortProductItem{{ProductId: productId.String(), Count: 1}},
	})

	assert.Error(t, err)
	// БАГ: хендлер возвращает не nil, а пустой &CreateOrderResponse{} даже
	// при ошибке — фиксируем текущее поведение.
	assert.NotNil(t, resp)
	assert.Equal(t, "", resp.OrderId)
}

func TestCreateNewOrder_InvalidProductId_SilentlyDropsAll(t *testing.T) {
	// БАГ: convertProductsModel при невалидном UUID молча возвращает пустой
	// слайс и не пробрасывает ошибку наверх. CreateNewOrder уходит в repo
	// с пустым списком товаров вместо InvalidArgument.
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("CreateOrder", mock.MatchedBy(func(products []model.OrderProduct) bool {
		return len(products) == 0
	})).Return(uuid.New(), nil)

	resp, err := h.CreateNewOrder(context.Background(), &orderv1.CreateOrderRequest{
		Products: []*listingv1.ShortProductItem{{ProductId: "not-a-uuid", Count: 1}},
	})

	// Текущее поведение: ошибки парсинга не будет, запрос "успешно" создаст
	// заказ без товаров.
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	repo.AssertExpectations(t)
}

// ==================== RejectOrder ====================

func TestRejectOrder_PanicsWhenNotImplemented(t *testing.T) {
	h := newTestHandler(nil, nil, nil)

	assert.Panics(t, func() {
		_, _ = h.RejectOrder(context.Background(), &orderv1.RejectOrderRequest{})
	})
}

// ==================== PayOrder ====================

func TestPayOrder_InvalidUUID(t *testing.T) {
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: "bad-id"})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestPayOrder_OrderNotFound(t *testing.T) {
	orderId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderById", mock.Anything, orderId).Return(nil, repository.ErrOrderNotFound)

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestPayOrder_GetOrderByIdOtherError_MapsToInternal(t *testing.T) {
	orderId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderById", mock.Anything, orderId).Return(nil, errors.New("connection reset"))

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestPayOrder_GetUnpaidCountsError(t *testing.T) {
	orderId := uuid.New()
	userId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderById", mock.Anything, orderId).Return(&model.Order{OrderID: orderId, UserID: userId}, nil)
	repo.On("GetUnpaidProductCounts", mock.Anything, userId).Return(nil, errors.New("db error"))

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestPayOrder_NoUnpaidOrders(t *testing.T) {
	orderId := uuid.New()
	userId := uuid.New()
	repo := new(MockOrderRepository)
	h := newTestHandler(repo, nil, nil)

	repo.On("GetOrderById", mock.Anything, orderId).Return(&model.Order{OrderID: orderId, UserID: userId}, nil)
	repo.On("GetUnpaidProductCounts", mock.Anything, userId).Return([]model.ProductCount{}, nil)

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
}

func TestPayOrder_ListingClientError(t *testing.T) {
	orderId := uuid.New()
	userId := uuid.New()
	productId := uuid.New()
	repo := new(MockOrderRepository)
	listingClient := new(MockListingClient)
	h := newTestHandler(repo, listingClient, nil)

	repo.On("GetOrderById", mock.Anything, orderId).Return(&model.Order{OrderID: orderId, UserID: userId}, nil)
	repo.On("GetUnpaidProductCounts", mock.Anything, userId).Return([]model.ProductCount{
		{ProductId: productId, Count: 2},
	}, nil)
	listingClient.On("GetItemInfo", mock.Anything, mock.Anything).Return(nil, errors.New("listing down"))

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestPayOrder_InsufficientBalance(t *testing.T) {
	orderId := uuid.New()
	userId := uuid.New()
	productId := uuid.New()
	repo := new(MockOrderRepository)
	listingClient := new(MockListingClient)
	paymentClient := new(MockPaymentClient)
	h := newTestHandler(repo, listingClient, paymentClient)

	repo.On("GetOrderById", mock.Anything, orderId).Return(&model.Order{OrderID: orderId, UserID: userId}, nil)
	repo.On("GetUnpaidProductCounts", mock.Anything, userId).Return([]model.ProductCount{
		{ProductId: productId, Count: 2},
	}, nil)
	listingClient.On("GetItemInfo", mock.Anything, &listingv1.ItemInfoRequest{ItemId: productId.String()}).
		Return(&listingv1.ProductItemInfo{Price: 100}, nil)
	paymentClient.On("CreateBalanceNote", mock.Anything, &paymentv1.BalanceNoteRequest{
		UserId:  userId.String(),
		IsDebit: false,
		Value:   200,
	}).Return(nil, status.Error(codes.FailedPrecondition, "insufficient balance"))

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
	assert.Contains(t, st.Message(), "top up")
	repo.AssertNotCalled(t, "MarkOrdersPaid", mock.Anything, mock.Anything)
}

func TestPayOrder_PaymentInternalError(t *testing.T) {
	orderId := uuid.New()
	userId := uuid.New()
	productId := uuid.New()
	repo := new(MockOrderRepository)
	listingClient := new(MockListingClient)
	paymentClient := new(MockPaymentClient)
	h := newTestHandler(repo, listingClient, paymentClient)

	repo.On("GetOrderById", mock.Anything, orderId).Return(&model.Order{OrderID: orderId, UserID: userId}, nil)
	repo.On("GetUnpaidProductCounts", mock.Anything, userId).Return([]model.ProductCount{
		{ProductId: productId, Count: 1},
	}, nil)
	listingClient.On("GetItemInfo", mock.Anything, mock.Anything).
		Return(&listingv1.ProductItemInfo{Price: 50}, nil)
	paymentClient.On("CreateBalanceNote", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.Unavailable, "payment service down"))

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestPayOrder_MarkOrdersPaidError(t *testing.T) {
	orderId := uuid.New()
	userId := uuid.New()
	productId := uuid.New()
	repo := new(MockOrderRepository)
	listingClient := new(MockListingClient)
	paymentClient := new(MockPaymentClient)
	h := newTestHandler(repo, listingClient, paymentClient)

	repo.On("GetOrderById", mock.Anything, orderId).Return(&model.Order{OrderID: orderId, UserID: userId}, nil)
	repo.On("GetUnpaidProductCounts", mock.Anything, userId).Return([]model.ProductCount{
		{ProductId: productId, Count: 1},
	}, nil)
	listingClient.On("GetItemInfo", mock.Anything, mock.Anything).
		Return(&listingv1.ProductItemInfo{Price: 50}, nil)
	paymentClient.On("CreateBalanceNote", mock.Anything, mock.Anything).
		Return(&emptypb.Empty{}, nil)
	repo.On("MarkOrdersPaid", mock.Anything, userId).Return(errors.New("update failed"))

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	// Важный кейс: деньги уже списаны (payment client отработал успешно),
	// но статус заказа не проставился — возвращается ошибка, но списание
	// уже необратимо. См. обсуждение saga/компенсации в диалоге.
	assert.Error(t, err)
	assert.Nil(t, resp)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestPayOrder_Success(t *testing.T) {
	orderId := uuid.New()
	userId := uuid.New()
	productId := uuid.New()
	repo := new(MockOrderRepository)
	listingClient := new(MockListingClient)
	paymentClient := new(MockPaymentClient)
	h := newTestHandler(repo, listingClient, paymentClient)

	repo.On("GetOrderById", mock.Anything, orderId).Return(&model.Order{OrderID: orderId, UserID: userId}, nil)
	repo.On("GetUnpaidProductCounts", mock.Anything, userId).Return([]model.ProductCount{
		{ProductId: productId, Count: 2},
	}, nil)
	listingClient.On("GetItemInfo", mock.Anything, &listingv1.ItemInfoRequest{ItemId: productId.String()}).
		Return(&listingv1.ProductItemInfo{Price: 100}, nil)
	paymentClient.On("CreateBalanceNote", mock.Anything, &paymentv1.BalanceNoteRequest{
		UserId:  userId.String(),
		IsDebit: false,
		Value:   200,
	}).Return(&emptypb.Empty{}, nil)
	repo.On("MarkOrdersPaid", mock.Anything, userId).Return(nil)

	resp, err := h.PayOrder(context.Background(), &orderv1.OrderRequest{OrderId: orderId.String()})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	repo.AssertExpectations(t)
	listingClient.AssertExpectations(t)
	paymentClient.AssertExpectations(t)
}
