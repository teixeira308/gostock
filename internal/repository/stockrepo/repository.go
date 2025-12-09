package stockrepo

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

// StockRepository implementa a interface domain.StockRepository (a ser definida no domínio, ou aqui se for um subdomínio).
type StockRepository struct {
	DB        *sql.DB
	DBTimeout time.Duration
	logger    logger.Logger
}

// NewStockRepository cria e retorna uma nova instância do Repositório de Estoque.
func NewStockRepository(db *sql.DB, dbTimeout time.Duration, logger logger.Logger) *StockRepository {
	return &StockRepository{
		DB:        db,
		DBTimeout: dbTimeout,
		logger:    logger,
	}
}

// GetStockLevel busca o nível de estoque para uma variante em um armazém.
func (r *StockRepository) GetStockLevel(ctx context.Context, variantID, warehouseID string) (domain.StockLevel, error) {
	r.logger.Debug("Buscando nível de estoque no repositório.", map[string]interface{}{"variant_id": variantID, "warehouse_id": warehouseID})

	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	query := `
        SELECT id, variant_id, warehouse_id, quantity, version, created_at, updated_at
        FROM stock_levels
        WHERE variant_id = $1 AND warehouse_id = $2`

	var sl domain.StockLevel
	err := r.DB.QueryRowContext(ctxTimeout, query, variantID, warehouseID).Scan(
		&sl.ID, &sl.VariantID, &sl.WarehouseID, &sl.Quantity, &sl.Version, &sl.CreatedAt, &sl.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		r.logger.Info("Nível de estoque não encontrado.", map[string]interface{}{"variant_id": variantID, "warehouse_id": warehouseID})
		return domain.StockLevel{}, errors.NewNotFoundError(fmt.Sprintf("Estoque para variante %s no armazém %s não encontrado.", variantID, warehouseID))
	}
	if err != nil {
		r.logger.Error("Falha ao buscar nível de estoque no DB.", err)
		return domain.StockLevel{}, errors.NewDBError("Falha ao buscar nível de estoque", err)
	}

	r.logger.Debug("Nível de estoque encontrado.", map[string]interface{}{"variant_id": variantID, "warehouse_id": warehouseID, "quantity": sl.Quantity, "version": sl.Version})
	return sl, nil
}

// UpdateStockLevel aplica um ajuste ao estoque, utilizando transação e controle de concorrência otimista (OCC).
func (r *StockRepository) UpdateStockLevel(ctx context.Context, adjustment domain.StockAdjustmentRequest) (domain.StockLevel, error) {
	r.logger.Debug("Iniciando atualização de estoque no repositório.", map[string]interface{}{
		"variant_id": adjustment.VariantID,
		"warehouse_id": adjustment.WarehouseID,
		"delta": adjustment.Delta,
	})

	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	tx, err := r.DB.BeginTx(ctxTimeout, nil)
	if err != nil {
		r.logger.Error("Falha ao iniciar transação para atualização de estoque.", err)
		return domain.StockLevel{}, errors.NewDBError("Falha ao iniciar transação", err)
	}
	defer tx.Rollback() // Rollback em caso de erro

	// 1. Obter o nível de estoque atual (com FOR UPDATE para bloquear a linha na transação)
	//    É crucial selecionar a 'version' atual aqui.
	var currentStock domain.StockLevel
	querySelect := `
        SELECT id, variant_id, warehouse_id, quantity, version, created_at, updated_at
        FROM stock_levels
        WHERE variant_id = $1 AND warehouse_id = $2 FOR UPDATE`

	err = tx.QueryRowContext(ctxTimeout, querySelect, adjustment.VariantID, adjustment.WarehouseID).Scan(
		&currentStock.ID, &currentStock.VariantID, &currentStock.WarehouseID, &currentStock.Quantity,
		&currentStock.Version, &currentStock.CreatedAt, &currentStock.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Se não houver registro, é uma inserção inicial
		newID := uuid.New().String()
		newQuantity := adjustment.Delta
		if newQuantity < 0 {
			r.logger.Warn("Tentativa de criar estoque com quantidade negativa.", map[string]interface{}{"variant_id": adjustment.VariantID, "warehouse_id": adjustment.WarehouseID, "delta": adjustment.Delta})
			return domain.StockLevel{}, errors.NewValidationError("Não é possível criar estoque com quantidade negativa.")
		}

		queryInsert := `
            INSERT INTO stock_levels (id, variant_id, warehouse_id, quantity, version, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            RETURNING id, variant_id, warehouse_id, quantity, version, created_at, updated_at`

		var newSl domain.StockLevel
		err = tx.QueryRowContext(ctxTimeout, queryInsert,
			newID, adjustment.VariantID, adjustment.WarehouseID, newQuantity, 1, time.Now(), time.Now(),
		).Scan(
			&newSl.ID, &newSl.VariantID, &newSl.WarehouseID, &newSl.Quantity,
			&newSl.Version, &newSl.CreatedAt, &newSl.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Falha ao inserir novo nível de estoque.", err)
			return domain.StockLevel{}, errors.NewDBError("Falha ao inserir novo nível de estoque", err)
		}

		if commitErr := tx.Commit(); commitErr != nil {
			r.logger.Error("Falha ao commitar transação de inserção de estoque.", commitErr)
			return domain.StockLevel{}, errors.NewDBError("Falha ao commitar transação", commitErr)
		}
		r.logger.Info("Novo nível de estoque criado com sucesso.", map[string]interface{}{"variant_id": adjustment.VariantID, "warehouse_id": adjustment.WarehouseID, "quantity": newSl.Quantity})
		return newSl, nil

	} else if err != nil {
		r.logger.Error("Falha ao selecionar nível de estoque para atualização.", err)
		return domain.StockLevel{}, errors.NewDBError("Falha ao buscar estoque para atualização", err)
	}

	// 2. Aplicar o ajuste e verificar se a quantidade resultará em negativo
	newQuantity := currentStock.Quantity + adjustment.Delta
	if newQuantity < 0 {
		r.logger.Warn("Tentativa de ajustar estoque para quantidade negativa.", map[string]interface{}{"variant_id": adjustment.VariantID, "warehouse_id": adjustment.WarehouseID, "current_quantity": currentStock.Quantity, "delta": adjustment.Delta})
		return domain.StockLevel{}, errors.NewValidationError("Ajuste resultaria em quantidade de estoque negativa.")
	}

	// 3. Atualizar o nível de estoque com OCC
	queryUpdate := `
        UPDATE stock_levels
        SET quantity = $1, version = $2, updated_at = $3
        WHERE variant_id = $4 AND warehouse_id = $5 AND version = $6`

	result, err := tx.ExecContext(ctxTimeout, queryUpdate,
		newQuantity,
		currentStock.Version + 1, // Incrementa a versão
		time.Now(),
		adjustment.VariantID,
		adjustment.WarehouseID,
		currentStock.Version, // Checa a versão antiga para OCC
	)
	if err != nil {
		r.logger.Error("Falha ao atualizar nível de estoque.", err)
		return domain.StockLevel{}, errors.NewDBError("Falha ao atualizar estoque", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Falha ao verificar linhas afetadas após atualização de estoque.", err)
		return domain.StockLevel{}, errors.NewDBError("Falha ao verificar linhas afetadas", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("Falha no controle de concorrência otimista (OCC). Versão do registro desatualizada.", map[string]interface{}{
			"variant_id": adjustment.VariantID,
			"warehouse_id": adjustment.WarehouseID,
			"expected_version": currentStock.Version,
		})
		// Erro de concorrência otimista: o registro foi modificado por outra transação.
		return domain.StockLevel{}, errors.NewConflictError("O estoque foi modificado por outra operação. Tente novamente.")
	}

	// 4. Commitar a transação
	if commitErr := tx.Commit(); commitErr != nil {
		r.logger.Error("Falha ao commitar transação de atualização de estoque.", commitErr)
		return domain.StockLevel{}, errors.NewDBError("Falha ao commitar transação", commitErr)
	}

	currentStock.Quantity = newQuantity
	currentStock.Version++
	currentStock.UpdatedAt = time.Now() // Atualiza o campo UpdatedAt para refletir a mudança
	r.logger.Info("Nível de estoque atualizado com sucesso.", map[string]interface{}{
		"variant_id": adjustment.VariantID,
		"warehouse_id": adjustment.WarehouseID,
		"new_quantity": newQuantity,
		"new_version": currentStock.Version,
	})
	return currentStock, nil
}
