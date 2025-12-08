package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	// Usamos o driver pq para PostgreSQL
	// No seu projeto real, você precisará: go get github.com/lib/pq
	_ "github.com/lib/pq"
)

// NewPostgresDB inicializa e configura o pool de conexões com o PostgreSQL.
// Retorna a conexão *sql.DB pronta para uso.
func NewPostgresDB(dataSourceName string) (*sql.DB, error) {

	// 1. Abrir a Conexão (Sem tentar ainda usar o pool)
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		// Falha ao abrir a conexão (erro de driver, formato da DSN, etc.)
		return nil, fmt.Errorf("falha ao abrir a conexão com o DB: %w", err)
	}

	// 2. Testar a Conexão Imediatamente
	// Garante que as credenciais e o servidor estão corretos
	err = db.Ping()
	if err != nil {
		// Falha no ping (DB inacessível, credenciais erradas)
		db.Close() // Fecha a conexão aberta se falhar
		return nil, fmt.Errorf("falha ao realizar o ping inicial no DB: %w", err)
	}

	// 3. Configuração do Connection Pool (Crucial para Performance e Escalabilidade)
	// (Módulo do Curso: Configuring the DB Connection Pool)

	// MaxOpenConns: Número máximo de conexões abertas com o banco de dados.
	// Deve ser ajustado ao limite do seu servidor DB e ao tráfego esperado.
	db.SetMaxOpenConns(25) // Exemplo: 25 conexões

	// MaxIdleConns: Número máximo de conexões ociosas no pool.
	// Se for muito baixo, o Go precisará criar e destruir conexões frequentemente.
	db.SetMaxIdleConns(10) // Exemplo: 10 conexões ociosas

	// ConnMaxLifetime: Tempo máximo de vida de uma conexão (evita problemas de rede/firewall).
	db.SetConnMaxLifetime(5 * time.Minute) // Conexões morrem após 5 minutos

	// ConnMaxIdleTime: Tempo máximo que uma conexão pode ficar ociosa antes de ser fechada.
	db.SetConnMaxIdleTime(2 * time.Minute) // Conexões ociosas morrem após 2 minutos

	log.Println("✅ Pool de Conexões PostgreSQL configurado e pronto.")

	return db, nil
}
