-- Connect to appdb and seed initial data
\c appdb

-- Seed user data
INSERT INTO users (id, name, email, address) VALUES
    ('550e8400-e29b-41d4-a716-446655440000', 'Akshai', 'akshai@gmail.com', '123 Main St, Anytown, USA')
ON CONFLICT (email) DO NOTHING;

-- Seed product data
INSERT INTO products (id, name, type, company, items_available, price) VALUES
    ('660e8400-e29b-41d4-a716-446655440001', 'iPhone 15', 'ELECTRONICS', 'Apple', 10, 120000.00)
ON CONFLICT (id) DO NOTHING;
