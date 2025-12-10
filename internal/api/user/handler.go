package user

import (
	"context"
	"encoding/json"
	"net/http"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
)

// UserService define o contrato para as operações de registro e login.
type UserService interface {
	Register(ctx context.Context, registration domain.UserRegistration) (domain.User, error)
	Login(ctx context.Context, email string, password string) (string, error)
}

// LoginRequest representa o payload de entrada para o login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Handler agrupa todos os métodos de Handler do usuário.
type Handler struct {
	Service UserService
	Logger  logger.Logger
}

// NewHandler cria uma nova instância do Handler, injetando o Service e o Logger.
func NewHandler(svc UserService, log logger.Logger) *Handler {
	return &Handler{
		Service: svc,
		Logger:  log,
	}
}

// handleServiceResponse é uma função auxiliar (similar à que você já tem em product/handler.go)
// para padronizar o tratamento de erros e respostas HTTP.
func (h *Handler) handleServiceResponse(w http.ResponseWriter, r *http.Request, data interface{}, err error, successStatus int) {
	// Função auxiliar completa deve ser implementada para evitar repetição.
	// Por brevidade, assumimos que ela está presente ou que usaremos o método do ProductHandler
	// se estiver em um pacote utilitário. Aqui, faremos o essencial:

	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(successStatus)
		if data != nil {
			json.NewEncoder(w).Encode(data)
		}
		return
	}

	// Mapeamento de Erros de Negócio para Status HTTP
	status, category, message := apperror.MapToHTTPStatus(err)

	// Log apenas de erros graves
	if status >= 500 {
		h.Logger.Error("Erro interno no serviço de usuário:", err)
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

// RegisterUserHandler lida com a requisição POST /v1/register.
// @Summary Registra um novo usuário
// @Description Cria um novo usuário, hasheia a senha e salva no banco de dados.
// @Tags users
// @Accept json
// @Produce json
// @Param registration body domain.UserRegistration true "Credenciais de registro (email e senha)"
// @Success 201 {object} domain.User "Usuário criado com sucesso"
// @Failure 400 {object} domain.ErrorResponse "Payload inválido (JSON malformado ou campos obrigatórios ausentes)"
// @Failure 409 {object} domain.ErrorResponse "Email já cadastrado"
// @Failure 500 {object} domain.ErrorResponse "Erro interno do servidor"
// @Router /register [post]
func (h *Handler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var reg domain.UserRegistration
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Payload JSON inválido."), http.StatusCreated)
		return
	}

	// 1. Chamar o Serviço (Haverá hashing e persistência)
	newUser, err := h.Service.Register(ctx, reg)

	if err != nil {
		// Se o serviço falhar, o handleServiceResponse traduzirá o erro.
		// Ex: ConflictError (e-mail duplicado) -> 409
		// Ex: ValidationError -> 400
		h.handleServiceResponse(w, r, nil, err, http.StatusCreated)
		return
	}

	// 2. Resposta de Sucesso (201 Created)
	// O objeto newUser retornado pelo serviço já tem o PasswordHash limpo,
	// pois a struct domain.User usa a tag `json:"-"`.
	h.handleServiceResponse(w, r, newUser, nil, http.StatusCreated)
}

// ... (abaixo do RegisterUserHandler) ...

// LoginUserHandler lida com a requisição POST /v1/login.
// @Summary Autentica um usuário e retorna um JWT
// @Description Recebe email/senha, verifica a validade e emite um JSON Web Token.
// @Tags users
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Credenciais do usuário (email e senha)"
// @Success 200 {object} map[string]string "Token JWT emitido"
// @Failure 400 {object} domain.ErrorResponse "Payload inválido"
// @Failure 401 {object} domain.ErrorResponse "Credenciais inválidas"
// @Failure 500 {object} domain.ErrorResponse "Erro interno do servidor"
// @Router /login [post]
func (h *Handler) LoginUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		h.handleServiceResponse(w, r, nil, apperror.NewValidationError("Payload JSON inválido."), http.StatusOK) // Usando 200/StatusOK temporariamente
		return
	}

	// 1. Chamar o Serviço de Login
	token, err := h.Service.Login(ctx, loginReq.Email, loginReq.Password)

	if err != nil {
		// O handleServiceResponse traduzirá 401 Unauthorized, 400 Validation, 500 Internal
		h.handleServiceResponse(w, r, nil, err, http.StatusOK)
		return
	}

	// 2. Resposta de Sucesso (200 OK com o Token)
	response := map[string]string{"token": token}
	h.handleServiceResponse(w, r, response, nil, http.StatusOK)
}
