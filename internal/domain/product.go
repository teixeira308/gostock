package domain

import (
	"time"
)

// Product representa o item principal do catálogo (a Entidade).
// Contém informações essenciais e de metadados.
type Product struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"` // Stock Keeping Unit (código único de produto)
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relações (serão gerenciadas por outras entidades/serviços)
	// Variantes []Variant // Exemplo: Lista de tamanhos/cores
	Variants []Variant `json:"variants"`
}

// Variant representa as variações de um Produto (e.g., cor, tamanho).
// O controle de estoque (StockLevels) será feito a nível de Variant.
type Variant struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Attribute string  `json:"attribute"` // Ex: "Cor"
	Value     string  `json:"value"`     // Ex: "Vermelho"
	Barcode   string  `json:"barcode"`
	PriceDiff float64 `json:"price_diff"` // Ajuste de preço para esta variante
}

// --- Interfaces de Contrato (O CORAÇÃO DA ARQUITETURA LIMPA) ---

// ProductService é a interface que a camada de Serviço (Business Logic) DEVE implementar.
// Ela define o que o Handler (Camada API) pode pedir para a camada de Serviço fazer.
type ProductService interface {
	CreateProduct(ctx Context, product Product) (Product, error)
	GetProductByID(ctx Context, id string) (Product, error)
	ListProducts(ctx Context, filter ProductFilter) ([]Product, error)
	UpdateProduct(ctx Context, product Product) error
	DeleteProduct(ctx Context, id string) error
}

// ProductRepository é a interface que a camada de Repositório (Data Access) DEVE implementar.
// Ela define o que a camada de Serviço (Service) pode pedir para a camada de Persistência (DB/Cache) fazer.
type ProductRepository interface {
	Save(ctx Context, product Product) (Product, error)
	FindByID(ctx Context, id string) (Product, error)
	FindAll(ctx Context, filter ProductFilter) ([]Product, error)
	Update(ctx Context, product Product) error
	Delete(ctx Context, id string) error
}

// --- Estruturas Auxiliares (Filtros e Contexto) ---

// ProductFilter define os parâmetros de busca e paginação (RF 1.3).
type ProductFilter struct {
	Page       int
	Limit      int
	Name       string
	SKU        string
	ActiveOnly bool
}

// Context é uma interface que encapsula o Go context.Context.
// É usado para propagar o timeout e sinais de cancelamento pelas camadas.
// Isso evita a dependência direta do pacote "context".
type Context interface{}
