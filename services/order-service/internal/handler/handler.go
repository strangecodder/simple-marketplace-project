package handler

import (
	"context"
	"fmt"
	"order-service/internal/model"
	"order-service/internal/repository"
	listingv1 "simple-marketplace-project/gen/listing/v1"
	orderv1 "simple-marketplace-project/gen/order/v1"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/emptypb"
)

type OrderHandler struct {
	orderv1.OrderServiceServer
	listingClient listingv1.ListingServiceClient
	repo          repository.OrderRepository
}

func NewHandler(repo repository.OrderRepository) *OrderHandler {
	return &OrderHandler{repo: repo}
}

func (h *OrderHandler) GetOrderProducts(ctx context.Context, request *orderv1.OrderRequest) (*orderv1.OrderResponse, error) {
	parsedId, parseErr := uuid.Parse(request.OrderId)
	if parseErr != nil {
		return nil, parseErr
	}
	products, err := h.repo.GetOrderProducts(parsedId)
	if err != nil {
		return nil, err
	}

	productInfos := make([]*listingv1.ProductItemInfo, len(products))

	g, gCtx := errgroup.WithContext(ctx)
	for i, product := range products {
		g.Go(func() error {
			info, grpcErr := h.listingClient.GetItemInfo(gCtx, &listingv1.ItemInfoRequest{
				ItemId: product.ProductId.String(),
			})
			if grpcErr != nil {
				return grpcErr
			}
			productInfos[i] = info
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	items := make([]*orderv1.ProductItem, len(products))
	for i, product := range products {
		info := productInfos[i]
		totalPrice := info.Price * float64(product.Count)

		items[i] = &orderv1.ProductItem{
			OrderId:      request.OrderId,
			Name:         info.Name,
			Description:  info.Description,
			Count:        product.Count,
			PricePerItem: info.Price,
			TotalPrice:   totalPrice,
		}
	}

	return &orderv1.OrderResponse{
		Products: items,
	}, nil
}

func (h *OrderHandler) GetOrderState(ctx context.Context, request *orderv1.OrderRequest) (*orderv1.OrderStateResponse, error) {
	parsedId, parseErr := uuid.Parse(request.OrderId)
	if parseErr != nil {
		return nil, parseErr
	}
	orderStatus, err := h.repo.GetOrderState(parsedId)
	if err != nil {
		return nil, err
	}

	mappedStatus, err := h.mapOrderState(orderStatus)
	if err != nil {
		return nil, err
	}

	return &orderv1.OrderStateResponse{State: mappedStatus}, nil
}

func (h *OrderHandler) mapOrderState(s string) (orderv1.OrderState, error) {
	switch s {
	case "CREATED":
		return orderv1.OrderState_CREATED, nil
	case "WAIT_PAID":
		return orderv1.OrderState_WAIT_PAID, nil
	case "PAID":
		return orderv1.OrderState_PAID, nil
	case "REJECTED":
		return orderv1.OrderState_REJECTED, nil
	default:
		return 0, fmt.Errorf("unknown order state: %q", s)
	}
}

func (h *OrderHandler) CreateNewOrder(ctx context.Context, request *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	createdId, err := h.repo.CreateOrder(h.convertProductsModel(request.Products))
	if err != nil || createdId == uuid.Nil {
		return &orderv1.CreateOrderResponse{}, err
	}

	return &orderv1.CreateOrderResponse{OrderId: createdId.String()}, nil
}

func (h *OrderHandler) convertProductsModel(requestedItems []*listingv1.ShortProductItem) []model.OrderProduct {
	products := make([]model.OrderProduct, len(requestedItems))
	for i, product := range requestedItems {
		productIdUUID, err := uuid.Parse(product.ProductId)
		if err != nil {
			return []model.OrderProduct{}
		}
		products[i] = model.OrderProduct{ProductId: productIdUUID, Count: product.Count}
	}
	return products
}

// todo: добавить отмену заказа
func (h *OrderHandler) RejectOrder(ctx context.Context, request *orderv1.RejectOrderRequest) (*emptypb.Empty, error) {
	panic("implement me")
}
