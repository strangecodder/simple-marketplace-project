package handler

import (
	"context"
	"errors"
	"io"
	"listing-service/internal/model"
	"log/slog"
	"testing"

	listingv1 "simple-marketplace-project/gen/listing/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/emptypb"
)

// --- Mock repository ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindAllProducts(ctx context.Context) ([]model.Product, error) {
	args := m.Called(ctx)
	var products []model.Product
	if args.Get(0) != nil {
		products = args.Get(0).([]model.Product)
	}
	return products, args.Error(1)
}

func (m *MockRepository) FindProductById(ctx context.Context, id uuid.UUID) (model.Product, error) {
	args := m.Called(ctx, id)
	var p model.Product
	if args.Get(0) != nil {
		p = args.Get(0).(model.Product)
	}
	return p, args.Error(1)
}

func (m *MockRepository) CreateProduct(ctx context.Context, sellerId uuid.UUID, name, description string, price float64) (uuid.UUID, error) {
	args := m.Called(ctx, sellerId, name, description, price)
	var id uuid.UUID
	if args.Get(0) != nil {
		id = args.Get(0).(uuid.UUID)
	}
	return id, args.Error(1)
}

func (m *MockRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) UpdateProduct(ctx context.Context, productId uuid.UUID, sellerId uuid.UUID, name string, description string, price int64) error {
	args := m.Called(ctx, productId, sellerId, name, description, price)
	return args.Error(0)
}

// silentLogger возвращает slog.Logger, который никуда не пишет — чтобы тесты
// не засоряли вывод, но код логирования всё равно исполнялся.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- GetAllItems ---

func TestGetAllItems_Success(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	productId := uuid.New()
	repo.On("FindAllProducts", mock.Anything).Return([]model.Product{
		{ProductId: productId, Name: "Item 1", Description: "desc", Price: 100, Count: 5},
	}, nil)

	resp, err := h.GetAllItems(context.Background(), &emptypb.Empty{})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, productId.String(), resp.Items[0].ProductId)
	assert.Equal(t, int64(100), resp.Items[0].Price)
	assert.Equal(t, int64(5), resp.Items[0].Count)
	repo.AssertExpectations(t)
}

func TestGetAllItems_EmptyList(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	repo.On("FindAllProducts", mock.Anything).Return([]model.Product{}, nil)

	resp, err := h.GetAllItems(context.Background(), &emptypb.Empty{})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Items, 0)
	repo.AssertExpectations(t)
}

func TestGetAllItems_RepoError(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	repo.On("FindAllProducts", mock.Anything).Return(nil, errors.New("db connection failed"))

	resp, err := h.GetAllItems(context.Background(), &emptypb.Empty{})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

// --- GetItemInfo ---

func TestGetItemInfo_Success(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	productId := uuid.New()
	repo.On("FindProductById", mock.Anything, productId).Return(model.Product{
		ProductId:   productId,
		Name:        "Item 1",
		Description: "desc",
		Price:       200,
		Count:       3,
	}, nil)

	resp, err := h.GetItemInfo(context.Background(), &listingv1.ItemInfoRequest{ItemId: productId.String()})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, productId.String(), resp.ProductId)
	assert.Equal(t, "Item 1", resp.Name)
	assert.Equal(t, "desc", resp.Description)
	assert.Equal(t, int64(200), resp.Price)
	assert.Equal(t, int64(3), resp.Count)
	repo.AssertExpectations(t)
}

func TestGetItemInfo_InvalidUUID(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	resp, err := h.GetItemInfo(context.Background(), &listingv1.ItemInfoRequest{ItemId: "not-a-uuid"})

	assert.Error(t, err)
	assert.Nil(t, resp)
	// repo не должен вызываться, если парсинг UUID упал раньше
	repo.AssertNotCalled(t, "FindProductById", mock.Anything, mock.Anything)
}

func TestGetItemInfo_NotFound(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	productId := uuid.New()
	repo.On("FindProductById", mock.Anything, productId).Return(model.Product{}, errors.New("not found"))

	resp, err := h.GetItemInfo(context.Background(), &listingv1.ItemInfoRequest{ItemId: productId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

// --- CreateNewItem ---

func TestCreateNewItem_Success(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	sellerId := uuid.New()
	createdId := uuid.New()

	repo.On("CreateProduct", mock.Anything, sellerId, "New Item", "desc", int64(500)).
		Return(createdId, nil)

	resp, err := h.CreateNewItem(context.Background(), &listingv1.NewItemRequest{
		SellerId:    sellerId.String(),
		Name:        "New Item",
		Description: "desc",
		Price:       500,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, createdId.String(), resp.ItemId)
	repo.AssertExpectations(t)
}

func TestCreateNewItem_InvalidSellerUUID(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	resp, err := h.CreateNewItem(context.Background(), &listingv1.NewItemRequest{
		SellerId:    "not-a-uuid",
		Name:        "New Item",
		Description: "desc",
		Price:       500,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertNotCalled(t, "CreateProduct", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateNewItem_RepoError(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	sellerId := uuid.New()
	repo.On("CreateProduct", mock.Anything, sellerId, "New Item", "desc", int64(500)).
		Return(uuid.Nil, errors.New("db insert failed"))

	resp, err := h.CreateNewItem(context.Background(), &listingv1.NewItemRequest{
		SellerId:    sellerId.String(),
		Name:        "New Item",
		Description: "desc",
		Price:       500,
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

// --- GetAllItemInfo ---
// ВНИМАНИЕ: несмотря на название, метод фактически вызывает h.repo.DeleteProduct,
// а не операцию чтения. Похоже на баг (копипаста из DeleteProduct-хендлера с
// неверным именем). Тесты ниже проверяют РЕАЛЬНОЕ поведение метода как он
// написан сейчас; если это баг — переименуйте метод/поправьте логику и тесты
// нужно будет соответственно обновить.

func TestGetAllItemInfo_Success(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	productId := uuid.New()
	repo.On("DeleteProduct", mock.Anything, productId).Return(nil)

	resp, err := h.GetAllItemInfo(context.Background(), &listingv1.ItemInfoRequest{ItemId: productId.String()})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	repo.AssertExpectations(t)
}

func TestGetAllItemInfo_InvalidUUID(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	resp, err := h.GetAllItemInfo(context.Background(), &listingv1.ItemInfoRequest{ItemId: "not-a-uuid"})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertNotCalled(t, "DeleteProduct", mock.Anything, mock.Anything)
}

func TestGetAllItemInfo_RepoError(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	productId := uuid.New()
	repo.On("DeleteProduct", mock.Anything, productId).Return(errors.New("delete failed"))

	resp, err := h.GetAllItemInfo(context.Background(), &listingv1.ItemInfoRequest{ItemId: productId.String()})

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

// --- UpdateProduct ---
// Метод сейчас паникует (panic("implement me")), поэтому тест фиксирует
// текущее поведение через recover. Как только метод будет реализован,
// замените этот тест на нормальные кейсы (успех/ошибка/невалидный UUID).

func TestUpdateProduct_PanicsWhenNotImplemented(t *testing.T) {
	repo := new(MockRepository)
	h := NewHandler(repo, silentLogger())

	assert.Panics(t, func() {
		_, _ = h.UpdateProduct(context.Background(), &listingv1.NewItemRequest{})
	})
}
