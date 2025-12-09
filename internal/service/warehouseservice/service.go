package warehouseservice

import (
	"context"

	"strings"

	"github.com/google/uuid"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
)

// WarehouseRepository define o contrato que o Serviço de Armazéns espera da camada de Persistência.
type WarehouseRepository interface {
	CreateWarehouse(ctx context.Context, warehouse domain.Warehouse) (domain.Warehouse, error)
	GetWarehouseByID(ctx context.Context, id string) (domain.Warehouse, error)
	GetAllWarehouses(ctx context.Context) ([]domain.Warehouse, error)
	UpdateWarehouse(ctx context.Context, warehouse domain.Warehouse) (domain.Warehouse, error)
	DeleteWarehouse(ctx context.Context, id string) error
}

// Service é a estrutura que implementa a interface domain.WarehouseService (a ser definida).
type Service struct {
	repo   WarehouseRepository
	logger logger.Logger
}

// NewService cria e retorna uma nova instância do Serviço de Armazéns.
func NewService(repo WarehouseRepository, logger logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// CreateWarehouse cria um novo armazém após validações de negócio.
func (s *Service) CreateWarehouse(ctx domain.Context, warehouse domain.Warehouse) (domain.Warehouse, error) {
	s.logger.Debug("Iniciando criação de armazém no serviço.", map[string]interface{}{"name": warehouse.Name})

	if err := s.validateWarehouseName(warehouse.Name); err != nil {
		s.logger.Warn("Falha na validação do nome do armazém.", map[string]interface{}{"name": warehouse.Name, "error": err.Error()})
		return domain.Warehouse{}, err
	}

	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
		s.logger.Warn("Contexto de domínio inválido, usando context.Background() para CreateWarehouse", nil)
	}

	createdWarehouse, err := s.repo.CreateWarehouse(ctxGo, warehouse)
	if err != nil {
		s.logger.Error("Falha ao criar armazém no repositório.", err)
		return domain.Warehouse{}, apperror.NewInternalError("Falha interna ao criar armazém.", err)
	}

	s.logger.Info("Armazém criado com sucesso.", map[string]interface{}{"id": createdWarehouse.ID, "name": createdWarehouse.Name})
	return createdWarehouse, nil
}

// GetWarehouseByID busca um armazém pelo ID após validações de formato.
func (s *Service) GetWarehouseByID(ctx domain.Context, id string) (domain.Warehouse, error) {
	s.logger.Debug("Iniciando busca de armazém por ID no serviço.", map[string]interface{}{"id": id})

	if _, err := uuid.Parse(id); err != nil {
		s.logger.Warn("ID de armazém inválido fornecido.", map[string]interface{}{"id": id, "error": err.Error()})
		return domain.Warehouse{}, apperror.NewValidationError("O ID do armazém deve ser um UUID válido.")
	}

	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
		s.logger.Warn("Contexto de domínio inválido, usando context.Background() para GetWarehouseByID", nil)
	}

	warehouse, err := s.repo.GetWarehouseByID(ctxGo, id)
	if err != nil {
		s.logger.Error("Falha ao buscar armazém no repositório.", err)
		return domain.Warehouse{}, err // Erros do repositório já são NotFoundError ou DBError
	}

	s.logger.Info("Armazém encontrado.", map[string]interface{}{"id": warehouse.ID, "name": warehouse.Name})
	return warehouse, nil
}

// GetAllWarehouses busca todos os armazéns.
func (s *Service) GetAllWarehouses(ctx domain.Context) ([]domain.Warehouse, error) {
	s.logger.Debug("Iniciando busca de todos os armazéns no serviço.", nil)

	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
		s.logger.Warn("Contexto de domínio inválido, usando context.Background() para GetAllWarehouses", nil)
	}

	warehouses, err := s.repo.GetAllWarehouses(ctxGo)
	if err != nil {
		s.logger.Error("Falha ao buscar todos os armazéns no repositório.", err)
		return nil, apperror.NewInternalError("Falha interna ao buscar armazéns.", err)
	}

	s.logger.Info("Todos os armazéns encontrados com sucesso.", map[string]interface{}{"count": len(warehouses)})
	return warehouses, nil
}

// UpdateWarehouse atualiza um armazém existente.
func (s *Service) UpdateWarehouse(ctx domain.Context, warehouse domain.Warehouse) (domain.Warehouse, error) {
	s.logger.Debug("Iniciando atualização de armazém no serviço.", map[string]interface{}{"id": warehouse.ID, "name": warehouse.Name})

	if _, err := uuid.Parse(warehouse.ID); err != nil {
		s.logger.Warn("ID de armazém inválido fornecido para atualização.", map[string]interface{}{"id": warehouse.ID, "error": err.Error()})
		return domain.Warehouse{}, apperror.NewValidationError("O ID do armazém deve ser um UUID válido.")
	}

	if err := s.validateWarehouseName(warehouse.Name); err != nil {
		s.logger.Warn("Falha na validação do nome do armazém para atualização.", map[string]interface{}{"name": warehouse.Name, "error": err.Error()})
		return domain.Warehouse{}, err
	}

	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
		s.logger.Warn("Contexto de domínio inválido, usando context.Background() para UpdateWarehouse", nil)
	}

	updatedWarehouse, err := s.repo.UpdateWarehouse(ctxGo, warehouse)
	if err != nil {
		s.logger.Error("Falha ao atualizar armazém no repositório.", err)
		return domain.Warehouse{}, err // Erros do repositório já são NotFoundError ou DBError
	}

	s.logger.Info("Armazém atualizado com sucesso.", map[string]interface{}{"id": updatedWarehouse.ID, "name": updatedWarehouse.Name})
	return updatedWarehouse, nil
}

// DeleteWarehouse remove um armazém.
func (s *Service) DeleteWarehouse(ctx domain.Context, id string) error {
	s.logger.Debug("Iniciando exclusão de armazém no serviço.", map[string]interface{}{"id": id})

	if _, err := uuid.Parse(id); err != nil {
		s.logger.Warn("ID de armazém inválido fornecido para exclusão.", map[string]interface{}{"id": id, "error": err.Error()})
		return apperror.NewValidationError("O ID do armazém deve ser um UUID válido.")
	}

	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
		s.logger.Warn("Contexto de domínio inválido, usando context.Background() para DeleteWarehouse", nil)
	}

	err := s.repo.DeleteWarehouse(ctxGo, id)
	if err != nil {
		s.logger.Error("Falha ao deletar armazém no repositório.", err)
		return err // Erros do repositório já são NotFoundError ou DBError
	}

	s.logger.Info("Armazém deletado com sucesso.", map[string]interface{}{"id": id})
	return nil
}

// validateWarehouseName é uma função auxiliar para validar o nome do armazém.
func (s *Service) validateWarehouseName(name string) error {
	if strings.TrimSpace(name) == "" {
		return apperror.NewValidationError("O nome do armazém não pode ser vazio.")
	}
	if len(name) < 3 || len(name) > 100 {
		return apperror.NewValidationError("O nome do armazém deve ter entre 3 e 100 caracteres.")
	}
	// Poderia adicionar mais validações, como caracteres permitidos, unicidade, etc.
	return nil
}
