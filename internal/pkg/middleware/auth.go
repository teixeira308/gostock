package middleware

import (
	"context"
	"gostock/internal/domain" // Para usar a role do usuário
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/token"
	"net/http"
)

// UserClaimsKey é a chave usada para armazenar as claims do usuário no contexto.
// Usamos um tipo int para garantir que esta chave seja única e não haja conflito
// com outras chaves string (Context Keys devem ser não-exportadas e de um tipo único).
type ContextKey int

const (
	UserClaimsKey ContextKey = iota
)

// UserClaims representa os dados do usuário extraídos do token JWT,
// que serão anexados ao contexto.
type UserClaims struct {
	UserID string
	Role   domain.UserRole
}

// TokenService define o contrato de validação necessário para o middleware.
type TokenService interface {
	ValidateToken(tokenString string) (*token.CustomClaims, error)
}

// NewAuthMiddleware cria uma função de middleware que valida um JWT e anexa as claims
// (UserID e Role) ao contexto da requisição.
func NewAuthMiddleware(tokenSvc TokenService) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			// 1. Extrair o Token do Header Authorization: Bearer <token>
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				// Se o header estiver ausente ou malformado, retorna 401
				http.Error(w, apperror.NewUnauthorizedError("Token de autorização ausente ou malformado.").Error(), http.StatusUnauthorized)
				return
			}

			tokenString := authHeader[7:]

			// 2. Validar o Token
			claims, err := tokenSvc.ValidateToken(tokenString)
			if err != nil {
				// Se a validação falhar (expirado, inválido, etc.), retorna 401
				http.Error(w, apperror.NewUnauthorizedError("Token inválido ou expirado.").Error(), http.StatusUnauthorized)
				return
			}

			// 3. Anexar Claims ao Contexto
			userClaims := UserClaims{
				UserID: claims.UserID,
				Role:   domain.UserRole(claims.Role), // Converte a string da claim para domain.UserRole
			}

			// Cria um novo contexto com as claims anexadas
			ctx := context.WithValue(r.Context(), UserClaimsKey, userClaims)

			// Chama o próximo handler com o novo contexto
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}

// GetUserClaimsFromContext é uma função utilitária para extrair as claims no handler.
func GetUserClaimsFromContext(ctx context.Context) (UserClaims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(UserClaims)
	return claims, ok
}

// RequiredRoles define a lista de funções que têm permissão para acessar o recurso.
func PermissionMiddleware(requiredRoles ...domain.UserRole) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			// 1. Tentar extrair as Claims do contexto
			claims, ok := GetUserClaimsFromContext(r.Context())

			// Se o AuthMiddleware não foi executado ou falhou em anexar as claims,
			// ou se houver uma falha interna, tratamos como não autorizado.
			if !ok {
				http.Error(w, apperror.NewUnauthorizedError("Autorização necessária. Token não processado.").Error(), http.StatusUnauthorized)
				return
			}

			// 2. Verificar Permissão (AuthZ)
			isAuthorized := false
			userRole := claims.Role

			// Itera sobre as roles necessárias para ver se a role do usuário corresponde a alguma delas.
			for _, requiredRole := range requiredRoles {
				if userRole == requiredRole {
					isAuthorized = true
					break
				}
			}

			if !isAuthorized {
				// Se a role do usuário não estiver na lista de roles permitidas (requiredRoles)
				http.Error(w, apperror.NewUnauthorizedError("Acesso negado. Você não tem a permissão necessária.").Error(), http.StatusForbidden) // 403 Forbidden
				return
			}

			// 3. Permissão concedida: Chama o próximo handler
			next.ServeHTTP(w, r)
		}
	}
}
