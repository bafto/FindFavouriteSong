-- name: AddSession :one
INSERT INTO session
(id, playlist, current_round, user, winner, creation_timestamp) VALUES (NULL, ?, 0, ?, NULL, CURRENT_TIMESTAMP)
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
(id, session, round_number, winner, loser, creation_timestamp) VALUES (NULL, ?, ?, ?, ?, CURRENT_TIMESTAMP);

-- name: CountMatchesForRound :one
SELECT COUNT(*) FROM match
WHERE session = ? AND round_number = ?;

-- name: GetSession :one
SELECT * FROM session
WHERE id = ?;

-- name: GetNumberOfMatchesCompleted :one
SELECT COUNT(*) FROM match
WHERE session = ?;

-- name: DeleteSession :exec
DELETE FROM session WHERE id = ?;

-- name: DeleteMatchesForSession :exec
DELETE FROM match WHERE session = ?;
