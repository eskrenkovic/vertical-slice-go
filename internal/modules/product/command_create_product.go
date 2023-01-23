package product

import (
	"context"
	"fmt"

	"github.com/eskrenkovic/vertical-slice-go/internal/modules/core"
	"github.com/google/uuid"
)

type CreateProductCommand struct {
	Product CreateProductModel `json:"product"`
}

type CreateProductResponse struct {
	ProductID uuid.UUID
}

func (c CreateProductCommand) Validate() error {
	var validationErrors []error
	if c.Product.SKU == "" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("empty SKU"),
		)
	}

	if c.Product.Name == "" {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("empty product name"),
		)
	}

	if len(validationErrors) > 0 {
		return core.ValidationError{ValidationErrors: validationErrors}
	}

	return nil
}

type CreateProductHandler struct {
	repository *ProductRepository
}

func NewCreateProductHandler(repository *ProductRepository) *CreateProductHandler {
	return &CreateProductHandler{repository}
}

func (h *CreateProductHandler) Handle(ctx context.Context, request CreateProductCommand) (CreateProductResponse, error) {
	productID := uuid.New()
	product := request.Product
	err := h.repository.SaveProduct(ctx, Product{
		ID:          productID,
		SKU:         product.SKU,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
	})
	if err != nil {
		return CreateProductResponse{}, core.NewCommandError(500, err, "failed to insert product to database")
	}

	return CreateProductResponse{ProductID: productID}, nil
}
