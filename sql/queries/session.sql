-- name: AddSession :one
INSERT INTO session
(id, playlist, current_round, user, winner) VALUES (NULL, ?, 0, ?, NULL)
RETURNING session.id;

-- name: GetWinner :one
SELECT winner FROM session
WHERE id = ?;

-- name: SetWinner :exec
UPDATE session
SET winner = ?
WHERE id = ?;

-- name: GetCurrentRound :one
SELECT current_round FROM session
WHERE id = ?;

-- name: SetCurrentRound :exec
UPDATE session
SET current_round = ?
WHERE id = ?;

-- name: AddMatch :exec
INSERT INTO match
(id, session, round_number, winner, loser) VALUES (NULL, ?, ?, ?, ?);

