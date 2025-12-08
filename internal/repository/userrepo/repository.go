package userrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
)

// UserRepository implementa a interface domain.UserRepository
type UserRepository struct {
	DB          *sql.DB
	DBTimeout   time.Duration
	ProductSQLs struct { // Pode ser removido se não for usado, mas aqui mantemos o padrão
		Insert string
	}
	logger      logger.Logger
}

// NewUserRepository cria uma nova instância do UserRepository, injetando o DB.
func NewUserRepository(db *sql.DB, dbTimeout time.Duration, logger logger.Logger) *UserRepository {
	// Definimos a query SQL para inserção de usuário
	insertSQL := `INSERT INTO users (id, email, password_hash, role, created_at, updated_at) 
                  VALUES ($1, $2, $3, $4, $5, $6)`

	return &UserRepository{
		DB:        db,
		DBTimeout: dbTimeout,
		ProductSQLs: struct {
			Insert string
		}{
			Insert: insertSQL,
		},
		logger:    logger,
	}
}

// Save insere um novo usuário no banco de dados.
func (r *UserRepository) Save(ctx domain.Context, user domain.User) (domain.User, error) {
	r.logger.Debug("Iniciando Save de usuário no repositório.", map[string]interface{}{"email": user.Email})

	// 1. Configura Contexto com Timeout
	ctxTimeout, cancel := context.WithTimeout(ctx.(context.Context), r.DBTimeout)
	defer cancel()

	// 2. Prepara dados e ID
	user.ID = uuid.NewString()
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	r.logger.Debug("Gerado novo ID e timestamps para o usuário.", map[string]interface{}{"user_id": user.ID, "email": user.Email})

	// 3. Executa o INSERT
	_, err := r.DB.ExecContext(
		ctxTimeout,
		r.ProductSQLs.Insert,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("Falha ao inserir usuário no DB.", err)
		// Verifica se é um erro de duplicidade (ex: email já existe)
		// No PostgreSQL, isso exige verificar o erro específico do driver (pq)
		// Por enquanto, simplificamos como um erro interno de DB
		return domain.User{}, apperror.NewDBError("failed to insert user (DB)", err)
	}

	r.logger.Info("Usuário salvo com sucesso no repositório.", map[string]interface{}{"user_id": user.ID, "email": user.Email})
	return user, nil
}

// FindByEmail busca um usuário pelo endereço de e-mail.
func (r *UserRepository) FindByEmail(ctx domain.Context, email string) (domain.User, error) {
	r.logger.Debug("Iniciando FindByEmail de usuário no repositório.", map[string]interface{}{"email_attempt": email})

	// 1. Configura Contexto com Timeout
	ctxTimeout, cancel := context.WithTimeout(ctx.(context.Context), r.DBTimeout)
	defer cancel()

	// 2. Define a query SQL
	query := `SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = $1`
	r.logger.Debug("Executando query FindByEmail.", map[string]interface{}{"email": email})

	// 3. Executa a busca
	row := r.DB.QueryRowContext(ctxTimeout, query, email)

	var user domain.User

	// 4. Mapeia o resultado para a struct User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Info("Usuário não encontrado no DB por email.", map[string]interface{}{"email": email})
			// Retorna um erro tipado 404
			return domain.User{}, apperror.NewNotFoundError(fmt.Sprintf("Usuário com email '%s' não encontrado", email))
		}
		r.logger.Error("Falha ao buscar usuário por email no DB.", err)
		// Retorna um erro interno de DB para qualquer outra falha de SQL
		return domain.User{}, apperror.NewDBError("failed to find user by email (DB)", err)
	}

	r.logger.Info("Usuário encontrado no repositório por email.", map[string]interface{}{"user_id": user.ID, "email": user.Email})
	return user, nil
}
