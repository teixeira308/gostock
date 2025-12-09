package productrepo

import (
	"context" // Usamos o pacote context do Go
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gostock/internal/domain"
	"gostock/internal/errors"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/cache"
	"gostock/internal/pkg/logger"
)

// ProductRepository implementa a interface domain.ProductRepository.
// Ela contém as conexões necessárias para acessar dados.
type ProductRepository struct {
	DB        *sql.DB      // Conexão principal com o banco de dados (PostgreSQL)
	Cache     cache.Client // Cliente para operações de cache (Redis)
	DBTimeout time.Duration
	logger    logger.Logger
}

// NewProductRepository cria e retorna uma nova instância do Repositório.
// Aqui injetamos as dependências de Infraestrutura (DB e Cache).
func NewProductRepository(db *sql.DB, cacheClient cache.Client, dbTimeout time.Duration, logger logger.Logger) *ProductRepository {
	return &ProductRepository{
		DB:        db,
		Cache:     cacheClient,
		DBTimeout: dbTimeout,
		logger:    logger,
	}
}

// Save persiste um novo Produto e suas Variantes no banco de dados.
// (Implementa um dos métodos da interface domain.ProductRepository)
func (r *ProductRepository) Save(ctx context.Context, product domain.Product) (domain.Product, error) {
	r.logger.Debug("Iniciando Save de produto no repositório.", map[string]interface{}{"sku": product.SKU})
	ctxTimeout, cancel := context.WithTimeout(ctx, r.DBTimeout)
	defer cancel()

	tx, err := r.DB.BeginTx(ctxTimeout, nil)
	if err != nil {
		r.logger.Error("Falha ao iniciar transação para Save de produto.", err)
		return domain.Product{}, errors.NewDBError("failed to start tx", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			r.logger.Warn("Transação de Save de produto desfeita devido a erro.", nil)
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
		r.logger.Error("Falha ao inserir produto no DB.", err)
		return domain.Product{}, errors.NewDBError("failed to insert product", err)
	}
	r.logger.Debug("Produto inserido no DB.", map[string]interface{}{"product_id": product.ID, "sku": product.SKU})

	const variantSQL = `INSERT INTO variants(id, product_id, attribute, value, barcode, price_diff)
                        VALUES ($1,$2,$3,$4,$5,$6)`

	for _, v := range product.Variants {
		_, err = tx.ExecContext(ctxTimeout, variantSQL,
			v.ID,
			v.ProductID,
			v.Attribute,
			v.Value,
			v.Barcode,
			v.PriceDiff,
		)
		if err != nil {
			r.logger.Error("Falha ao inserir variante no DB.", err)
			return domain.Product{}, errors.NewDBError("failed to insert variants", err)
		}
		r.logger.Debug("Variante inserida no DB.", map[string]interface{}{"variant_id": v.ID, "product_id": v.ProductID})
	}

	if err = tx.Commit(); err != nil {
		r.logger.Error("Falha ao commitar transação para Save de produto.", err)
		return domain.Product{}, errors.NewDBError("failed to commit tx", err)
	}

	r.logger.Info("Produto e variantes salvos com sucesso no repositório.", map[string]interface{}{"product_id": product.ID, "sku": product.SKU})
	return product, nil
}

// Outros métodos (FindByID, FindAll, Update, Delete) seriam implementados aqui.
// Define a chave de cache para produtos.
const productCacheKey = "product:%s"

// FindByID busca um produto pelo ID, utilizando a estratégia Cache-Aside.
// (Implementa um dos métodos da interface domain.ProductRepository)
func (r *ProductRepository) FindByID(ctx domain.Context, id string) (domain.Product, error) {
	r.logger.Debug("Iniciando FindByID de produto no repositório.", map[string]interface{}{"product_id_attempt": id})

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
			r.logger.Info("Produto encontrado no cache.", map[string]interface{}{"product_id": id})
			// Sucesso na desserialização, retorna o produto do cache
			return product, nil
		}
		r.logger.Error("Falha ao desserializar produto do cache.", err)
		// Se a desserialização falhar, logar e continuar para o DB
	} else if err != cache.ErrCacheMiss { // ErrCacheMiss indica que a chave não existe
		// Se houver um erro real de cache (ex: conexão perdida), logamos, mas continuamos.
		r.logger.Warn("Erro ao ler do cache Redis (não é um cache miss).", map[string]interface{}{"error": err.Error()})
	} else {
		r.logger.Debug("Cache miss para produto.", map[string]interface{}{"product_id": id})
	}

	// --- 3. Busca no Banco de Dados (PostgreSQL) ---
	r.logger.Debug("Buscando produto no DB.", map[string]interface{}{"product_id": id})
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
		r.logger.Info("Produto não encontrado no DB.", map[string]interface{}{"product_id": id})
		// Se não houver linhas, retornamos um erro de Domínio NotFoundError
		// O Serviço receberá isso e o Handler o mapeará para 404.
		return domain.Product{}, errors.NewNotFoundError(fmt.Sprintf("Produto com ID %s não existe na base de dados.", id))
	}
	if err != nil {
		r.logger.Error("Falha ao buscar produto no DB.", err)
		// Qualquer outro erro é um InternalError (DB falhou, timeout, etc.)
		return domain.Product{}, errors.NewDBError("Falha ao buscar produto no DB", err)
	}
	r.logger.Debug("Produto encontrado no DB.", map[string]interface{}{"product_id": product.ID, "sku": product.SKU})

	// --- 5. Estratégia Cache-Aside (WRITE) ---
	// Se encontrado no DB, populamos o cache para futuras requisições.
	productJSON, marshalErr := json.Marshal(product)
	if marshalErr == nil {
		r.logger.Debug("Salvando produto no cache.", map[string]interface{}{"product_id": product.ID})
		// Define o produto no cache com uma expiração (TTL)
		// TTL de 5 minutos, por exemplo (deve vir do config)
		r.Cache.Set(ctxGo, key, productJSON, 5*time.Minute)
	} else {
		r.logger.Error("Falha ao serializar produto para cache.", marshalErr)
	}

	// 🚨 NOVO: Buscar e anexar variações
	variants, err := r.FindVariantsByProductID(ctx, product.ID)
	if err != nil {
		r.logger.Warn("Falha ao buscar variações para o produto (pode ser aceitável).", map[string]interface{}{"product_id": product.ID, "error": err.Error()})
		// Se a busca de variações falhar, logamos mas podemos optar por retornar o produto sem elas
		// Ou retornar o erro, dependendo da criticidade. Retornar o erro é mais seguro.
		return domain.Product{}, err
	}
	product.Variants = variants
	r.logger.Info("Produto e suas variantes recuperados com sucesso do repositório.", map[string]interface{}{"product_id": product.ID, "sku": product.SKU})
	return product, nil
}

// FindVariantsByProductID busca todas as variações para um dado ID de produto.
func (r *ProductRepository) FindVariantsByProductID(ctx domain.Context, productID string) ([]domain.Variant, error) {
	r.logger.Debug("Iniciando busca de variantes por ProductID.", map[string]interface{}{"product_id": productID})
	ctxTimeout, cancel := context.WithTimeout(ctx.(context.Context), r.DBTimeout)
	defer cancel()

	query := `
        SELECT id, product_id, attribute, value, barcode, price_diff
        FROM variants
        WHERE product_id = $1
    `

	rows, err := r.DB.QueryContext(ctxTimeout, query, productID)
	if err != nil {
		r.logger.Error("Falha ao executar QueryContext para buscar variantes.", err)
		return nil, apperror.NewDBError("Falha ao buscar variações do produto (DB)", err)
	}
	defer rows.Close()

	variants := make([]domain.Variant, 0)

	for rows.Next() {
		var v domain.Variant
		var priceDiff sql.NullFloat64 // Usar NullFloat64 para lidar com valores NULL no DB

		err := rows.Scan(
			&v.ID, &v.ProductID, &v.Attribute, &v.Value, &v.Barcode,
			&priceDiff, // Scan para NullFloat64
		)
		if err != nil {
			r.logger.Error("Falha ao mapear linha de variante do DB.", err)
			return nil, apperror.NewDBError("Falha ao mapear variações do produto (DB)", err)
		}

		// Atribui o valor de PriceDiff se não for NULL
		if priceDiff.Valid {
			v.PriceDiff = priceDiff.Float64
		}

		variants = append(variants, v)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Erro após iteração das linhas de variantes do DB.", err)
		return nil, apperror.NewDBError("Erro após iteração de variações (DB)", err)
	}

	r.logger.Info("Variantes encontradas com sucesso.", map[string]interface{}{"product_id": productID, "count": len(variants)})
	return variants, nil
}

// FindAll busca uma lista de produtos, aplicando filtros e paginação.
// (Implementa um dos métodos da interface domain.ProductRepository)
func (r *ProductRepository) FindAll(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, error) {
	r.logger.Debug("Iniciando FindAll de produtos no repositório.", map[string]interface{}{"filter": filter})

	ctxGo, cancel := context.WithTimeout(ctx.(context.Context), r.DBTimeout)
	defer cancel()

	// --- 1. Construção Dinâmica da Query ---

	// Base da Query
	query := `
        SELECT id, sku, name, description, price, is_active, created_at, updated_at
        FROM products 
        WHERE 1=1 ` // 1=1 é um truque para facilitar a concatenação de WHERE clauses

	args := []interface{}{}
	argCounter := 1 // Contador para os parâmetros SQL ($1, $2, ...)

	// Aplicar Filtros (Exemplo: Name e SKU)
	if filter.Name != "" {
		query += fmt.Sprintf(" AND name ILIKE $%d", argCounter) // ILIKE para busca case-insensitive
		args = append(args, "%"+filter.Name+"%")
		argCounter++
	}

	if filter.SKU != "" {
		query += fmt.Sprintf(" AND sku = $%d", argCounter)
		args = append(args, filter.SKU)
		argCounter++
	}

	if filter.ActiveOnly {
		query += fmt.Sprintf(" AND is_active = $%d", argCounter)
		args = append(args, true)
		argCounter++
	}

	// Opcional: Ordenar para garantir consistência na paginação
	query += " ORDER BY created_at DESC"

	// --- 2. Aplicar Paginação (LIMIT e OFFSET) ---

	// Definir LIMIT (máximo de itens por página)
	limit := filter.Limit
	if limit <= 0 {
		limit = 10 // Padrão: 10 itens por página
	}
	query += fmt.Sprintf(" LIMIT $%d", argCounter)
	args = append(args, limit)
	argCounter++

	// Definir OFFSET (Pular itens: (Page - 1) * Limit)
	offset := (filter.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" OFFSET $%d", argCounter)
	args = append(args, offset)

	r.logger.Debug("Executando FindAll query", map[string]interface{}{"sql": query, "args_count": len(args)})

	// --- 3. Executar a Query e Mapear Resultados ---
	rows, err := r.DB.QueryContext(ctxGo, query, args...)
	if err != nil {
		r.logger.Error("Falha ao executar FindAll query.", err)
		return nil, errors.NewDBError("Falha ao buscar produtos (FindAll)", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		err := rows.Scan(
			&p.ID,
			&p.SKU,
			&p.Name,
			&p.Description,
			&p.Price,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Falha ao mapear produto na iteração de FindAll.", err)
			return nil, errors.NewDBError("Falha ao mapear produtos do DB (FindAll)", err)
		}

		// 🚨 Opcional: Anexar Variações (Se necessário, você chamaria FindVariantsByProductID aqui)
		// Por questões de performance na listagem, geralmente as variantes NÃO são carregadas aqui.
		// Se precisar, adicione a lógica de FindVariantsByProductID para cada produto.

		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Erro após iteração de produtos FindAll.", err)
		return nil, errors.NewDBError("Erro após iteração de produtos (FindAll)", err)
	}

	r.logger.Info("FindAll concluído com sucesso.", map[string]interface{}{"total_results": len(products)})
	return products, nil
}
