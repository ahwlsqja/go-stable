-- name: CreateProduct :execresult
INSERT INTO products (sku, name, price, status)
VALUES (?, ?, ?, ?);

-- name: GetProduct :one
SELECT * FROM products WHERE id = ? LIMIT 1;

-- name: GetProductBySKU :one
SELECT * FROM products WHERE sku = ? LIMIT 1;

-- name: ListProducts :many
SELECT * FROM products
WHERE status = COALESCE(sqlc.narg('status'), status)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateProduct :exec
UPDATE products
SET name = ?, price = ?, status = ?
WHERE id = ?;

-- name: DeleteProduct :exec
UPDATE products SET status = 'INACTIVE' WHERE id = ?;
