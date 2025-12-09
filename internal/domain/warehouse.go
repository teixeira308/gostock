package domain

import (
	"time"
)

// Warehouse representa um armazém físico ou lógico no sistema.
type Warehouse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
