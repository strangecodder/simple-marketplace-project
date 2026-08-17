package handler

import (
	"context"
	listingv1 "gen/listing/v1"
	"listing-service/internal/repository"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Handler struct {
	listingv1.ListingServiceServer
	repo repository.Repository
}

func NewHandler(repo repository.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetAllItems(ctx context.Context, empty *emptypb.Empty) (*listingv1.AllItemsResponse, error) {
	products, err := h.repo.FindAllProducts(ctx)
	if err != nil {
		return nil, err
	}
	var responseArray []*listingv1.ShortProductItem
	for _, product := range products {
		responseArray = append(responseArray, &listingv1.ShortProductItem{ProductId: product.ProductId.String(), Price: product.Price, Count: product.Count})
	}
	return &listingv1.AllItemsResponse{Items: responseArray}, nil
}

func (h *Handler) GetItemInfo(ctx context.Context, req *listingv1.ItemInfoRequest) (*listingv1.ProductItemInfo, error) {
	parseUUID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, err
	}
	item, err := h.repo.FindProductById(ctx, parseUUID)
	if err != nil {
		return nil, err
	}

	return &listingv1.ProductItemInfo{
		ProductId:   item.ProductId.String(),
		Name:        item.Name,
		Description: item.Description,
		Price:       item.Price,
		Count:       item.Count,
	}, nil
}

func (h *Handler) CreateNewItem(ctx context.Context, req *listingv1.NewItemRequest) (*listingv1.ItemInfoRequest, error) {
	parseUUID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, err
	}
	createdId, err := h.repo.CreateProduct(ctx, parseUUID, req.Name, req.Description, req.Price)
	if err != nil {
		return nil, err
	}
	return &listingv1.ItemInfoRequest{ItemId: createdId.String()}, nil
}

func (h *Handler) UpdateProduct(ctx context.Context, req *listingv1.NewItemRequest) (*emptypb.Empty, error) {
	panic("implement me")

}

func (h *Handler) GetAllItemInfo(ctx context.Context, req *listingv1.ItemInfoRequest) (*emptypb.Empty, error) {
	parseUUID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, err
	}
	err = h.repo.DeleteProduct(ctx, parseUUID)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
