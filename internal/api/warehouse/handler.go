package warehouse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
)

// WarehouseService define o contrato que o Handler espera da camada de Serviço.
type WarehouseService interface {
	CreateWarehouse(ctx domain.Context, warehouse domain.Warehouse) (domain.Warehouse, error)
	GetWarehouseByID(ctx domain.Context, id string) (domain.Warehouse, error)
	GetAllWarehouses(ctx domain.Context) ([]domain.Warehouse, error)
	UpdateWarehouse(ctx domain.Context, warehouse domain.Warehouse) (domain.Warehouse, error)
	DeleteWarehouse(ctx domain.Context, id string) error
}

// Handler agrupa todos os métodos de Handler de armazéns.
type Handler struct {
	Service WarehouseService
	Logger  logger.Logger
}

// NewHandler cria uma nova instância do Handler, injetando o Service e o Logger.
func NewHandler(svc WarehouseService, log logger.Logger) *Handler {
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

// CreateWarehouseHandler lida com a requisição POST /v1/warehouses.
// @Summary Cria um novo armazém
// @Description Cria um novo armazém no sistema.
// @Tags warehouses
// @Accept json
// @Produce json
// @Param warehouse body domain.Warehouse true "Dados do armazém para criação"
// @Success 201 {object} domain.Warehouse "Armazém criado com sucesso"
// @Failure 400 {object} domain.ErrorResponse "Payload inválido"
// @Failure 500 {object} domain.ErrorResponse "Erro interno do servidor"
// @Security ApiKeyAuth
// @Router /warehouses [post]
func (h *Handler) CreateWarehouseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	var warehouse domain.Warehouse
	if err := json.NewDecoder(r.Body).Decode(&warehouse); err != nil {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Payload inválido. Verifique o formato JSON."), http.StatusBadRequest)
		return
	}

	createdWarehouse, err := h.Service.CreateWarehouse(ctx, warehouse)
	if err != nil {
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	h.handleServiceResponse(w, r, createdWarehouse, nil, http.StatusCreated)
}

// GetWarehouseByIDHandler lida com a requisição GET /v1/warehouses/{id}.
// @Summary Obtém um armazém por ID
// @Description Busca um armazém específico pelo seu ID.
// @Tags warehouses
// @Produce json
// @Param id path string true "ID do Armazém"
// @Success 200 {object} domain.Warehouse "Armazém encontrado"
// @Failure 404 {object} domain.ErrorResponse "Armazém não encontrado"
// @Failure 500 {object} domain.ErrorResponse "Erro interno do servidor"
// @Router /warehouses/{id} [get]
func (h *Handler) GetWarehouseByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	id := strings.TrimPrefix(r.URL.Path, "/v1/warehouses/") // Assumes URL path like /v1/warehouses/{id}

	warehouse, err := h.Service.GetWarehouseByID(ctx, id)
	if err != nil {
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	h.handleServiceResponse(w, r, warehouse, nil, http.StatusOK)
}

// GetAllWarehousesHandler lida com a requisição GET /v1/warehouses.
// @Summary Lista todos os armazéns
// @Description Retorna uma lista de todos os armazéns cadastrados.
// @Tags warehouses
// @Produce json
// @Success 200 {array} domain.Warehouse "Lista de armazéns"
// @Failure 500 {object} domain.ErrorResponse "Erro interno do servidor"
// @Router /warehouses [get]
func (h *Handler) GetAllWarehousesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	warehouses, err := h.Service.GetAllWarehouses(ctx)
	if err != nil {
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	h.handleServiceResponse(w, r, warehouses, nil, http.StatusOK)
}

// UpdateWarehouseHandler lida com a requisição PUT /v1/warehouses/{id}.
// @Summary Atualiza um armazém
// @Description Atualiza os dados de um armazém existente.
// @Tags warehouses
// @Accept json
// @Produce json
// @Param id path string true "ID do Armazém"
// @Param warehouse body domain.Warehouse true "Dados do armazém para atualização"
// @Success 200 {object} domain.Warehouse "Armazém atualizado com sucesso"
// @Failure 400 {object} domain.ErrorResponse "Payload inválido"
// @Failure 404 {object} domain.ErrorResponse "Armazém não encontrado"
// @Failure 500 {object} domain.ErrorResponse "Erro interno do servidor"
// @Security ApiKeyAuth
// @Router /warehouses/{id} [put]
func (h *Handler) UpdateWarehouseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	id := strings.TrimPrefix(r.URL.Path, "/v1/warehouses/") // Assumes URL path like /v1/warehouses/{id}

	var warehouse domain.Warehouse
	if err := json.NewDecoder(r.Body).Decode(&warehouse); err != nil {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Payload inválido. Verifique o formato JSON."), http.StatusBadRequest)
		return
	}
	warehouse.ID = id // Ensure ID from URL path is used

	updatedWarehouse, err := h.Service.UpdateWarehouse(ctx, warehouse)
	if err != nil {
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	h.handleServiceResponse(w, r, updatedWarehouse, nil, http.StatusOK)
}

// DeleteWarehouseHandler lida com a requisição DELETE /v1/warehouses/{id}.
// @Summary Deleta um armazém
// @Description Remove um armazém do sistema pelo seu ID.
// @Tags warehouses
// @Param id path string true "ID do Armazém"
// @Success 204 "Nenhum conteúdo"
// @Failure 404 {object} domain.ErrorResponse "Armazém não encontrado"
// @Failure 500 {object} domain.ErrorResponse "Erro interno do servidor"
// @Security ApiKeyAuth
// @Router /warehouses/{id} [delete]
func (h *Handler) DeleteWarehouseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	id := strings.TrimPrefix(r.URL.Path, "/v1/warehouses/") // Assumes URL path like /v1/warehouses/{id}

	err := h.Service.DeleteWarehouse(ctx, id)
	if err != nil {
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	h.handleServiceResponse(w, r, nil, nil, http.StatusNoContent)
}
