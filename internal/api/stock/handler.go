package stock

import (
	"encoding/json"
	"fmt"
	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
	"net/http"
)

// StockService define o contrato que o Handler espera da camada de Serviço.
type StockService interface {
	AdjustStock(ctx domain.Context, adjustment domain.StockAdjustmentRequest) (domain.StockLevel, error)
}

// Handler agrupa todos os métodos de Handler de estoque.
type Handler struct {
	Service StockService
	Logger  logger.Logger
}

// NewHandler cria uma nova instância do Handler, injetando o Service e o Logger.
func NewHandler(svc StockService, log logger.Logger) *Handler {
	return &Handler{
		Service: svc,
		Logger:  log,
	}
}

// handleServiceResponse processa erros de serviço e envia respostas padronizadas ao cliente.
func (h *Handler) handleServiceResponse(w http.ResponseWriter, r *http.Request, data interface{}, err error, successStatus int) {
	if err == nil {
		// Sucesso
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(successStatus)
		if data != nil {
			if jsonErr := json.NewEncoder(w).Encode(data); jsonErr != nil {
				h.Logger.Error("Falha ao codificar JSON de resposta", jsonErr)
				http.Error(w, "Erro ao codificar resposta", http.StatusInternalServerError)
			}
		}
		return
	}

	// TRATAMENTO DE ERROS
	status, category, message := apperror.MapToHTTPStatus(err)

	if status >= 500 {
		h.Logger.Error(fmt.Sprintf("Erro de Servidor: %s", category), err)
	} else {
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

// AdjustStockHandler lida com a requisição POST /v1/stock/update.
func (h *Handler) AdjustStockHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	var adjustmentRequest domain.StockAdjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&adjustmentRequest); err != nil {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Payload inválido. Verifique o formato JSON."), http.StatusBadRequest)
		return
	}

	stockLevel, err := h.Service.AdjustStock(ctx, adjustmentRequest)
	if err != nil {
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	h.handleServiceResponse(w, r, stockLevel, nil, http.StatusOK) // 200 OK for successful adjustment
}
