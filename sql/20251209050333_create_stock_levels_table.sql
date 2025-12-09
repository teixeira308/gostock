-- +goose Up
CREATE TABLE stock_levels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    variant_id UUID NOT NULL,
    warehouse_id UUID NOT NULL,
    quantity INT NOT NULL CHECK (quantity >= 0),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_variant_warehouse UNIQUE (variant_id, warehouse_id)
);

-- +goose Down
DROP TABLE stock_levels;
