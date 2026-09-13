package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"order-service/internal/model"
	"order-service/internal/repository"
	listingv1 "simple-marketplace-project/gen/listing/v1"
	orderv1 "simple-marketplace-project/gen/order/v1"
	paymentv1 "simple-marketplace-project/gen/payment/v1"
	"simple-marketplace-project/pkg/rabbit"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type OrderHandler struct {
	orderv1.OrderServiceServer
	listingClient  listingv1.ListingServiceClient
	paymentClient  paymentv1.PaymentServiceClient
	repo           repository.OrderRepository
	rabbitProducer rabbit.RabbitProducer
	logger         *slog.Logger
}

// todo: добавить клиентов
func NewHandler(repo repository.OrderRepository,
	rabbitProducer rabbit.RabbitProducer,
	logger *slog.Logger) *OrderHandler {
	return &OrderHandler{repo: repo, rabbitProducer: rabbitProducer, logger: logger}
}

func (h *OrderHandler) GetOrderProducts(ctx context.Context, request *orderv1.OrderRequest) (*orderv1.OrderResponse, error) {
	parsedId, parseErr := uuid.Parse(request.OrderId)
	if parseErr != nil {
		h.logger.Error(parseErr.Error())
		return nil, parseErr
	}
	products, err := h.repo.GetOrderProducts(parsedId)
	if err != nil {
		h.logger.Error(err.Error())
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
				h.logger.Error(grpcErr.Error())
				return grpcErr
			}
			productInfos[i] = info
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		h.logger.Error(err.Error())
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
		h.logger.Error(parseErr.Error())
		return nil, parseErr
	}
	orderStatus, err := h.repo.GetOrderState(parsedId)
	if err != nil {
		h.logger.Error(err.Error())
		return nil, err
	}

	mappedStatus, err := h.mapOrderState(orderStatus)
	if err != nil {
		h.logger.Error(err.Error())
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
		h.logger.Error(s)
		return 0, fmt.Errorf("unknown order state: %q", s)
	}
}

func (h *OrderHandler) CreateNewOrder(ctx context.Context, request *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	createdId, err := h.repo.CreateOrder(h.convertProductsModel(request.Products))
	if err != nil {
		h.logger.Error(err.Error())
		return &orderv1.CreateOrderResponse{}, err
	}

	return &orderv1.CreateOrderResponse{OrderId: createdId.String()}, nil
}

func (h *OrderHandler) convertProductsModel(requestedItems []*listingv1.ShortProductItem) []model.OrderProduct {
	products := make([]model.OrderProduct, len(requestedItems))
	for i, product := range requestedItems {
		productIdUUID, err := uuid.Parse(product.ProductId)
		if err != nil {
			h.logger.Error(err.Error())
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

func (h *OrderHandler) PayOrder(ctx context.Context, request *orderv1.OrderRequest) (*emptypb.Empty, error) {
	orderId, parseErr := uuid.Parse(request.OrderId)
	if parseErr != nil {
		h.logger.Error(parseErr.Error())
		return nil, status.Errorf(codes.InvalidArgument, "invalid orderId: %v", parseErr)
	}

	order, err := h.repo.GetOrderById(ctx, orderId)
	if err != nil {
		h.logger.Error(err.Error())
		if errors.Is(err, errors.New("order not found")) {
			return nil, status.Error(codes.NotFound, "order not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	counts, err := h.repo.GetUnpaidProductCounts(ctx, order.UserID)
	if err != nil {
		h.logger.Error(err.Error())
		return nil, status.Errorf(codes.Internal, "failed to get unpaid products: %v", err)
	}
	if len(counts) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no unpaid orders")
	}

	productInfos := make([]*listingv1.ProductItemInfo, len(counts))
	g, gCtx := errgroup.WithContext(ctx)
	for i, c := range counts {
		g.Go(func() error {
			info, grpcErr := h.listingClient.GetItemInfo(gCtx, &listingv1.ItemInfoRequest{
				ItemId: c.ProductId.String(),
			})
			if grpcErr != nil {
				h.logger.Error(grpcErr.Error())
				return grpcErr
			}
			productInfos[i] = info
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch product info: %v", err)
	}

	var totalValue float64
	for i, c := range counts {
		totalValue += productInfos[i].Price * float64(c.Count)
	}

	_, err = h.paymentClient.CreateBalanceNote(ctx, &paymentv1.BalanceNoteRequest{
		UserId:  order.UserID.String(),
		IsDebit: false,
		Value:   totalValue,
	})
	if err != nil {
		if status.Code(err) == codes.FailedPrecondition {
			return nil, status.Error(codes.FailedPrecondition, "insufficient balance, please top up your account")
		}
		h.logger.Error(err.Error())
		return nil, status.Errorf(codes.Internal, "payment failed: %v", err)
	}

	if err := h.repo.MarkOrdersPaid(ctx, order.UserID); err != nil {
		h.logger.Error(err.Error())
		return nil, status.Errorf(codes.Internal, "payment succeeded but failed to update order status: %v", err)
	}

	return &emptypb.Empty{}, nil
}
