package warehouseservice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"gostock/internal/domain"
	apperror "gostock/internal/errors"
	"gostock/internal/pkg/logger"
	"gostock/internal/service/warehouseservice"
)

// MockWarehouseRepository é uma implementação mock da interface WarehouseRepository
type MockWarehouseRepository struct {
	mock.Mock
}

func (m *MockWarehouseRepository) CreateWarehouse(ctx context.Context, warehouse domain.Warehouse) (domain.Warehouse, error) {
	args := m.Called(ctx, warehouse)
	return args.Get(0).(domain.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) GetWarehouseByID(ctx context.Context, id string) (domain.Warehouse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) GetAllWarehouses(ctx context.Context) ([]domain.Warehouse, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) UpdateWarehouse(ctx context.Context, warehouse domain.Warehouse) (domain.Warehouse, error) {
	args := m.Called(ctx, warehouse)
	return args.Get(0).(domain.Warehouse), args.Error(1)
}

func (m *MockWarehouseRepository) DeleteWarehouse(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Helper function to create a basic logger
func newTestLogger() logger.Logger {
	return logger.NewLogger("debug") // Or a mock logger if you want to assert logs
}

// --- Testes para CreateWarehouse ---

func TestCreateWarehouse_Success(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	newWarehouse := domain.Warehouse{Name: "Warehouse Alpha"}
	expectedWarehouse := newWarehouse
	expectedWarehouse.ID = uuid.New().String()
	expectedWarehouse.CreatedAt = time.Now()
	expectedWarehouse.UpdatedAt = time.Now()

	mockRepo.On("CreateWarehouse", mock.Anything, newWarehouse).Return(expectedWarehouse, nil)

	ctx := context.Background()
	result, err := svc.CreateWarehouse(ctx, newWarehouse)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedWarehouse.Name, result.Name)
	assert.NotEqual(t, "", result.ID)
	mockRepo.AssertExpectations(t)
}

func TestCreateWarehouse_Fail_InvalidName(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	invalidWarehouse := domain.Warehouse{Name: ""} // Empty name
	ctx := context.Background()
	_, err := svc.CreateWarehouse(ctx, invalidWarehouse)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ValidationError{}, err)
	assert.Contains(t, err.Error(), "não pode ser vazio")
	mockRepo.AssertNotCalled(t, "CreateWarehouse")
}

func TestCreateWarehouse_Fail_RepoError(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	newWarehouse := domain.Warehouse{Name: "Warehouse Beta"}
	repoError := errors.New("database connection failed")

	mockRepo.On("CreateWarehouse", mock.Anything, newWarehouse).Return(domain.Warehouse{}, repoError)

	ctx := context.Background()
	_, err := svc.CreateWarehouse(ctx, newWarehouse)

	assert.Error(t, err)
	assert.IsType(t, &apperror.InternalError{}, err)
	assert.Contains(t, err.Error(), "Falha interna ao criar armazém")
	mockRepo.AssertExpectations(t)
}

// --- Testes para GetWarehouseByID ---

func TestGetWarehouseByID_Success(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseID := uuid.New().String()
	expectedWarehouse := domain.Warehouse{
		ID:   warehouseID,
		Name: "Warehouse Gamma",
	}

	mockRepo.On("GetWarehouseByID", mock.Anything, warehouseID).Return(expectedWarehouse, nil)

	ctx := context.Background()
	result, err := svc.GetWarehouseByID(ctx, warehouseID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedWarehouse.ID, result.ID)
	mockRepo.AssertExpectations(t)
}

func TestGetWarehouseByID_Fail_InvalidID(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	invalidID := "invalid-uuid"
	ctx := context.Background()
	_, err := svc.GetWarehouseByID(ctx, invalidID)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ValidationError{}, err)
	assert.Contains(t, err.Error(), "UUID válido")
	mockRepo.AssertNotCalled(t, "GetWarehouseByID")
}

func TestGetWarehouseByID_Fail_NotFound(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseID := uuid.New().String()
	repoError := apperror.NewNotFoundError("Armazém não encontrado")

	mockRepo.On("GetWarehouseByID", mock.Anything, warehouseID).Return(domain.Warehouse{}, repoError)

	ctx := context.Background()
	_, err := svc.GetWarehouseByID(ctx, warehouseID)

	assert.Error(t, err)
	assert.IsType(t, &apperror.NotFoundError{}, err)
	mockRepo.AssertExpectations(t)
}

func TestGetWarehouseByID_Fail_RepoError(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseID := uuid.New().String()
	repoError := errors.New("database error")

	mockRepo.On("GetWarehouseByID", mock.Anything, warehouseID).Return(domain.Warehouse{}, repoError)

	ctx := context.Background()
	_, err := svc.GetWarehouseByID(ctx, warehouseID)

	assert.Error(t, err)
	// The service simply propagates this error as it is not translated
	assert.Equal(t, repoError, err)
	mockRepo.AssertExpectations(t)
}

// --- Testes para GetAllWarehouses ---

func TestGetAllWarehouses_Success(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	expectedWarehouses := []domain.Warehouse{
		{ID: uuid.New().String(), Name: "W1"},
		{ID: uuid.New().String(), Name: "W2"},
	}

	mockRepo.On("GetAllWarehouses", mock.Anything).Return(expectedWarehouses, nil)

	ctx := context.Background()
	results, err := svc.GetAllWarehouses(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results, 2)
	assert.Equal(t, expectedWarehouses, results)
	mockRepo.AssertExpectations(t)
}

func TestGetAllWarehouses_Fail_RepoError(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	repoError := errors.New("network error")

	mockRepo.On("GetAllWarehouses", mock.Anything).Return([]domain.Warehouse{}, repoError)

	ctx := context.Background()
	_, err := svc.GetAllWarehouses(ctx)

	assert.Error(t, err)
	assert.IsType(t, &apperror.InternalError{}, err)
	assert.Contains(t, err.Error(), "Falha interna ao buscar armazéns")
	mockRepo.AssertExpectations(t)
}

// --- Testes para UpdateWarehouse ---

func TestUpdateWarehouse_Success(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseToUpdate := domain.Warehouse{
		ID:   uuid.New().String(),
		Name: "Updated Warehouse Name",
	}
	expectedUpdatedWarehouse := warehouseToUpdate
	expectedUpdatedWarehouse.UpdatedAt = time.Now()

	mockRepo.On("UpdateWarehouse", mock.Anything, warehouseToUpdate).Return(expectedUpdatedWarehouse, nil)

	ctx := context.Background()
	result, err := svc.UpdateWarehouse(ctx, warehouseToUpdate)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedUpdatedWarehouse.Name, result.Name)
	mockRepo.AssertExpectations(t)
}

func TestUpdateWarehouse_Fail_InvalidID(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	invalidWarehouse := domain.Warehouse{ID: "invalid-uuid", Name: "New Name"}
	ctx := context.Background()
	_, err := svc.UpdateWarehouse(ctx, invalidWarehouse)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ValidationError{}, err)
	assert.Contains(t, err.Error(), "UUID válido")
	mockRepo.AssertNotCalled(t, "UpdateWarehouse")
}

func TestUpdateWarehouse_Fail_InvalidName(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	invalidWarehouse := domain.Warehouse{ID: uuid.New().String(), Name: ""} // Empty name
	ctx := context.Background()
	_, err := svc.UpdateWarehouse(ctx, invalidWarehouse)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ValidationError{}, err)
	assert.Contains(t, err.Error(), "não pode ser vazio")
	mockRepo.AssertNotCalled(t, "UpdateWarehouse")
}

func TestUpdateWarehouse_Fail_NotFound(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseToUpdate := domain.Warehouse{ID: uuid.New().String(), Name: "Non Existent"}
	repoError := apperror.NewNotFoundError("Armazém não encontrado")

	mockRepo.On("UpdateWarehouse", mock.Anything, warehouseToUpdate).Return(domain.Warehouse{}, repoError)

	ctx := context.Background()
	_, err := svc.UpdateWarehouse(ctx, warehouseToUpdate)

	assert.Error(t, err)
	assert.IsType(t, &apperror.NotFoundError{}, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateWarehouse_Fail_RepoError(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseToUpdate := domain.Warehouse{ID: uuid.New().String(), Name: "Repo Error"}
	repoError := errors.New("db timeout")

	mockRepo.On("UpdateWarehouse", mock.Anything, warehouseToUpdate).Return(domain.Warehouse{}, repoError)

	ctx := context.Background()
	_, err := svc.UpdateWarehouse(ctx, warehouseToUpdate)

	assert.Error(t, err)
	assert.Equal(t, repoError, err)
	mockRepo.AssertExpectations(t)
}

// --- Testes para DeleteWarehouse ---

func TestDeleteWarehouse_Success(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseID := uuid.New().String()
	mockRepo.On("DeleteWarehouse", mock.Anything, warehouseID).Return(nil)

	ctx := context.Background()
	err := svc.DeleteWarehouse(ctx, warehouseID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteWarehouse_Fail_InvalidID(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	invalidID := "invalid-uuid"
	ctx := context.Background()
	err := svc.DeleteWarehouse(ctx, invalidID)

	assert.Error(t, err)
	assert.IsType(t, &apperror.ValidationError{}, err)
	assert.Contains(t, err.Error(), "UUID válido")
	mockRepo.AssertNotCalled(t, "DeleteWarehouse")
}

func TestDeleteWarehouse_Fail_NotFound(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseID := uuid.New().String()
	repoError := apperror.NewNotFoundError("Armazém não encontrado")

	mockRepo.On("DeleteWarehouse", mock.Anything, warehouseID).Return(repoError)

	ctx := context.Background()
	err := svc.DeleteWarehouse(ctx, warehouseID)

	assert.Error(t, err)
	assert.IsType(t, &apperror.NotFoundError{}, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteWarehouse_Fail_RepoError(t *testing.T) {
	mockRepo := new(MockWarehouseRepository)
	svc := warehouseservice.NewService(mockRepo, newTestLogger())

	warehouseID := uuid.New().String()
	repoError := errors.New("db error")

	mockRepo.On("DeleteWarehouse", mock.Anything, warehouseID).Return(repoError)

	ctx := context.Background()
	err := svc.DeleteWarehouse(ctx, warehouseID)

	assert.Error(t, err)
	assert.Equal(t, repoError, err)
	mockRepo.AssertExpectations(t)
}
