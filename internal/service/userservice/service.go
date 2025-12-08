package userservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
	"gostock/internal/pkg/token"
)

// UserService define o serviço de lógica de negócio para a entidade User.
type UserService struct {
	UserRepo domain.UserRepository
	TokenSvc TokenService
	logger   logger.Logger
}

// TokenService é o contrato da camada de token (internal/pkg/token)
type TokenService interface {
	GenerateToken(userID string, userRole string) (string, error)
	ValidateToken(tokenString string) (*token.CustomClaims, error) // Assumindo importação correta
}

// NewService cria uma nova instância do UserService, injetando o Repositório.
func NewService(repo domain.UserRepository, tokenSvc TokenService, logger logger.Logger) *UserService {
	return &UserService{
		UserRepo: repo,
		TokenSvc: tokenSvc, // Salva o serviço injetado
		logger:   logger,
	}
}

// Register registra um novo usuário no sistema.
// Ele faz o hashing da senha e lida com validações básicas.
func (s *UserService) Register(ctx context.Context, registration domain.UserRegistration) (domain.User, error) {
	s.logger.Debug("Iniciando registro de novo usuário.", map[string]interface{}{"email": registration.Email})

	// 1. Validação Básica (Simplificada)
	if registration.Email == "" || registration.Password == "" {
		s.logger.Warn("Tentativa de registro com email ou senha vazios.", nil)
		return domain.User{}, apperror.NewValidationError("Email e senha são obrigatórios.")
	}

	// 2. Hashing da Senha
	// Gera um hash forte para a senha informada.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registration.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Falha ao gerar hash da senha.", err)
		return domain.User{}, apperror.NewInternalError("Falha ao gerar hash da senha.", err)
	}
	s.logger.Debug("Senha hash gerada com sucesso.", map[string]interface{}{"email": registration.Email})

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
		var dbErr *apperror.InternalError
		if errors.As(err, &dbErr) {
			s.logger.Warn("Tentativa de registro com email duplicado.", map[string]interface{}{"email": registration.Email, "error": err.Error()})
			return domain.User{}, apperror.NewConflictError(
				fmt.Sprintf("O email '%s' já está em uso.", registration.Email),
			)
		}
		s.logger.Error("Erro ao salvar usuário no repositório durante o registro.", err)
		return domain.User{}, err
	}

	s.logger.Info("Usuário registrado com sucesso.", map[string]interface{}{"user_id": user.ID, "email": user.Email})
	return user, nil
}

// Login autentica um usuário, verifica a senha e gera um JWT.
func (s *UserService) Login(ctx context.Context, email string, password string) (string, error) {
	s.logger.Debug("Iniciando tentativa de login.", map[string]interface{}{"email_attempt": email})

	// 1. Validação Básica
	if email == "" || password == "" {
		s.logger.Warn("Tentativa de login com email ou senha vazios.", nil)
		// Usamos UnauthorizedError para login/senha inválidos
		return "", apperror.NewUnauthorizedError("Email e senha são obrigatórios.")
	}

	// 2. Buscar Usuário pelo Email
	user, err := s.UserRepo.FindByEmail(ctx, email)
	if err != nil {
		var notFoundErr *apperror.NotFoundError
		if errors.As(err, &notFoundErr) {
			s.logger.Info("Tentativa de login com email não encontrado.", map[string]interface{}{"email": email})
			return "", apperror.NewUnauthorizedError("Credenciais inválidas.")
		}
		s.logger.Error("Erro interno ao buscar usuário por email.", err)
		// Retorna erro interno se falhar a busca (DB error)
		return "", err
	}
	s.logger.Debug("Usuário encontrado.", map[string]interface{}{"user_id": user.ID, "email": user.Email})

	// 3. Comparar Senhas (Hashing)
	// Compara a senha informada (texto puro) com o hash salvo no DB.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.logger.Warn("Tentativa de login com senha incorreta.", map[string]interface{}{"email": email})
		// Se a comparação falhar (senha incorreta)
		return "", apperror.NewUnauthorizedError("Credenciais inválidas.")
	}
	s.logger.Debug("Senha verificada com sucesso.", map[string]interface{}{"email": email})

	// 4. Gerar JWT
	// Se a senha estiver correta, geramos o token JWT
	tokenString, err := s.TokenSvc.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		s.logger.Error("Falha ao gerar token de autenticação.", err)
		// Log interno de erro de geração de token
		return "", apperror.NewInternalError("Falha ao gerar token de autenticação.", err)
	}
	s.logger.Info("Token JWT gerado com sucesso para o usuário.", map[string]interface{}{"user_id": user.ID})

	// 5. Sucesso
	return tokenString, nil
}

// validateProduct verifica as regras de negócio básicas do produto e suas variações.
func (s *UserService) validateProduct(p domain.Product) error {
	if p.SKU == "" {
		return apperror.NewValidationError("O SKU do produto é obrigatório.")
	}
	if p.Name == "" {
		return apperror.NewValidationError("O nome do produto é obrigatório.")
	}
	if p.Price <= 0 {
		return apperror.NewValidationError("O preço do produto deve ser um valor positivo.")
	}

	// Validação das Variações
	if len(p.Variants) == 0 {
		return apperror.NewValidationError("O produto deve ter pelo menos uma variação.")
	}

	for i, v := range p.Variants {
		if v.Attribute == "" || v.Value == "" {
			return apperror.NewValidationError(fmt.Sprintf("Atributo ou valor da variação %d está vazio.", i+1))
		}
		if v.PriceDiff < 0 {
			return apperror.NewValidationError(fmt.Sprintf("A diferença de preço da variação %d não pode ser negativa.", i+1))
		}
		if v.Barcode == "" {
			return apperror.NewValidationError(fmt.Sprintf("O código de barras da variação %d é obrigatório.", i+1))
		}
	}

	return nil
}
