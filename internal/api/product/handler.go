package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger" // Importação correta do nosso pacote Logger
	"net/http"
	"strings"
)

// ProductService define o contrato que o Handler espera da camada de Serviço.
// Usamos a assinatura com o tipo abstrato domain.Context para manter a pureza do domínio.
type ProductService interface {
	CreateProduct(ctx domain.Context, p domain.Product, variants []domain.Variant) (domain.Product, error)
	GetProductByID(ctx domain.Context, id string) (domain.Product, error)
	// ...
}

// Handler agrupa todos os métodos de Handler do produto.
type Handler struct {
	Service ProductService
	Logger  logger.Logger // 🚨 CORREÇÃO 1: Adicionar o campo Logger com a interface correta
}

// NewHandler cria uma nova instância do Handler, injetando o Service e o Logger.
func NewHandler(svc ProductService, log logger.Logger) *Handler {
	// 🚨 CORREÇÃO 2: Salvar o Logger injetado na struct
	return &Handler{
		Service: svc,
		Logger:  log,
	}
}

// --- Funções Auxiliares (do passo anterior, adaptadas) ---

// handleServiceResponse processa erros de serviço e envia respostas padronizadas ao cliente.
func (h *Handler) handleServiceResponse(w http.ResponseWriter, r *http.Request, data interface{}, err error, successStatus int) {
	if err == nil {
		// Sucesso
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(successStatus)

		// Log de Sucesso (Registro da operação)
		h.Logger.Info("Requisição concluída com sucesso", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": successStatus,
		})

		if data != nil {
			if jsonErr := json.NewEncoder(w).Encode(data); jsonErr != nil {
				h.Logger.Error("Falha ao codificar JSON de resposta", jsonErr)
				http.Error(w, "Erro ao codificar resposta", http.StatusInternalServerError)
			}
		}
		return
	}

	// TRATAMENTO DE ERROS (Módulo: Error Handling)
	status, category, message := apperror.MapToHTTPStatus(err)

	if status >= 500 {
		h.Logger.Error(fmt.Sprintf("Erro de Servidor: %s", category), err)
	} else {
		// Erros de cliente (4xx) são logged como info/warn
		h.Logger.Debug(fmt.Sprintf("Requisição rejeitada com status %d. Categoria: %s", status, category), map[string]interface{}{"path": r.URL.Path})
	}

	errorResponse := map[string]interface{}{
		"code":     status,
		"category": category,
		"message":  message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse)
}

// --- Handlers de Produto ---

// CreateProductHandler lida com a requisição POST /v1/products.
func (h *Handler) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// Aqui, o log de erro simples é aceitável, pois é um erro de protocolo base.
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// O contexto nativo (context.Context) será passado como domain.Context
	ctx := r.Context()

	// Decodificação do Payload (Usando struct anônima temporária para incluir Variants)
	var productRequest struct {
		Product  domain.Product
		Variants []domain.Variant
	}

	if err := json.NewDecoder(r.Body).Decode(&productRequest); err != nil {
		// Usa a função padronizada para erros de validação
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Payload inválido. Verifique o formato JSON."), http.StatusCreated)
		return
	}

	// 1. Chamar o Serviço (Lógica de Negócio)
	newProduct, err := h.Service.CreateProduct(ctx, productRequest.Product, productRequest.Variants)

	if err != nil {

		// --- Interceptação e Log de Erros 500 ---

		// 🚨 NOVO: Variável placeholder para o tipo InternalError
		var internalErr *apperror.InternalError

		// errors.As verifica se algum erro na cadeia (Unwrap) é do tipo *InternalError.
		if errors.As(err, &internalErr) {

			// O erro é um InternalError (que inclui DBError).
			// O h.Logger irá imprimir a CAUSA RAIZ (o erro SQL subjacente).
			h.Logger.Error("ERRO CRÍTICO (500) NA TRANSAÇÃO SQL:", internalErr)

			// Passamos o erro para a função auxiliar que o formatará como um 500 genérico.
			h.handleServiceResponse(w, r, nil, internalErr, http.StatusCreated)
			return
		}

		// Se não for um InternalError (500), é um erro de cliente (400, 404, 409).
		// A função auxiliar handleServiceResponse cuidará do mapeamento.
		h.handleServiceResponse(w, r, nil, err, http.StatusCreated)
		return
	}
	// 2. Resposta de Sucesso ou Erro (Usando a função auxiliar)
	h.handleServiceResponse(w, r, newProduct, err, http.StatusCreated)
}

// GetProductByIDHandler lida com a requisição GET /v1/products/{id}.
func (h *Handler) GetProductByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// 1. Extrair ID do Segmento da URL
	// Assumimos que a URL foi roteada corretamente e o ID está no caminho.
	// Exemplo de extração simples:
	segments := r.URL.Path
	if len(segments) == 0 {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("ID do produto ausente na URL."), http.StatusOK)
		return
	}

	// Supondo uma URL como /v1/products/UUID
	// Vamos usar a função auxiliar para extrair o ID
	id := extractIDFromURL(r.URL.Path)

	if id == "" {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Formato de URL inválido ou ID ausente."), http.StatusOK)
		return
	}

	// 2. Chamar o Serviço (Lógica de Negócio)
	product, err := h.Service.GetProductByID(ctx, id)

	// 3. Tratamento de Erro
	if err != nil {
		// Log detalhado de erros 500 (mesmo padrão do CreateProductHandler)
		var internalErr *apperror.InternalError
		if errors.As(err, &internalErr) {
			h.Logger.Error("ERRO CRÍTICO (500) NA BUSCA DO PRODUTO:", internalErr)
			// handleServiceResponse retornará 500 genérico
		}

		// Se for um NotFoundError (404), o handleServiceResponse fará o mapeamento
		// Se for um InternalError (500), o handleServiceResponse fará o mapeamento
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	// 4. Resposta de Sucesso (200 OK)
	h.handleServiceResponse(w, r, product, nil, http.StatusOK)
}

// extractIDFromURL é uma função simples para extrair o último segmento de uma URL /path/to/id
// OBS: Em um projeto real, usar um router como Gorilla Mux ou Chi simplificaria isso.
func extractIDFromURL(path string) string {
	// A rota esperada é /v1/products/ID
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "products" && parts[len(parts)-1] != "v1" {
		return parts[len(parts)-1]
	}
	return ""
}
