package productservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
	"gostock/internal/service/productservice"
)

// MockProductRepository é uma implementação mock da interface ProductRepository
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Save(ctx context.Context, product domain.Product) (domain.Product, error) {
	args := m.Called(ctx, product)
	return args.Get(0).(domain.Product), args.Error(1)
}

func (m *MockProductRepository) FindByID(ctx context.Context, id string) (domain.Product, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Product), args.Error(1)
}

func (m *MockProductRepository) FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]domain.Product), args.Error(1)
}

// TestGetProducts_Success_NoFilters testa a busca de produtos sem filtros.
func TestGetProducts_Success_NoFilters(t *testing.T) {
	mockRepo := new(MockProductRepository)
	mockLogger := logger.NewLogger("debug")

	svc := productservice.NewService(mockRepo, mockLogger)

	// Dados de teste
	expectedProducts := []domain.Product{
		{ID: uuid.New().String(), Name: "Product A", SKU: "SKU001"},
		{ID: uuid.New().String(), Name: "Product B", SKU: "SKU002"},
	}

	mockRepo.On("FindAll", mock.Anything, domain.ProductFilter{Page: 1, Limit: 10}).Return(expectedProducts, nil)

	ctx := context.Background()
	products, err := svc.GetProducts(ctx, 1, 10, nil)

	assert.NoError(t, err)
	assert.NotNil(t, products)
	assert.Len(t, products, 2)
	assert.Equal(t, expectedProducts, products)
	mockRepo.AssertExpectations(t)
}

// TestGetProducts_Success_WithFilters testa a busca de produtos com filtros.
func TestGetProducts_Success_WithFilters(t *testing.T) {
	mockRepo := new(MockProductRepository)
	mockLogger := logger.NewLogger("debug")

	svc := productservice.NewService(mockRepo, mockLogger)

	// Dados de teste
	expectedProducts := []domain.Product{
		{ID: uuid.New().String(), Name: "Filtered Product", SKU: "SKUFILT"},
	}
	filters := map[string]string{
		"name":       "Filtered",
		"sku":        "SKUFILT",
		"is_active": "true",
	}
	expectedFilter := domain.ProductFilter{
		Page:       1,
		Limit:      10,
		Name:       "Filtered",
		SKU:        "SKUFILT",
		ActiveOnly: true,
	}

	mockRepo.On("FindAll", mock.Anything, expectedFilter).Return(expectedProducts, nil)

	ctx := context.Background()
	products, err := svc.GetProducts(ctx, 1, 10, filters)

	assert.NoError(t, err)
	assert.NotNil(t, products)
	assert.Len(t, products, 1)
	assert.Equal(t, expectedProducts, products)
	mockRepo.AssertExpectations(t)
}

// TestGetProducts_Success_EmptyResults testa quando nenhum produto é encontrado.
func TestGetProducts_Success_EmptyResults(t *testing.T) {
	mockRepo := new(MockProductRepository)
	mockLogger := logger.NewLogger("debug")

	svc := productservice.NewService(mockRepo, mockLogger)

	mockRepo.On("FindAll", mock.Anything, domain.ProductFilter{Page: 1, Limit: 10}).Return([]domain.Product{}, nil)

	ctx := context.Background()
	products, err := svc.GetProducts(ctx, 1, 10, nil)

	assert.NoError(t, err)
	assert.NotNil(t, products)
	assert.Len(t, products, 0)
	mockRepo.AssertExpectations(t)
}

// TestGetProducts_Fail_RepoError testa um erro do repositório.
func TestGetProducts_Fail_RepoError(t *testing.T) {
	mockRepo := new(MockProductRepository)
	mockLogger := logger.NewLogger("debug")

	svc := productservice.NewService(mockRepo, mockLogger)

	// O mock do repositório deve retornar um erro genérico (simulando um erro de DB)
	repoError := errors.New("database connection lost")
	mockRepo.On("FindAll", mock.Anything, domain.ProductFilter{Page: 1, Limit: 10}).Return([]domain.Product{}, repoError)

	ctx := context.Background()
	_, err := svc.GetProducts(ctx, 1, 10, nil)

	assert.Error(t, err)
	// O serviço deve converter o erro genérico do repo para um apperror.InternalError
	assert.IsType(t, &apperror.InternalError{}, err)
	// Verificar se a mensagem de erro contém a parte do serviço e a parte original do repo
	assert.Contains(t, err.Error(), "Erro Interno: Falha interna ao buscar produtos.")
	assert.Contains(t, err.Error(), "database connection lost")
	mockRepo.AssertExpectations(t)
}

// TestGetProducts_LimitSafeguard testa o limite máximo de itens por página.
func TestGetProducts_LimitSafeguard(t *testing.T) {
	mockRepo := new(MockProductRepository)
	mockLogger := logger.NewLogger("debug")

	svc := productservice.NewService(mockRepo, mockLogger)

	// Limite maior que o máximo permitido (100)
	mockRepo.On("FindAll", mock.Anything, domain.ProductFilter{Page: 1, Limit: 100}).Return([]domain.Product{}, nil)

	ctx := context.Background()
	_, err := svc.GetProducts(ctx, 1, 150, nil) // Tenta buscar 150 itens

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	// mockRepo.AssertCalled(t, "FindAll", mock.Anything, domain.ProductFilter{Page: 1, Limit: 100}) // Verifica que o limite foi ajustado
}

// TestGetProducts_InvalidPageOrLimit testa valores inválidos para page/limit
func TestGetProducts_InvalidPageOrLimit(t *testing.T) {
	mockRepo := new(MockProductRepository)
	mockLogger := logger.NewLogger("debug")

	svc := productservice.NewService(mockRepo, mockLogger)

	// Page < 1 deve ser ajustado para 1 no repo
	mockRepo.On("FindAll", mock.Anything, domain.ProductFilter{Page: 1, Limit: 10}).Return([]domain.Product{}, nil).Once()

	ctx := context.Background()
	_, err := svc.GetProducts(ctx, 0, 10, nil) // Page 0 should result in Page 1 to repo

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockRepo.ExpectedCalls = nil // Clear expectations for the next sub-test

	// Limit <= 0 deve ser passado como 0 para o repo, e o repo aplica o default de 10
	mockRepo.On("FindAll", mock.Anything, domain.ProductFilter{Page: 1, Limit: 0}).Return([]domain.Product{}, nil).Once()

	ctx = context.Background()
	_, err = svc.GetProducts(ctx, 1, 0, nil) // Limit 0 should be passed as 0 to repo

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}



