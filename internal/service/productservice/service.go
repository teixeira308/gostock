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
	Save(ctx context.Context, product domain.Product, variants []domain.Variant) (domain.Product, error)
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

	// 1. Casting e Contexto
	ctxGo, ok := ctx.(context.Context)
	if !ok {
		ctxGo = context.Background()
	}

	// 2. Validação de Regras de Negócio
	if product.Name == "" || product.SKU == "" {
		return domain.Product{}, apperror.NewValidationError("Nome e SKU são obrigatórios para o produto.")
	}
	if product.Price <= 0 {
		return domain.Product{}, apperror.NewValidationError("O preço do produto deve ser positivo.")
	}

	// ... (Preenchimento de IDs, IsActive, CreatedAt/UpdatedAt) ...
	if product.ID == "" {
		product.ID = uuid.New().String()
	}
	product.IsActive = true
	now := time.Now().UTC()
	product.CreatedAt = now
	product.UpdatedAt = now
	for i := range variants {
		if variants[i].ID == "" {
			variants[i].ID = uuid.New().String()
		}
		variants[i].ProductID = product.ID
		if variants[i].Attribute == "" || variants[i].Value == "" {
			return domain.Product{}, apperror.NewValidationError(fmt.Sprintf("Variante %d requer Atributo e Valor.", i+1))
		}
	}

	// 3. Delegação para a Camada de Persistência (Repository)
	createdProduct, err := s.repo.Save(ctxGo, product, variants) // Chamada com ctxGo
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
