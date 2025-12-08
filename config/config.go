package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config armazena todas as configurações do aplicativo GoStock.
// Todos os campos são definidos com base nos requisitos do projeto (DB, Cache, Segurança, Robustez).
type Config struct {
	// Geral
	Port        string
	Environment string
	LogLevel    string

	// Banco de Dados (PostgreSQL)
	DatabaseURL string
	DBTimeout   time.Duration // Módulo: Context and Timeouts

	// Cache (Redis)
	RedisAddr    string
	CacheTimeout time.Duration

	// Segurança (JWT)
	JWTSecretKey string
	TokenExpiry  time.Duration

	// Rate Limiting (RNF 5.2)
	RateLimitMaxRequests int
	RateLimitPeriod      time.Duration
}

// LoadConfig carrega as configurações a partir das variáveis de ambiente.
func LoadConfig() *Config {
	// Simular o carregamento de um arquivo .env em desenvolvimento,
	// mas focando na leitura de os.Getenv para não adicionar dependências externas agora.

	cfg := &Config{
		// 1. Geral
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENV", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// 2. Banco de Dados (PostgreSQL)
		// mustGetEnv garante que a aplicação não inicie se não houver credenciais de DB
		DatabaseURL: mustGetEnv("DATABASE_URL"),
		DBTimeout:   getDurationEnv("DB_TIMEOUT_SEC", 5) * time.Second, // 5s padrão

		// 3. Cache (Redis)
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		CacheTimeout: getDurationEnv("CACHE_TIMEOUT_SEC", 10) * time.Second, // 10s padrão

		// 4. Segurança (JWT)
		JWTSecretKey: mustGetEnv("JWT_SECRET_KEY"),
		TokenExpiry:  getDurationEnv("JWT_EXPIRY_MIN", 60) * time.Minute, // 60 min padrão

		// 5. Rate Limiting
		RateLimitMaxRequests: getIntEnv("RATE_LIMIT_MAX_REQUESTS", 100),
		RateLimitPeriod:      getDurationEnv("RATE_LIMIT_PERIOD_MIN", 1) * time.Minute, // 1 min padrão
	}

	return cfg
}

// Funções Helpers (Auxiliares)

// getEnv lê a variável de ambiente ou retorna um valor padrão.
func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// mustGetEnv lê a variável de ambiente, fatal se não estiver presente.
func mustGetEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Fatalf("❌ Erro de Configuração: A variável de ambiente %s deve ser definida.", key)
	return ""
}

// getDurationEnv lê uma variável de ambiente numérica e retorna-a como time.Duration.
func getDurationEnv(key string, defaultValue int) time.Duration {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return time.Duration(defaultValue)
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("⚠️ Aviso: Valor de %s ('%s') não é um número inteiro válido. Usando padrão (%d).", key, valueStr, defaultValue)
		return time.Duration(defaultValue)
	}
	return time.Duration(value)
}

// getIntEnv lê uma variável de ambiente numérica e retorna-a como int.
func getIntEnv(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("⚠️ Aviso: Valor de %s ('%s') não é um número inteiro válido. Usando padrão (%d).", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
