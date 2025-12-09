package router

import (
	"net/http"
	"strings"
	"time"

	"gostock/internal/api/product"
	"gostock/internal/api/user"
	"gostock/internal/domain"
	"gostock/internal/pkg/cache"
	"gostock/internal/pkg/middleware"
	"gostock/internal/pkg/token"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Defina a interface do TokenService para ser passada ao Router, evitando importação direta
// do pacote token (opcional se você já usa o pacote token, mas boa prática de abstração).
type TokenService interface {
	ValidateToken(tokenString string) (*token.CustomClaims, error)
}

// NewRouter configura e retorna o roteador da aplicação.
// 🚨 ATUALIZAÇÃO DA ASSINATURA: Agora recebe o TokenService e o cache.Client.
func NewRouter(productHandler *product.Handler, userHandler *user.Handler, tokenSvc TokenService, cacheClient cache.Client) *http.ServeMux {
	mux := http.NewServeMux()

	// 1. Inicializa os Middlewares
	authMiddleware := middleware.NewAuthMiddleware(tokenSvc)
	// Limita a 10 requisições por minuto por IP
	rateLimitMiddleware := middleware.RateLimiter(cacheClient, 10, time.Minute)

	// --- Rotas de Produto (/v1/products) ---
	// Aplica o Rate Limiter a todas as rotas de produto
	productRoutes := http.NewServeMux()
	productRoutes.HandleFunc("/v1/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			permissionMware := middleware.PermissionMiddleware(domain.RoleAdmin)
			finalHandler := permissionMware(productHandler.CreateProductHandler)
			authMiddleware(finalHandler).ServeHTTP(w, r)
		case http.MethodGet:
			productHandler.GetProductsHandler(w, r)
		default:
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})
	productRoutes.HandleFunc("/v1/products/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		segments := strings.Split(path, "/")
		if len(segments) != 3 {
			http.Error(w, "ID do produto inválido ou ausente na URL.", http.StatusNotFound)
			return
		}
		switch r.Method {
		case http.MethodGet:
			productHandler.GetProductByIDHandler(w, r)
		default:
			http.Error(w, "Método não permitido para esta URL.", http.StatusMethodNotAllowed)
		}
	})

	// --- Rotas de Usuário ---
	userRoutes := http.NewServeMux()
	userRoutes.HandleFunc("/v1/register", userHandler.RegisterUserHandler)
	userRoutes.HandleFunc("/v1/login", userHandler.LoginUserHandler)

	// Aplica o rate limiter
	mux.Handle("/v1/products", rateLimitMiddleware(productRoutes))
	mux.Handle("/v1/products/", rateLimitMiddleware(productRoutes))
	mux.Handle("/v1/register", rateLimitMiddleware(userRoutes))
	mux.Handle("/v1/login", rateLimitMiddleware(userRoutes))

	// Rota para o Swagger UI
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	return mux
}
