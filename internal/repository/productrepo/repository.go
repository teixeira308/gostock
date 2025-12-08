package productrepo

import (
	"context" // Usamos o pacote context do Go
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gostock/internal/domain"
	"gostock/internal/errors"
	"gostock/internal/pkg/cache"
)

// ProductRepository implementa a interface domain.ProductRepository.
// Ela contém as conexões necessárias para acessar dados.
type ProductRepository struct {
	DB        *sql.DB      // Conexão principal com o banco de dados (PostgreSQL)
	Cache     cache.Client // Cliente para operações de cache (Redis)
	DBTimeout time.Duration
}

// NewProductRepository cria e retorna uma nova instância do Repositório.
// Aqui injetamos as dependências de Infraestrutura (DB e Cache).
func NewProductRepository(db *sql.DB, cacheClient cache.Client, dbTimeout time.Duration) *ProductRepository {
	return &ProductRepository{
		DB:        db,
		Cache:     cacheClient,
		DBTimeout: dbTimeout,
	}
}

// Save persiste um novo Produto e suas Variantes no banco de dados.
// (Implementa um dos métodos da interface domain.ProductRepository)
func (r *ProductRepository) Save(ctx context.Context, product domain.Product, variants []domain.Variant) (domain.Product, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	tx, err := r.DB.BeginTx(ctxTimeout, nil)
	if err != nil {
		return domain.Product{}, errors.NewDBError("failed to start tx", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	const productSQL = `INSERT INTO products (id, sku, name, description, price, is_active, created_at, updated_at)
                         VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

	createdAt := product.CreatedAt.Format(time.RFC3339Nano)
	updatedAt := product.UpdatedAt.Format(time.RFC3339Nano)

	_, err = tx.ExecContext(ctxTimeout, productSQL,
		product.ID,
		product.SKU,
		product.Name,
		product.Description,
		product.Price,
		product.IsActive,
		createdAt,
		updatedAt,
	)

	if err != nil {
		return domain.Product{}, errors.NewDBError("failed to insert product", err)
	}

	const variantSQL = `INSERT INTO variants(id, product_id, attribute, value, barcode, price_diff)
                        VALUES ($1,$2,$3,$4,$5,$6)`

	for _, v := range variants {
		_, err = tx.ExecContext(ctxTimeout, variantSQL,
			v.ID,
			v.ProductID,
			v.Attribute,
			v.Value,
			v.Barcode,
			v.PriceDiff,
		)
		if err != nil {
			return domain.Product{}, errors.NewDBError("failed to insert variants", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return domain.Product{}, errors.NewDBError("failed to commit tx", err)
	}

	return product, nil
}

// Outros métodos (FindByID, FindAll, Update, Delete) seriam implementados aqui.
// Define a chave de cache para produtos.
const productCacheKey = "product:%s"

// FindByID busca um produto pelo ID, utilizando a estratégia Cache-Aside.
// (Implementa um dos métodos da interface domain.ProductRepository)
func (r *ProductRepository) FindByID(ctx domain.Context, id string) (domain.Product, error) {

	// 1. Casting e Contexto
	ctxGo, cancel := context.WithTimeout(ctx.(context.Context),
		r.DBTimeout) // Assumindo que você adicionou DBTimeout no struct
	defer cancel()

	// Chave de Cache
	key := fmt.Sprintf(productCacheKey, id)
	var product domain.Product

	// --- 2. Estratégia Cache-Aside (READ) ---
	// (Módulo: Caching Strategies)

	// Tentar obter do Cache (Redis)
	cachedData, err := r.Cache.Get(ctxGo, key)
	if err == nil {
		// Cache HIT
		if json.Unmarshal([]byte(cachedData), &product) == nil {
			// Sucesso na desserialização, retorna o produto do cache
			return product, nil
		}
		// Se a desserialização falhar, logar e continuar para o DB
	} else if err != cache.ErrCacheMiss { // ErrCacheMiss indica que a chave não existe
		// Se houver um erro real de cache (ex: conexão perdida), logamos, mas continuamos.
		// log.Printf("Aviso: Falha ao ler do cache Redis: %v", err)
	}

	// --- 3. Busca no Banco de Dados (PostgreSQL) ---

	// Query SQL
	productSQL := `
		SELECT id, sku, name, description, price, is_active, created_at, updated_at
		FROM products 
		WHERE id = $1`

	row := r.DB.QueryRowContext(ctxGo, productSQL, id)

	// Mapeamento dos campos do DB para a struct domain.Product
	err = row.Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	// 4. Tratamento do Erro de Busca (Crucial para o 404)
	if err == sql.ErrNoRows {
		// Se não houver linhas, retornamos um erro de Domínio NotFoundError
		// O Serviço receberá isso e o Handler o mapeará para 404.
		return domain.Product{}, errors.NewNotFoundError(fmt.Sprintf("Produto com ID %s não existe na base de dados.", id))
	}
	if err != nil {
		// Qualquer outro erro é um InternalError (DB falhou, timeout, etc.)
		return domain.Product{}, errors.NewDBError("Falha ao buscar produto no DB", err)
	}

	// --- 5. Estratégia Cache-Aside (WRITE) ---
	// Se encontrado no DB, populamos o cache para futuras requisições.
	productJSON, marshalErr := json.Marshal(product)
	if marshalErr == nil {
		// Define o produto no cache com uma expiração (TTL)
		// TTL de 5 minutos, por exemplo (deve vir do config)
		r.Cache.Set(ctxGo, key, productJSON, 5*time.Minute)
	} else {
		// log.Printf("Aviso: Falha ao serializar produto para cache: %v", marshalErr)
	}

	return product, nil
}
