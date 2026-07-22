-- +goose Up
CREATE TABLE chirps (
    id uuid PRIMARY KEY,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    body TEXT,
    user_id uuid REFERENCES users(id) ON DELETE CASCADE
)