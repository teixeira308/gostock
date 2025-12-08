package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	// Nossos pacotes de infraestrutura e utilitários
	"gostock/config"
	"gostock/internal/pkg/cache"
	"gostock/internal/pkg/database"
	"gostock/internal/pkg/logger"
	"gostock/internal/pkg/token"

	// Camadas do Produto para Injeção de Dependências
	"gostock/internal/api/product" // Handlers
	"gostock/internal/api/router"  // Roteador central
	"gostock/internal/api/user"
	"gostock/internal/repository/productrepo" // Acesso a Dados
	"gostock/internal/repository/userrepo"
	"gostock/internal/service/productservice" // Lógica de Negócio
	"gostock/internal/service/userservice"
)

func main() {
	// 1. Configuração e Inicialização
	log.Println("⚡ Inicializando serviço GoStock...")
	// 0. CARREGAR VARIÁVEIS DE AMBIENTE (.env)
	// O godotenv.Load() procura por um arquivo chamado .env na raiz.
	if err := godotenv.Load(); err != nil {
		// Se o arquivo .env não for encontrado (ou houver erro de leitura),
		// avisamos, mas continuamos, pois as variáveis essenciais podem
		// estar no ambiente do sistema (ex: Docker).
		log.Println("⚠️ Aviso: Arquivo .env não encontrado ou erro de leitura. Carregando configs apenas do ambiente do sistema.")
	}

	cfg := config.LoadConfig() // Carrega as configurações (URLs, Timeouts, etc.)
	log := logger.NewLogger(cfg.LogLevel)
	log.Info("Configurações carregadas.", nil)

	// 2. Conexão com Recursos de Infraestrutura

	// A. Banco de Dados (PostgreSQL)
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Falha ao conectar ao banco de dados.", err)
	}
	defer db.Close() // Fecha a conexão de DB ao sair
	log.Info("Conexão PostgreSQL estabelecida.", nil)

	// B. Cache (Redis)
	cacheClient := cache.NewRedisClient(cfg.RedisAddr)
	log.Info("Conexão Redis estabelecida.", nil)

	// 3. INJEÇÃO DE DEPENDÊNCIAS (Montagem da Clean Architecture)
	// Ordem: Repository -> Service -> Handler

	// A. Repositório (Camada de Acesso a Dados)
	// Recebe as conexões de Infraestrutura
	productRepo := productrepo.NewProductRepository(db, cacheClient, cfg.DBTimeout, log) // Passando o logger para o repositório
	log.Debug("Repositório de Produto inicializado.", nil)

	// B. Serviço (Camada de Lógica de Negócio)
	// Recebe o Repositório (a interface domain.ProductRepository)
	productSvc := productservice.NewService(productRepo, log) // Passando o logger para o serviço
	log.Debug("Serviço de Produto inicializado.", nil)

	// C. Handler (Camada de Apresentação)
	// Recebe o Serviço (a interface domain.ProductService)
	productHandler := product.NewHandler(productSvc, log) // Passando o logger para o handler
	log.Debug("Handler de Produto inicializado.", nil)

	// C. Serviço de Tokens (JWT)
	jwtExpiry := time.Hour * time.Duration(cfg.JWTExpiryHours)
	tokenSvc := token.NewService(cfg.JWTSecretKey, jwtExpiry)
	log.Debug("Serviço de Tokens JWT inicializado.", nil)

	// C. Repositório de Usuário (Camada de Acesso a Dados)
	userRepo := userrepo.NewUserRepository(db, cfg.DBTimeout, log) // Passando o logger para o repositório
	log.Debug("Repositório de Usuário inicializado.", nil)

	userSvc := userservice.NewService(userRepo, tokenSvc, log) // Passando o logger para o serviço
	log.Debug("Serviço de Usuário inicializado.", nil)

	// E. Handler de Usuário
	userHandler := user.NewHandler(userSvc, log)
	log.Debug("Handler de Usuário inicializado.", nil)

	// 4. Configuração e Início do Roteador/Servidor

	// O roteador recebe os Handlers e aplica middlewares (futuramente)
	r := router.NewRouter(productHandler, userHandler, tokenSvc, cacheClient)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r, // O roteador final
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Execução e Graceful Shutdown
	go func() {
		log.Info("Servidor GoStock ouvindo na porta", map[string]interface{}{"port": cfg.Port})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Servidor falhou: %v", err)
		}
	}()

	// Lógica do Graceful Shutdown (captura de sinal)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Sinal de encerramento recebido. Desligando servidor...", nil)

	// Timeout para desligamento (usa o contexto)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Desligamento do servidor forçado.", err)
	}

	log.Info("Servidor encerrado com sucesso.", nil)
}

