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
)

// UserRepository implementa a interface domain.UserRepository
type UserRepository struct {
	DB          *sql.DB
	DBTimeout   time.Duration
	ProductSQLs struct { // Pode ser removido se não for usado, mas aqui mantemos o padrão
		Insert string
	}
}

// NewUserRepository cria uma nova instância do UserRepository, injetando o DB.
func NewUserRepository(db *sql.DB, dbTimeout time.Duration) *UserRepository {
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
	}
}

// Save insere um novo usuário no banco de dados.
func (r *UserRepository) Save(ctx domain.Context, user domain.User) (domain.User, error) {
	// 1. Configura Contexto com Timeout
	ctxTimeout, cancel := context.WithTimeout(ctx.(context.Context), r.DBTimeout)
	defer cancel()

	// 2. Prepara dados e ID
	user.ID = uuid.NewString()
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt

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
		// Verifica se é um erro de duplicidade (ex: email já existe)
		// No PostgreSQL, isso exige verificar o erro específico do driver (pq)
		// Por enquanto, simplificamos como um erro interno de DB
		return domain.User{}, apperror.NewDBError("failed to insert user (DB)", err)
	}

	return user, nil
}

// FindByEmail busca um usuário pelo endereço de e-mail.
func (r *UserRepository) FindByEmail(ctx domain.Context, email string) (domain.User, error) {
	// 1. Configura Contexto com Timeout
	ctxTimeout, cancel := context.WithTimeout(ctx.(context.Context), r.DBTimeout)
	defer cancel()

	// 2. Define a query SQL
	query := `SELECT id, email, password_hash, role, created_at, updated_at FROM users WHERE email = $1`

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
			// Retorna um erro tipado 404
			return domain.User{}, apperror.NewNotFoundError(fmt.Sprintf("Usuário com email '%s' não encontrado", email))
		}
		// Retorna um erro interno de DB para qualquer outra falha de SQL
		return domain.User{}, apperror.NewDBError("failed to find user by email (DB)", err)
	}

	return user, nil
}
