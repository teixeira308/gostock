package router

import (
	"gostock/internal/api/product"
	"net/http"
)

// Router é a função principal que configura e retorna o roteador HTTP.
// Recebe os Handlers já inicializados.
func NewRouter(productHandler *product.Handler) http.Handler {

	// Usamos o ServeMux padrão do net/http para roteamento
	mux := http.NewServeMux()

	// 1. Rotas do Módulo de Produtos
	// POST /v1/products (Create)
	mux.HandleFunc("/v1/products", productHandler.CreateProductHandler)

	// GET /v1/products/{id} (Get by ID)
	// mux.HandleFunc("/v1/products/", productHandler.GetProductByIDHandler)
	// ^ Isso exige lógica de extração de ID

	// 2. Rota de Ping/Health Check (Pode ser separada, mas colocamos aqui por simplicidade)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// 3. Adicionar Middlewares Globais aqui mais tarde (e.g., Logging, CORS)
	// return LoggingMiddleware(mux)

	return mux
}
