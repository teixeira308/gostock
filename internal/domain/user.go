package domain

import "time"

// User representa a entidade do usuário no sistema.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Oculta o hash da senha no JSON de resposta
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserRole é um tipo string para representar o papel do usuário no sistema.
type UserRole string

// Constantes para os papéis de usuário (boas práticas)
const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
	RoleGuest UserRole = "guest"
)

// UserRegistration representa o payload de entrada para o registro.
type UserRegistration struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserRepository define o contrato de persistência para a entidade User.
type UserRepository interface {
	Save(ctx Context, user User) (User, error)
	FindByEmail(ctx Context, email string) (User, error)
}

// UserService define o contrato de lógica de negócio para a entidade User.
type UserService interface {
	Register(ctx Context, registration UserRegistration) (User, error)
	Login(ctx Context, email string, password string) (string, error)
}
