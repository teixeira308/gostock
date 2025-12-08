package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger" // Importação correta do nosso pacote Logger
	"gostock/internal/pkg/middleware"
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

	claims, ok := middleware.GetUserClaimsFromContext(ctx)
	if ok {
		// Logamos o ID do usuário que está criando o produto
		h.Logger.Info("Tentativa de criação de produto por", map[string]interface{}{
			"user_id": claims.UserID,
			"role":    claims.Role,
		})

		// Você usaria este ID para anexar o criador ao produto (product.CreatorID = claims.UserID)
	} else {
		// Isso só aconteceria se o middleware falhasse ou fosse ignorado na rota, mas é uma boa prática
		h.Logger.Warn("Tentativa de criação de produto sem claims de usuário no contexto.", nil)
	}

	// Decodificação do Payload (Usando struct anônima temporária para incluir Variants)
	var productRequest struct {
		Product  domain.Product   `json:"Product"`
		Variants []domain.Variant `json:"Variants"`
	}
	if err := json.NewDecoder(r.Body).Decode(&productRequest); err != nil {
		// Usa a função padronizada para erros de validação
		// (Ajustei o status para 400 Bad Request, que é o correto para erro de payload)
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Payload inválido. Verifique o formato JSON."), http.StatusBadRequest)
		return
	}
	productRequest.Product.Variants = productRequest.Variants
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

	ctx := r.Context()

	// 1. Extrair ID do Segmento da URL

	// a. Remove barras extras no início e no fim para normalizar
	path := strings.Trim(r.URL.Path, "/")
	// b. Divide a string em segmentos: ["v1", "products", "3c95b8c8..."]
	segments := strings.Split(path, "/")

	// O ID deve ser o último segmento (índice 2, pois o roteador já validou len == 3)
	if len(segments) != 3 {
		// Se a validação do router falhar, retornamos 400 (Bad Request)
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Formato de URL inválido ou ID ausente."), http.StatusOK)
		return
	}

	productID := segments[2]

	// Verificação de ID vazio (embora o len(segments) == 3 já minimize isso)
	if productID == "" {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("ID do produto é obrigatório."), http.StatusOK)
		return
	}

	// 2. Chamar o Serviço (Lógica de Negócio)
	product, err := h.Service.GetProductByID(ctx, productID)

	// 3. Tratamento de Erro
	if err != nil {
		// O handleServiceResponse fará o mapeamento de NotFoundError (404) ou InternalError (500)
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	// 4. Resposta de Sucesso (200 OK)
	h.handleServiceResponse(w, r, product, nil, http.StatusOK)
}
