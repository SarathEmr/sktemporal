-- Connect to appdb and create tables
\c appdb

-- Note: All timestamps use TIMESTAMPTZ (TIMESTAMP WITH TIME ZONE)
-- Application should always pass time in UTC

-- Create enum type for order status
CREATE TYPE order_status AS ENUM (
    'ADDED_TO_CART',
    'PAYMENT_FAILED',
    'SHIPPING_INITIATED',
    'ORDER_DELIVERED'
);

-- Create enum type for product type
CREATE TYPE product_type AS ENUM (
    'ELECTRONICS',
    'CLOTHS',
    'KITCHEN_ITEMS'
);

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    userID UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    products JSONB NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL,
    status order_status NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    address TEXT
);

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type product_type NOT NULL,
    company VARCHAR(255) NOT NULL,
    items_available INTEGER NOT NULL CHECK (items_available >= 0),
    price DECIMAL(10, 2) NOT NULL
);

-- Create function to update updated_at timestamp
-- Uses CURRENT_TIMESTAMP which returns TIMESTAMPTZ
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to automatically update updated_at on order table
CREATE TRIGGER update_order_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create trigger to automatically update updated_at on user table
CREATE TRIGGER update_user_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_orders_userID ON orders(userID);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_orders_updated_at ON orders(updated_at);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_products_type ON products(type);
CREATE INDEX IF NOT EXISTS idx_products_company ON products(company);
