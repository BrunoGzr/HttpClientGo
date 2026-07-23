-- name: ChirpsFindById :one
SELECT * FROM chirps WHERE id = $1;
