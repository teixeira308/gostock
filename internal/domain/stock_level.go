package domain

import "time"

// StockLevel representa o nível de estoque de uma variante específica em um armazém.
// Inclui uma coluna 'version' para controle de concorrência otimista.
type StockLevel struct {
	ID          string    `json:"id"`
	VariantID   string    `json:"variant_id"`
	WarehouseID string    `json:"warehouse_id"`
	Quantity    int       `json:"quantity"`
	Version     int       `json:"version"` // Para Controle de Concorrência Otimista (OCC)
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StockAdjustmentRequest é o payload esperado para a requisição de ajuste de estoque.
type StockAdjustmentRequest struct {
	VariantID   string `json:"variant_id" validate:"required,uuid"`
	WarehouseID string `json:"warehouse_id" validate:"required,uuid"`
	Delta       int    `json:"delta" validate:"required,numeric"` // Quantidade a ser adicionada/removida
}
