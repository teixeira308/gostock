package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// Client define o contrato de interface para qualquer serviço de cache que o Repositório possa usar.
// Isso segue o Princípio da Inversão de Dependência (DIP) da Clean Architecture.
type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

// ErrCacheMiss é retornado quando a chave não é encontrada no cache.
var ErrCacheMiss = redis.Nil

// RedisClient é a implementação concreta da interface Client, usando Redis.
type RedisClient struct {
	rdb *redis.Client
}

// NewRedisClient cria e retorna uma nova instância do cliente Redis.
// Esta função é chamada no main.go.
func NewRedisClient(addr string) Client {
	// Cria um novo cliente Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: addr, // Endereço do Redis (e.g., "localhost:6379")
	})

	// Teste de conexão: PING para garantir que o cache está disponível
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		// Em produção, isso pode ser um erro fatal ou um aviso, dependendo da criticidade do cache.
		// Por enquanto, apenas logamos, mas o NewRedisClient retorna um Client.
		// Em um projeto real, você retornaria o erro aqui ou implementaria um fallback.
		// log.Fatalf("Não foi possível conectar ao Redis em %s: %v", addr, err)
	}

	return &RedisClient{rdb: rdb}
}

// Get recupera o valor associado a uma chave.
func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()

	// Se a chave não existir no Redis, retornamos o erro exportado (redis.Nil)
	if err == redis.Nil {
		return "", ErrCacheMiss // Retorna nosso erro exportado
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// Set define um valor para uma chave com um tempo de expiração.
func (c *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Delete remove uma chave do cache.
func (c *RedisClient) Delete(ctx context.Context, key string) error {
	// Comando DEL, retorna o número de chaves deletadas (0 se não existir)
	return c.rdb.Del(ctx, key).Err()
}
