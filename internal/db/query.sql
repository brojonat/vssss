-- name: InsertSignal :one
INSERT INTO signals (path, description, type, datatype, unit)
VALUES (?, ?, ?, ?, ?)
RETURNING id;

-- name: GetSignalByID :one
SELECT id, path, description, type, datatype, unit
FROM signals
WHERE id = ?;

-- name: GetSignalByPath :one
SELECT id, path, description, type, datatype, unit
FROM signals
WHERE path = ?;

-- name: GetSignalsByIDs :many
SELECT id, path, description, type, datatype, unit
FROM signals
WHERE id IN (sqlc.slice('ids'));

-- name: CountSignals :one
SELECT COUNT(*) FROM signals;
