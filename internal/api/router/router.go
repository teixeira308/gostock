package router

import (
	"net/http"
	"strings"

	// Importe o TokenService e o Middleware
	"gostock/internal/api/product"
	"gostock/internal/api/user"
	"gostock/internal/domain"
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
// 🚨 ATUALIZAÇÃO DA ASSINATURA: Agora recebe o TokenService.
func NewRouter(productHandler *product.Handler, userHandler *user.Handler, tokenSvc TokenService) *http.ServeMux {
	mux := http.NewServeMux()

	// 1. Inicializa o Middleware de Autorização
	authMiddleware := middleware.NewAuthMiddleware(tokenSvc)

	// --- Rotas de Produto (/v1/products) ---
	mux.HandleFunc("/v1/products", func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case http.MethodPost:
			// Rota protegida: Cria um novo produto (AuthN + AuthZ)
			permissionMware := middleware.PermissionMiddleware(domain.RoleAdmin)
			finalHandler := permissionMware(productHandler.CreateProductHandler)
			authMiddleware(finalHandler).ServeHTTP(w, r)

		case http.MethodGet:
			// Se houver tempo, aqui seria o handler para GetAllProductsHandler
			http.Error(w, "Listagem de produtos (GET /v1/products) não implementada.", http.StatusNotImplemented)

		default:
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})

	// O "/" no final captura qualquer sub-caminho, assumindo que é o ID.
	mux.HandleFunc("/v1/products/", func(w http.ResponseWriter, r *http.Request) {

		// Esta rota exige que haja um ID na URL para ser válida
		path := strings.Trim(r.URL.Path, "/")
		segments := strings.Split(path, "/")

		// Verifica se há pelo menos 3 segmentos: ["v1", "products", "ID"]
		if len(segments) != 3 {
			// Redireciona para o handler de "não encontrado" ou retorna 404
			http.Error(w, "ID do produto inválido ou ausente na URL.", http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Rota pública: Obtém produto por ID
			productHandler.GetProductByIDHandler(w, r)

		case http.MethodPut:
			// Futuramente: UpdateProductHandler

		case http.MethodDelete:
			// Futuramente: DeleteProductHandler

		default:
			http.Error(w, "Método não permitido para esta URL.", http.StatusMethodNotAllowed)
		}
	})
	// ... (Rotas de Usuário e Swagger permanecem as mesmas) ...
	mux.HandleFunc("/v1/register", userHandler.RegisterUserHandler)
	mux.HandleFunc("/v1/login", userHandler.LoginUserHandler)

	// Rota para o Swagger UI
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	return mux
}
