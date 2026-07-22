-- +goose Up
CREATE TABLE users
(
    id         uuid PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    email      TEXT,
    hashed_password TEXT NOT NULL DEFAULT 'unset'
);

SELECT * FROM users;
-- +goose Down
DROP TABLE users;