package warehouserepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gostock/internal/domain"
	"gostock/internal/errors"
	"gostock/internal/pkg/logger"
)

// WarehouseRepository implementa a interface para operações CRUD de armazéns.
type WarehouseRepository struct {
	DB        *sql.DB
	DBTimeout time.Duration
	logger    logger.Logger
}

// NewWarehouseRepository cria e retorna uma nova instância do Repositório de Armazéns.
func NewWarehouseRepository(db *sql.DB, dbTimeout time.Duration, logger logger.Logger) *WarehouseRepository {
	return &WarehouseRepository{
		DB:        db,
		DBTimeout: dbTimeout,
		logger:    logger,
	}
}

// CreateWarehouse insere um novo armazém no banco de dados.
func (r *WarehouseRepository) CreateWarehouse(ctx context.Context, warehouse domain.Warehouse) (domain.Warehouse, error) {
	r.logger.Debug("Iniciando CreateWarehouse no repositório.", map[string]interface{}{"name": warehouse.Name})

	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	if warehouse.ID == "" {
		warehouse.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	warehouse.CreatedAt = now
	warehouse.UpdatedAt = now

	query := `
        INSERT INTO warehouses (id, name, created_at, updated_at)
        VALUES ($1, $2, $3, $4)
        RETURNING id, name, created_at, updated_at`

	err := r.DB.QueryRowContext(ctxTimeout, query,
		warehouse.ID, warehouse.Name, warehouse.CreatedAt, warehouse.UpdatedAt,
	).Scan(
		&warehouse.ID, &warehouse.Name, &warehouse.CreatedAt, &warehouse.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("Falha ao inserir armazém no DB.", err)
		return domain.Warehouse{}, errors.NewDBError("Falha ao criar armazém", err)
	}

	r.logger.Info("Armazém criado com sucesso.", map[string]interface{}{"id": warehouse.ID, "name": warehouse.Name})
	return warehouse, nil
}

// GetWarehouseByID busca um armazém pelo ID.
func (r *WarehouseRepository) GetWarehouseByID(ctx context.Context, id string) (domain.Warehouse, error) {
	r.logger.Debug("Iniciando GetWarehouseByID no repositório.", map[string]interface{}{"id": id})

	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	query := `
        SELECT id, name, created_at, updated_at
        FROM warehouses
        WHERE id = $1`

	var warehouse domain.Warehouse
	err := r.DB.QueryRowContext(ctxTimeout, query, id).Scan(
		&warehouse.ID, &warehouse.Name, &warehouse.CreatedAt, &warehouse.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		r.logger.Info("Armazém não encontrado.", map[string]interface{}{"id": id})
		return domain.Warehouse{}, errors.NewNotFoundError(fmt.Sprintf("Armazém com ID %s não encontrado.", id))
	}
	if err != nil {
		r.logger.Error("Falha ao buscar armazém no DB.", err)
		return domain.Warehouse{}, errors.NewDBError("Falha ao buscar armazém", err)
	}

	r.logger.Info("Armazém encontrado.", map[string]interface{}{"id": id, "name": warehouse.Name})
	return warehouse, nil
}

// GetAllWarehouses busca todos os armazéns.
func (r *WarehouseRepository) GetAllWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	r.logger.Debug("Iniciando GetAllWarehouses no repositório.", nil)

	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	query := `
        SELECT id, name, created_at, updated_at
        FROM warehouses
        ORDER BY name`

	rows, err := r.DB.QueryContext(ctxTimeout, query)
	if err != nil {
		r.logger.Error("Falha ao executar GetAllWarehouses query.", err)
		return nil, errors.NewDBError("Falha ao buscar todos os armazéns", err)
	}
	defer rows.Close()

	var warehouses []domain.Warehouse
	for rows.Next() {
		var warehouse domain.Warehouse
		err := rows.Scan(
			&warehouse.ID, &warehouse.Name, &warehouse.CreatedAt, &warehouse.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Falha ao mapear armazém na iteração de GetAllWarehouses.", err)
			return nil, errors.NewDBError("Falha ao mapear armazéns do DB", err)
		}
		warehouses = append(warehouses, warehouse)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Erro após iteração das linhas de armazéns.", err)
		return nil, errors.NewDBError("Erro após iteração de armazéns", err)
	}

	r.logger.Info("GetAllWarehouses concluído com sucesso.", map[string]interface{}{"total_warehouses": len(warehouses)})
	return warehouses, nil
}

// UpdateWarehouse atualiza um armazém existente.
func (r *WarehouseRepository) UpdateWarehouse(ctx context.Context, warehouse domain.Warehouse) (domain.Warehouse, error) {
	r.logger.Debug("Iniciando UpdateWarehouse no repositório.", map[string]interface{}{"id": warehouse.ID, "name": warehouse.Name})

	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	warehouse.UpdatedAt = time.Now().UTC()

	query := `
        UPDATE warehouses
        SET name = $1, updated_at = $2
        WHERE id = $3
        RETURNING id, name, created_at, updated_at`

	err := r.DB.QueryRowContext(ctxTimeout, query,
		warehouse.Name, warehouse.UpdatedAt, warehouse.ID,
	).Scan(
		&warehouse.ID, &warehouse.Name, &warehouse.CreatedAt, &warehouse.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		r.logger.Info("Armazém não encontrado para atualização.", map[string]interface{}{"id": warehouse.ID})
		return domain.Warehouse{}, errors.NewNotFoundError(fmt.Sprintf("Armazém com ID %s não encontrado para atualização.", warehouse.ID))
	}
	if err != nil {
		r.logger.Error("Falha ao atualizar armazém no DB.", err)
		return domain.Warehouse{}, errors.NewDBError("Falha ao atualizar armazém", err)
	}

	r.logger.Info("Armazém atualizado com sucesso.", map[string]interface{}{"id": warehouse.ID, "name": warehouse.Name})
	return warehouse, nil
}

// DeleteWarehouse remove um armazém pelo ID.
func (r *WarehouseRepository) DeleteWarehouse(ctx context.Context, id string) error {
	r.logger.Debug("Iniciando DeleteWarehouse no repositório.", map[string]interface{}{"id": id})

	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	query := `
        DELETE FROM warehouses
        WHERE id = $1`

	result, err := r.DB.ExecContext(ctxTimeout, query, id)
	if err != nil {
		r.logger.Error("Falha ao deletar armazém do DB.", err)
		return errors.NewDBError("Falha ao deletar armazém", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Falha ao verificar linhas afetadas após DeleteWarehouse.", err)
		return errors.NewDBError("Falha ao verificar linhas afetadas", err)
	}

	if rowsAffected == 0 {
		r.logger.Info("Armazém não encontrado para exclusão.", map[string]interface{}{"id": id})
		return errors.NewNotFoundError(fmt.Sprintf("Armazém com ID %s não encontrado para exclusão.", id))
	}

	r.logger.Info("Armazém deletado com sucesso.", map[string]interface{}{"id": id})
	return nil
}
