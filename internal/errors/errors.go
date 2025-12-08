package errors

import (
	"fmt"
	"net/http"
)

// AppError é a interface central para todos os erros customizados do GoStock.
// Ela permite que o código externo (Handler) acesse a Categoria e a Mensagem do erro.
type AppError interface {
	Error() string    // Implementa a interface error padrão do Go
	Category() string // Categoria do erro (e.g., "VALIDATION", "NOT_FOUND", "INTERNAL")
	HTTPStatus() int  // Código HTTP sugerido para o Handler
	Unwrap() error    // Permite encapsular erros subjacentes (original error)
}

// --- Tipos de Erro Específicos (Erros de Domínio) ---

// ValidationError representa falhas de validação de dados de entrada.
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string    { return fmt.Sprintf("Erro de Validação: %s", e.Msg) }
func (e *ValidationError) Category() string { return "VALIDATION_ERROR" }
func (e *ValidationError) HTTPStatus() int  { return http.StatusBadRequest } // 400
func (e *ValidationError) Unwrap() error    { return nil }                   // Não encapsula erro subjacente

// NewValidationError cria um novo erro de validação.
func NewValidationError(msg string) AppError {
	return &ValidationError{Msg: msg}
}

// NotFoundError representa a ausência de um recurso solicitado.
type NotFoundError struct {
	Msg string
}

func (e *NotFoundError) Error() string    { return fmt.Sprintf("Recurso não encontrado: %s", e.Msg) }
func (e *NotFoundError) Category() string { return "NOT_FOUND" }
func (e *NotFoundError) HTTPStatus() int  { return http.StatusNotFound } // 404
func (e *NotFoundError) Unwrap() error    { return nil }

// NewNotFoundError cria um novo erro de recurso não encontrado.
func NewNotFoundError(msg string) AppError {
	return &NotFoundError{Msg: msg}
}

// ConflictError representa um conflito na regra de negócio (e.g., OCC, recurso duplicado).
type ConflictError struct {
	Msg string
}

func (e *ConflictError) Error() string    { return fmt.Sprintf("Conflito de estado: %s", e.Msg) }
func (e *ConflictError) Category() string { return "CONFLICT" }
func (e *ConflictError) HTTPStatus() int  { return http.StatusConflict } // 409
func (e *ConflictError) Unwrap() error    { return nil }

// NewConflictError cria um novo erro de conflito (usado em OCC).
func NewConflictError(msg string) AppError {
	return &ConflictError{Msg: msg}
}

// --- Tipos de Erro de Infraestrutura (Encapsulamento) ---

// InternalError representa falhas inesperadas no servidor, serviço ou repositório.
type InternalError struct {
	Msg string
	Err error // Erro original subjacente (e.g., erro do driver SQL)
}

func (e *InternalError) Error() string    { return fmt.Sprintf("Erro Interno: %s", e.Msg) }
func (e *InternalError) Category() string { return "INTERNAL_ERROR" }
func (e *InternalError) HTTPStatus() int  { return http.StatusInternalServerError } // 500
func (e *InternalError) Unwrap() error    { return e.Err }

// NewInternalError cria um erro de servidor (para falhas de lógica ou código não esperado).
func NewInternalError(msg string, err error) AppError {
	return &InternalError{Msg: msg, Err: err}
}

// NewDBError é um atalho para criar um InternalError específico de falhas no DB.
func NewDBError(msg string, err error) AppError {
	// Poderia adicionar lógica aqui para verificar se o erro é de timeout ou conexão.
	return NewInternalError(fmt.Sprintf("%s (DB): %s", msg, err.Error()), err)
}

// --- Helper para o Handler (Tradução Final) ---

// MapToHTTPStatus recebe um erro e o traduz para o código HTTP e corpo de resposta.
func MapToHTTPStatus(err error) (int, string, string) {
	if appErr, ok := err.(AppError); ok {
		// O erro é tipado (ValidationError, NotFoundError, etc.)
		return appErr.HTTPStatus(), appErr.Category(), appErr.Error()
	}

	// Erro não tipado (e.g., erro simples de pacote Go que não implementa AppError)
	// Tratar como erro interno genérico.
	return http.StatusInternalServerError, "UNKNOWN_ERROR", "Ocorreu um erro inesperado."
}
