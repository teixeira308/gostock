package domain

// ErrorResponse é a estrutura padronizada para respostas de erro na API.
// @Description Estrutura padronizada para respostas de erro na API.
type ErrorResponse struct {
	Code     int    `json:"code" example:"400"`
	Category string `json:"category" example:"VALIDATION_ERROR"`
	Message  string `json:"message" example:"O nome do armazém não pode ser vazio."`
}
