package stockservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
	"gostock/internal/service/stockservice"
)

// MockStockRepository é uma implementação mock da interface StockRepository
type MockStockRepository struct {
	mock.Mock
}

func (m *MockStockRepository) GetStockLevel(ctx context.Context, variantID, warehouseID string) (domain.StockLevel, error) {
	args := m.Called(ctx, variantID, warehouseID)
	return args.Get(0).(domain.StockLevel), args.Error(1)
}

func (m *MockStockRepository) UpdateStockLevel(ctx context.Context, adjustment domain.StockAdjustmentRequest) (domain.StockLevel, error) {
	args := m.Called(ctx, adjustment)
	return args.Get(0).(domain.StockLevel), args.Error(1)
}

// TestAdjustStock_Success_ExistingStock testa um ajuste de estoque bem-sucedido para um item existente.
func TestAdjustStock_Success_ExistingStock(t *testing.T) {
	mockRepo := new(MockStockRepository)
	mockLogger := logger.NewLogger("debug") // Usar um logger mock ou nulo em testes reais.

	svc := stockservice.NewService(mockRepo, mockLogger)

	// Dados de teste
	variantID := uuid.New().String()
	warehouseID := uuid.New().String()
	delta := 5

	// Mock do comportamento do repositório
	initialStockLevel := domain.StockLevel{
		ID:          uuid.New().String(),
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Quantity:    10,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	expectedUpdatedStockLevel := domain.StockLevel{
		ID:          initialStockLevel.ID,
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Quantity:    initialStockLevel.Quantity + delta,
		Version:     initialStockLevel.Version + 1,
		CreatedAt:   initialStockLevel.CreatedAt,
		UpdatedAt:   time.Now(), // UpdatedAt será diferente, apenas verificar que não é zero
	}

	mockRepo.On("UpdateStockLevel", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("domain.StockAdjustmentRequest")).
		Return(expectedUpdatedStockLevel, nil)

	// Executar o método do serviço
	adjustment := domain.StockAdjustmentRequest{
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Delta:       delta,
	}
	ctx := context.Background() // Usar context.Background() para o teste

	result, err := svc.AdjustStock(ctx, adjustment)

	// Verificar resultados
	assert.NoError(t, err)
	assert.Equal(t, expectedUpdatedStockLevel.Quantity, result.Quantity)
	assert.Equal(t, expectedUpdatedStockLevel.Version, result.Version)
	assert.NotZero(t, result.UpdatedAt) // Apenas verificar que foi atualizado
	mockRepo.AssertExpectations(t)
}

// TestAdjustStock_Success_NewStock testa um ajuste de estoque que resulta na criação de um novo registro.
func TestAdjustStock_Success_NewStock(t *testing.T) {
	mockRepo := new(MockStockRepository)
	mockLogger := logger.NewLogger("debug")

	svc := stockservice.NewService(mockRepo, mockLogger)

	variantID := uuid.New().String()
	warehouseID := uuid.New().String()
	delta := 15

	expectedNewStockLevel := domain.StockLevel{
		ID:          uuid.New().String(),
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Quantity:    delta,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockRepo.On("UpdateStockLevel", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("domain.StockAdjustmentRequest")).
		Return(expectedNewStockLevel, nil) // Repositório "cria" o novo estoque

	adjustment := domain.StockAdjustmentRequest{
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Delta:       delta,
	}
	ctx := context.Background()

	result, err := svc.AdjustStock(ctx, adjustment)

	assert.NoError(t, err)
	assert.Equal(t, expectedNewStockLevel.Quantity, result.Quantity)
	assert.Equal(t, expectedNewStockLevel.Version, result.Version)
	mockRepo.AssertExpectations(t)
}

// TestAdjustStock_Fail_NegativeResultingStock testa a prevenção de estoque negativo.
func TestAdjustStock_Fail_NegativeResultingStock(t *testing.T) {
	mockRepo := new(MockStockRepository)
	mockLogger := logger.NewLogger("debug")

	svc := stockservice.NewService(mockRepo, mockLogger)

	variantID := uuid.New().String()
	warehouseID := uuid.New().String()
	delta := -15

	// Simular que o repositório retorna um erro de validação de estoque negativo
	mockRepo.On("UpdateStockLevel", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("domain.StockAdjustmentRequest")).
		Return(domain.StockLevel{}, apperror.NewValidationError("Ajuste resultaria em quantidade de estoque negativa."))

	adjustment := domain.StockAdjustmentRequest{
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Delta:       delta,
	}
	ctx := context.Background()

	_, err := svc.AdjustStock(ctx, adjustment)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ValidationError{}, err)
	assert.Contains(t, err.Error(), "estoque negativa")
	mockRepo.AssertExpectations(t)
}

// TestAdjustStock_Fail_OCCConflict testa um conflito de concorrência otimista.
func TestAdjustStock_Fail_OCCConflict(t *testing.T) {
	mockRepo := new(MockStockRepository)
	mockLogger := logger.NewLogger("debug")

	svc := stockservice.NewService(mockRepo, mockLogger)

	variantID := uuid.New().String()
	warehouseID := uuid.New().String()
	delta := 1

	// Simular que o repositório retorna um erro de conflito de concorrência
	mockRepo.On("UpdateStockLevel", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("domain.StockAdjustmentRequest")).
		Return(domain.StockLevel{}, apperror.NewConflictError("O estoque foi modificado por outra operação. Tente novamente."))

	adjustment := domain.StockAdjustmentRequest{
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Delta:       delta,
	}
	ctx := context.Background()

	_, err := svc.AdjustStock(ctx, adjustment)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ConflictError{}, err)
	assert.Contains(t, err.Error(), "concorrência")
	mockRepo.AssertExpectations(t)
}

// TestAdjustStock_Fail_ZeroDelta testa o caso onde o delta é zero.
func TestAdjustStock_Fail_ZeroDelta(t *testing.T) {
	mockRepo := new(MockStockRepository)
	mockLogger := logger.NewLogger("debug")

	svc := stockservice.NewService(mockRepo, mockLogger)

	variantID := uuid.New().String()
	warehouseID := uuid.New().String()
	delta := 0 // Delta zero

	// O repositório não deve ser chamado
	mockRepo.AssertNotCalled(t, "UpdateStockLevel", mock.Anything, mock.Anything) // Corrigido para AssertNotCalled com args

	adjustment := domain.StockAdjustmentRequest{
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Delta:       delta,
	}
	ctx := context.Background()

	_, err := svc.AdjustStock(ctx, adjustment)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ValidationError{}, err)
	assert.Contains(t, err.Error(), "não pode ser zero")
	mockRepo.AssertExpectations(t)
}

// TestAdjustStock_Fail_InternalError testa um erro interno do repositório.
func TestAdjustStock_Fail_InternalError(t *testing.T) {
	mockRepo := new(MockStockRepository)
	mockLogger := logger.NewLogger("debug")

	svc := stockservice.NewService(mockRepo, mockLogger)

	variantID := uuid.New().String()
	warehouseID := uuid.New().String()
	delta := 1

	// Simular um erro genérico do repositório
	repoError := errors.New("falha de conexão com o DB")
	mockRepo.On("UpdateStockLevel", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("domain.StockAdjustmentRequest")).
		Return(domain.StockLevel{}, repoError)

	adjustment := domain.StockAdjustmentRequest{
		VariantID:   variantID,
		WarehouseID: warehouseID,
		Delta:       delta,
	}
	ctx := context.Background()

	_, err := svc.AdjustStock(ctx, adjustment)

	assert.Error(t, err)
	assert.IsType(t, &apperror.InternalError{}, err) // O serviço deve converter para InternalError
	assert.Contains(t, err.Error(), "Falha interna ao ajustar estoque.")
	mockRepo.AssertExpectations(t)
}
