package router

import (
	"gostock/internal/api/product"
	"net/http"
)

// NewRouter configura e retorna o roteador HTTP principal.
// Recebe os Handlers já inicializados por injeção de dependências.
func NewRouter(productHandler *product.Handler) http.Handler {

	// Usamos o ServeMux padrão do net/http para roteamento
	// Em projetos maiores, pode-se usar um mux de terceiros (e.g., gorilla/mux, chi)
	mux := http.NewServeMux()

	// --- 1. Rotas de Health Check ---
	// Endpoint /ping (do seu main.go original) agora está no roteador.
	mux.HandleFunc("/ping", PingHandler)

	// --- 2. Rotas do Módulo de Produtos (v1) ---

	// POST /v1/products (Criar Produto)
	mux.HandleFunc("/v1/products", productHandler.CreateProductHandler)

	// GET /v1/products/{id} (Buscar Produto por ID)
	// Será implementado na próxima fase.
	mux.HandleFunc("/v1/products/", productHandler.GetProductByIDHandler)

	// --- 3. Aplicação de Middlewares (Próximos Passos) ---

	// Aqui o mux seria envolvido por middlewares globais:
	// Exemplo: return middleware.LoggingMiddleware(middleware.CORSMiddleware(mux))

	return mux
}

// PingHandler é uma função utilitária para o health check.
// Movemos a lógica do main.go para dentro do pacote router.
func PingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
