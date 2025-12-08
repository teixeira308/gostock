package productservice

import (
	"context" // Necessário para o casting e chamadas de infraestrutura
	"fmt"
	"time"

	// Importar o pacote errors nativo (para errors.Is e errors.Unwrap)
	"errors"

	"github.com/google/uuid"

	"gostock/internal/domain"
	apperror "gostock/internal/errors" // 🚨 CORREÇÃO: Usar o nome renomeado para evitar conflito
)

// ProductRepository define o contrato (interface) que este Serviço espera
// da camada de Persistência (DB, Cache).
// Usamos context.Context nativo para que o Service possa passar o contexto com timeout para o Repo.
type ProductRepository interface {
	// 🚨 CORREÇÃO DE ASSINATURA: A implementação deve usar context.Context nativo,
	// pois o Repositório é a camada de infraestrutura.
	Save(ctx context.Context, product domain.Product) (domain.Product, error)
	FindByID(ctx domain.Context, id string) (domain.Product, error)
}

// Service é a estrutura que implementa a interface domain.ProductService.
type Service struct {
	repo ProductRepository
}

// NewService cria e retorna uma nova instância do Serviço de Produto.
func NewService(repo ProductRepository) *Service {
	return &Service{repo: repo}
}

// --- Implementação: CreateProduct ---
func (s *Service) CreateProduct(ctx domain.Context, product domain.Product, variants []domain.Variant) (domain.Product, error) {

	product.Variants = variants

	// 🚨 NOVO: 1. Validação de Domínio
	if err := s.validateProduct(product); err != nil {
		return domain.Product{}, err
	}

	// 2. Geração de IDs (se a variação não tiver ID, o serviço a define)
	if product.ID == "" {
		product.ID = uuid.New().String()
	}
	product.IsActive = true
	now := time.Now().UTC()
	product.CreatedAt = now
	product.UpdatedAt = now

	for i := range product.Variants {
		if product.Variants[i].ID == "" {
			product.Variants[i].ID = uuid.New().String()
		}
		// Linkar a chave estrangeira (ProductID)
		product.Variants[i].ProductID = product.ID
	}

	// 1. Casting e Contexto
	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
	}

	// 3. Delegação para a Camada de Persistência (Repository)
	createdProduct, err := s.repo.Save(ctxGo, product) // Chamada com ctxGo
	if err != nil {
		// Propaga o erro retornado pelo Repositório (que deve ser um apperror.InternalError ou similar)
		return domain.Product{}, fmt.Errorf("falha ao salvar produto no repositório: %w", err)
	}

	return createdProduct, nil
}

// --- Implementação: GetProductByID (Única e Corrigida) ---
func (s *Service) GetProductByID(ctx domain.Context, id string) (domain.Product, error) {

	// 1. Validação de Formato (Business Logic)
	if _, err := uuid.Parse(id); err != nil {
		return domain.Product{}, apperror.NewValidationError("O ID do produto deve ser um UUID válido.")
	}

	// 2. Casting e Configuração do Contexto (Converte domain.Context para context.Context)
	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
	}

	// 3. Delegação para o Repositório
	product, err := s.repo.FindByID(ctxGo, id)

	if err != nil {
		// 4. Tratamento e Tradução de Erro (Mapeamento de Erros)

		// Verifica se o erro retornado pelo Repositório é um NotFoundError.
		// 🚨 CORREÇÃO: Usar errors.Is do pacote nativo Go para verificar a cadeia de erros
		var notFound *apperror.NotFoundError
		if errors.Is(err, notFound) {
			// Se o Repositório retornou NotFound, retornamos o erro de negócio 404.
			return domain.Product{}, apperror.NewNotFoundError(fmt.Sprintf("Produto com ID %s não foi encontrado.", id))
		}

		// Para qualquer outro erro (DB falhou, conexão perdida - 500), propagamos o erro de infraestrutura.
		return domain.Product{}, err
	}

	// 5. Sucesso
	return product, nil
}

// validateProduct verifica as regras de negócio básicas do produto e suas variações.
func (s *Service) validateProduct(p domain.Product) error {
	if p.SKU == "" {
		return apperror.NewValidationError("O SKU do produto é obrigatório.")
	}
	if p.Name == "" {
		return apperror.NewValidationError("O nome do produto é obrigatório.")
	}
	if p.Price <= 0 {
		return apperror.NewValidationError("O preço do produto deve ser um valor positivo.")
	}

	// Validação das Variações
	if len(p.Variants) == 0 {
		return apperror.NewValidationError("O produto deve ter pelo menos uma variação.")
	}

	for i, v := range p.Variants {
		if v.Attribute == "" || v.Value == "" {
			return apperror.NewValidationError(fmt.Sprintf("Atributo ou valor da variação %d está vazio.", i+1))
		}
		if v.PriceDiff < 0 {
			return apperror.NewValidationError(fmt.Sprintf("A diferença de preço da variação %d não pode ser negativa.", i+1))
		}
		if v.Barcode == "" {
			return apperror.NewValidationError(fmt.Sprintf("O código de barras da variação %d é obrigatório.", i+1))
		}
	}

	return nil
}
