package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenService define o contrato para manipulação de JWTs.
type TokenService interface {
	GenerateToken(userID string, userRole string) (string, error)
	ValidateToken(tokenString string) (*CustomClaims, error)
}

// CustomClaims define as informações específicas que queremos armazenar no JWT.
// É obrigatório incorporar jwt.RegisteredClaims.
type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Service implementa a interface TokenService
type Service struct {
	secretKey []byte
	expiry    time.Duration
}

// NewService cria uma nova instância do serviço Token.
func NewService(secretKey string, expiry time.Duration) *Service {
	return &Service{
		secretKey: []byte(secretKey),
		expiry:    expiry,
	}
}

// GenerateToken cria um novo JWT assinado contendo o ID e a Role do usuário.
func (s *Service) GenerateToken(userID string, userRole string) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Role:   userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "GoStock-API",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Assina o token com a chave secreta
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("falha ao assinar o token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken valida o token string e retorna as claims se for válido.
func (s *Service) ValidateToken(tokenString string) (*CustomClaims, error) {
	claims := &CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verifica se o método de assinatura é o esperado (HS256)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		// Trata erros comuns de JWT, como token expirado ou inválido
		return nil, fmt.Errorf("token inválido: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token não é válido")
	}

	// O claims já foi preenchido durante o ParseWithClaims
	return claims, nil
}
