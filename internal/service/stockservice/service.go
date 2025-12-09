package stockservice

import (
	"context"
	"fmt"

	"errors"
	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
)

// StockRepository define o contrato que o Serviço de Estoque espera da camada de Persistência.
type StockRepository interface {
	GetStockLevel(ctx context.Context, variantID, warehouseID string) (domain.StockLevel, error)
	UpdateStockLevel(ctx context.Context, adjustment domain.StockAdjustmentRequest) (domain.StockLevel, error)
	// Add more methods for Warehouse CRUD if needed later
}

// Service é a estrutura que implementa a interface domain.StockService (a ser definida).
type Service struct {
	repo   StockRepository
	logger logger.Logger
}

// NewService cria e retorna uma nova instância do Serviço de Estoque.
func NewService(repo StockRepository, logger logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// AdjustStock aplica um ajuste ao nível de estoque de um produto em um armazém.
func (s *Service) AdjustStock(ctx domain.Context, adjustment domain.StockAdjustmentRequest) (domain.StockLevel, error) {
	s.logger.Debug("Iniciando ajuste de estoque no serviço.", map[string]interface{}{
		"variant_id":   adjustment.VariantID,
		"warehouse_id": adjustment.WarehouseID,
		"delta":        adjustment.Delta,
	})

	// Basic validation (more comprehensive validation can be in the handler or domain)
	if adjustment.Delta == 0 {
		return domain.StockLevel{}, apperror.NewValidationError("O ajuste de estoque (delta) não pode ser zero.")
	}

	// Casting e Configuração do Contexto (Converte domain.Context para context.Context)
	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
		s.logger.Warn("Contexto de domínio inválido, usando context.Background() para AdjustStock", nil)
	}

	stockLevel, err := s.repo.UpdateStockLevel(ctxGo, adjustment)
	if err != nil {
		s.logger.Error("Falha ao ajustar estoque no repositório.", err)
		// Translate repository errors to service/domain errors if necessary
		var conflictErr *apperror.ConflictError
		if errors.As(err, &conflictErr) {
			return domain.StockLevel{}, apperror.NewConflictError(fmt.Sprintf("Falha de concorrência: %s", conflictErr.Error()))
		}
		var validationErr *apperror.ValidationError
		if errors.As(err, &validationErr) {
			return domain.StockLevel{}, apperror.NewValidationError(fmt.Sprintf("Validação do estoque: %s", validationErr.Error()))
		}
		return domain.StockLevel{}, apperror.NewInternalError("Falha interna ao ajustar estoque.", err)
	}

	s.logger.Info("Estoque ajustado com sucesso.", map[string]interface{}{
		"variant_id":   stockLevel.VariantID,
		"warehouse_id": stockLevel.WarehouseID,
		"new_quantity": stockLevel.Quantity,
		"new_version":  stockLevel.Version,
	})
	return stockLevel, nil
}
