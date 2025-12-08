package userservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/token"
)

// UserService define o serviço de lógica de negócio para a entidade User.
type UserService struct {
	UserRepo domain.UserRepository
	TokenSvc TokenService
}

// TokenService é o contrato da camada de token (internal/pkg/token)
type TokenService interface {
	GenerateToken(userID string, userRole string) (string, error)
	ValidateToken(tokenString string) (*token.CustomClaims, error) // Assumindo importação correta
}

// NewService cria uma nova instância do UserService, injetando o Repositório.
func NewService(repo domain.UserRepository, tokenSvc TokenService) *UserService {
	return &UserService{
		UserRepo: repo,
		TokenSvc: tokenSvc, // Salva o serviço injetado
	}
}

// Register registra um novo usuário no sistema.
// Ele faz o hashing da senha e lida com validações básicas.
func (s *UserService) Register(ctx context.Context, registration domain.UserRegistration) (domain.User, error) {
	// 1. Validação Básica (Simplificada)
	if registration.Email == "" || registration.Password == "" {
		return domain.User{}, apperror.NewValidationError("Email e senha são obrigatórios.")
	}

	// 2. Hashing da Senha
	// Gera um hash forte para a senha informada.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registration.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, apperror.NewInternalError("Falha ao gerar hash da senha.", err)
	}

	// 3. Criação do Objeto User
	newUser := domain.User{
		Email:        registration.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user", // Define o role padrão
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 4. Chamada ao Repositório para Persistência
	user, err := s.UserRepo.Save(ctx, newUser)

	if err != nil {
		// Verifica se o erro do DB é de violação de unicidade (e-mail duplicado)
		// Em um ambiente de produção, faríamos uma verificação mais específica
		// para o código de erro do PostgreSQL. Por simplicidade, assumimos o DBError.

		// Se for um erro de DB (possivelmente e-mail duplicado), o traduzimos
		// para um erro de Conflito de Negócio (409 Conflict).
		var dbErr *apperror.InternalError
		if errors.As(err, &dbErr) {
			// Assumimos que o DBError é o resultado de uma chave única violada (e-mail)
			return domain.User{}, apperror.NewConflictError(
				fmt.Sprintf("O email '%s' já está em uso.", registration.Email),
			)
		}

		// Retorna o erro original (ex: 500 Interno, timeout)
		return domain.User{}, err
	}

	return user, nil
}

// Login autentica um usuário, verifica a senha e gera um JWT.
func (s *UserService) Login(ctx context.Context, email string, password string) (string, error) {
	// 1. Validação Básica
	if email == "" || password == "" {
		// Usamos UnauthorizedError para login/senha inválidos
		return "", apperror.NewUnauthorizedError("Email e senha são obrigatórios.")
	}

	// 2. Buscar Usuário pelo Email
	user, err := s.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		// Se for um NotFoundError (404), tratamos como Unauthorized (401) para não dar dicas a invasores.
		var notFoundErr *apperror.NotFoundError
		if errors.As(err, &notFoundErr) {
			return "", apperror.NewUnauthorizedError("Credenciais inválidas.")
		}
		// Retorna erro interno se falhar a busca (DB error)
		return "", err
	}

	// 3. Comparar Senhas (Hashing)
	// Compara a senha informada (texto puro) com o hash salvo no DB.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// Se a comparação falhar (senha incorreta)
		return "", apperror.NewUnauthorizedError("Credenciais inválidas.")
	}

	// 4. Gerar JWT
	// Se a senha estiver correta, geramos o token JWT
	tokenString, err := s.TokenSvc.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		// Log interno de erro de geração de token
		return "", apperror.NewInternalError("Falha ao gerar token de autenticação.", err)
	}

	// 5. Sucesso
	return tokenString, nil
}
