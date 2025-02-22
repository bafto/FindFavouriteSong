// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package db

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.addMatchStmt, err = db.PrepareContext(ctx, addMatch); err != nil {
		return nil, fmt.Errorf("error preparing query AddMatch: %w", err)
	}
	if q.addOrUpdatePlaylistStmt, err = db.PrepareContext(ctx, addOrUpdatePlaylist); err != nil {
		return nil, fmt.Errorf("error preparing query AddOrUpdatePlaylist: %w", err)
	}
	if q.addOrUpdatePlaylistItemStmt, err = db.PrepareContext(ctx, addOrUpdatePlaylistItem); err != nil {
		return nil, fmt.Errorf("error preparing query AddOrUpdatePlaylistItem: %w", err)
	}
	if q.addPlaylistAddedByUserStmt, err = db.PrepareContext(ctx, addPlaylistAddedByUser); err != nil {
		return nil, fmt.Errorf("error preparing query AddPlaylistAddedByUser: %w", err)
	}
	if q.addPlaylistItemBelongsToPlaylistStmt, err = db.PrepareContext(ctx, addPlaylistItemBelongsToPlaylist); err != nil {
		return nil, fmt.Errorf("error preparing query AddPlaylistItemBelongsToPlaylist: %w", err)
	}
	if q.addSessionStmt, err = db.PrepareContext(ctx, addSession); err != nil {
		return nil, fmt.Errorf("error preparing query AddSession: %w", err)
	}
	if q.addUserStmt, err = db.PrepareContext(ctx, addUser); err != nil {
		return nil, fmt.Errorf("error preparing query AddUser: %w", err)
	}
	if q.countMatchesForRoundStmt, err = db.PrepareContext(ctx, countMatchesForRound); err != nil {
		return nil, fmt.Errorf("error preparing query CountMatchesForRound: %w", err)
	}
	if q.deleteItemFromPlaylistStmt, err = db.PrepareContext(ctx, deleteItemFromPlaylist); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteItemFromPlaylist: %w", err)
	}
	if q.deleteMatchesForSessionStmt, err = db.PrepareContext(ctx, deleteMatchesForSession); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteMatchesForSession: %w", err)
	}
	if q.deletePossibleNextItemsForSessionStmt, err = db.PrepareContext(ctx, deletePossibleNextItemsForSession); err != nil {
		return nil, fmt.Errorf("error preparing query DeletePossibleNextItemsForSession: %w", err)
	}
	if q.deleteSessionStmt, err = db.PrepareContext(ctx, deleteSession); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteSession: %w", err)
	}
	if q.getAllWinnersForUserStmt, err = db.PrepareContext(ctx, getAllWinnersForUser); err != nil {
		return nil, fmt.Errorf("error preparing query GetAllWinnersForUser: %w", err)
	}
	if q.getCurrentRoundStmt, err = db.PrepareContext(ctx, getCurrentRound); err != nil {
		return nil, fmt.Errorf("error preparing query GetCurrentRound: %w", err)
	}
	if q.getItemIdsForPlaylistStmt, err = db.PrepareContext(ctx, getItemIdsForPlaylist); err != nil {
		return nil, fmt.Errorf("error preparing query GetItemIdsForPlaylist: %w", err)
	}
	if q.getNextPairStmt, err = db.PrepareContext(ctx, getNextPair); err != nil {
		return nil, fmt.Errorf("error preparing query GetNextPair: %w", err)
	}
	if q.getNonActiveUserSessionsStmt, err = db.PrepareContext(ctx, getNonActiveUserSessions); err != nil {
		return nil, fmt.Errorf("error preparing query GetNonActiveUserSessions: %w", err)
	}
	if q.getNumberOfMatchesCompletedStmt, err = db.PrepareContext(ctx, getNumberOfMatchesCompleted); err != nil {
		return nil, fmt.Errorf("error preparing query GetNumberOfMatchesCompleted: %w", err)
	}
	if q.getPlaylistStmt, err = db.PrepareContext(ctx, getPlaylist); err != nil {
		return nil, fmt.Errorf("error preparing query GetPlaylist: %w", err)
	}
	if q.getPlaylistItemStmt, err = db.PrepareContext(ctx, getPlaylistItem); err != nil {
		return nil, fmt.Errorf("error preparing query GetPlaylistItem: %w", err)
	}
	if q.getPlaylistsForUserStmt, err = db.PrepareContext(ctx, getPlaylistsForUser); err != nil {
		return nil, fmt.Errorf("error preparing query GetPlaylistsForUser: %w", err)
	}
	if q.getSessionStmt, err = db.PrepareContext(ctx, getSession); err != nil {
		return nil, fmt.Errorf("error preparing query GetSession: %w", err)
	}
	if q.getStatistics1Stmt, err = db.PrepareContext(ctx, getStatistics1); err != nil {
		return nil, fmt.Errorf("error preparing query GetStatistics1: %w", err)
	}
	if q.getUserStmt, err = db.PrepareContext(ctx, getUser); err != nil {
		return nil, fmt.Errorf("error preparing query GetUser: %w", err)
	}
	if q.getWinnerStmt, err = db.PrepareContext(ctx, getWinner); err != nil {
		return nil, fmt.Errorf("error preparing query GetWinner: %w", err)
	}
	if q.initializePossibleNextItemsForSessionStmt, err = db.PrepareContext(ctx, initializePossibleNextItemsForSession); err != nil {
		return nil, fmt.Errorf("error preparing query InitializePossibleNextItemsForSession: %w", err)
	}
	if q.setCurrentRoundStmt, err = db.PrepareContext(ctx, setCurrentRound); err != nil {
		return nil, fmt.Errorf("error preparing query SetCurrentRound: %w", err)
	}
	if q.setUserSessionStmt, err = db.PrepareContext(ctx, setUserSession); err != nil {
		return nil, fmt.Errorf("error preparing query SetUserSession: %w", err)
	}
	if q.setWinnerStmt, err = db.PrepareContext(ctx, setWinner); err != nil {
		return nil, fmt.Errorf("error preparing query SetWinner: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.addMatchStmt != nil {
		if cerr := q.addMatchStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addMatchStmt: %w", cerr)
		}
	}
	if q.addOrUpdatePlaylistStmt != nil {
		if cerr := q.addOrUpdatePlaylistStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addOrUpdatePlaylistStmt: %w", cerr)
		}
	}
	if q.addOrUpdatePlaylistItemStmt != nil {
		if cerr := q.addOrUpdatePlaylistItemStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addOrUpdatePlaylistItemStmt: %w", cerr)
		}
	}
	if q.addPlaylistAddedByUserStmt != nil {
		if cerr := q.addPlaylistAddedByUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addPlaylistAddedByUserStmt: %w", cerr)
		}
	}
	if q.addPlaylistItemBelongsToPlaylistStmt != nil {
		if cerr := q.addPlaylistItemBelongsToPlaylistStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addPlaylistItemBelongsToPlaylistStmt: %w", cerr)
		}
	}
	if q.addSessionStmt != nil {
		if cerr := q.addSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addSessionStmt: %w", cerr)
		}
	}
	if q.addUserStmt != nil {
		if cerr := q.addUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing addUserStmt: %w", cerr)
		}
	}
	if q.countMatchesForRoundStmt != nil {
		if cerr := q.countMatchesForRoundStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countMatchesForRoundStmt: %w", cerr)
		}
	}
	if q.deleteItemFromPlaylistStmt != nil {
		if cerr := q.deleteItemFromPlaylistStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteItemFromPlaylistStmt: %w", cerr)
		}
	}
	if q.deleteMatchesForSessionStmt != nil {
		if cerr := q.deleteMatchesForSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteMatchesForSessionStmt: %w", cerr)
		}
	}
	if q.deletePossibleNextItemsForSessionStmt != nil {
		if cerr := q.deletePossibleNextItemsForSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deletePossibleNextItemsForSessionStmt: %w", cerr)
		}
	}
	if q.deleteSessionStmt != nil {
		if cerr := q.deleteSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteSessionStmt: %w", cerr)
		}
	}
	if q.getAllWinnersForUserStmt != nil {
		if cerr := q.getAllWinnersForUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getAllWinnersForUserStmt: %w", cerr)
		}
	}
	if q.getCurrentRoundStmt != nil {
		if cerr := q.getCurrentRoundStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getCurrentRoundStmt: %w", cerr)
		}
	}
	if q.getItemIdsForPlaylistStmt != nil {
		if cerr := q.getItemIdsForPlaylistStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getItemIdsForPlaylistStmt: %w", cerr)
		}
	}
	if q.getNextPairStmt != nil {
		if cerr := q.getNextPairStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNextPairStmt: %w", cerr)
		}
	}
	if q.getNonActiveUserSessionsStmt != nil {
		if cerr := q.getNonActiveUserSessionsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNonActiveUserSessionsStmt: %w", cerr)
		}
	}
	if q.getNumberOfMatchesCompletedStmt != nil {
		if cerr := q.getNumberOfMatchesCompletedStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNumberOfMatchesCompletedStmt: %w", cerr)
		}
	}
	if q.getPlaylistStmt != nil {
		if cerr := q.getPlaylistStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPlaylistStmt: %w", cerr)
		}
	}
	if q.getPlaylistItemStmt != nil {
		if cerr := q.getPlaylistItemStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPlaylistItemStmt: %w", cerr)
		}
	}
	if q.getPlaylistsForUserStmt != nil {
		if cerr := q.getPlaylistsForUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPlaylistsForUserStmt: %w", cerr)
		}
	}
	if q.getSessionStmt != nil {
		if cerr := q.getSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getSessionStmt: %w", cerr)
		}
	}
	if q.getStatistics1Stmt != nil {
		if cerr := q.getStatistics1Stmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getStatistics1Stmt: %w", cerr)
		}
	}
	if q.getUserStmt != nil {
		if cerr := q.getUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getUserStmt: %w", cerr)
		}
	}
	if q.getWinnerStmt != nil {
		if cerr := q.getWinnerStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getWinnerStmt: %w", cerr)
		}
	}
	if q.initializePossibleNextItemsForSessionStmt != nil {
		if cerr := q.initializePossibleNextItemsForSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing initializePossibleNextItemsForSessionStmt: %w", cerr)
		}
	}
	if q.setCurrentRoundStmt != nil {
		if cerr := q.setCurrentRoundStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setCurrentRoundStmt: %w", cerr)
		}
	}
	if q.setUserSessionStmt != nil {
		if cerr := q.setUserSessionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setUserSessionStmt: %w", cerr)
		}
	}
	if q.setWinnerStmt != nil {
		if cerr := q.setWinnerStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing setWinnerStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                                        DBTX
	tx                                        *sql.Tx
	addMatchStmt                              *sql.Stmt
	addOrUpdatePlaylistStmt                   *sql.Stmt
	addOrUpdatePlaylistItemStmt               *sql.Stmt
	addPlaylistAddedByUserStmt                *sql.Stmt
	addPlaylistItemBelongsToPlaylistStmt      *sql.Stmt
	addSessionStmt                            *sql.Stmt
	addUserStmt                               *sql.Stmt
	countMatchesForRoundStmt                  *sql.Stmt
	deleteItemFromPlaylistStmt                *sql.Stmt
	deleteMatchesForSessionStmt               *sql.Stmt
	deletePossibleNextItemsForSessionStmt     *sql.Stmt
	deleteSessionStmt                         *sql.Stmt
	getAllWinnersForUserStmt                  *sql.Stmt
	getCurrentRoundStmt                       *sql.Stmt
	getItemIdsForPlaylistStmt                 *sql.Stmt
	getNextPairStmt                           *sql.Stmt
	getNonActiveUserSessionsStmt              *sql.Stmt
	getNumberOfMatchesCompletedStmt           *sql.Stmt
	getPlaylistStmt                           *sql.Stmt
	getPlaylistItemStmt                       *sql.Stmt
	getPlaylistsForUserStmt                   *sql.Stmt
	getSessionStmt                            *sql.Stmt
	getStatistics1Stmt                        *sql.Stmt
	getUserStmt                               *sql.Stmt
	getWinnerStmt                             *sql.Stmt
	initializePossibleNextItemsForSessionStmt *sql.Stmt
	setCurrentRoundStmt                       *sql.Stmt
	setUserSessionStmt                        *sql.Stmt
	setWinnerStmt                             *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                                        tx,
		tx:                                        tx,
		addMatchStmt:                              q.addMatchStmt,
		addOrUpdatePlaylistStmt:                   q.addOrUpdatePlaylistStmt,
		addOrUpdatePlaylistItemStmt:               q.addOrUpdatePlaylistItemStmt,
		addPlaylistAddedByUserStmt:                q.addPlaylistAddedByUserStmt,
		addPlaylistItemBelongsToPlaylistStmt:      q.addPlaylistItemBelongsToPlaylistStmt,
		addSessionStmt:                            q.addSessionStmt,
		addUserStmt:                               q.addUserStmt,
		countMatchesForRoundStmt:                  q.countMatchesForRoundStmt,
		deleteItemFromPlaylistStmt:                q.deleteItemFromPlaylistStmt,
		deleteMatchesForSessionStmt:               q.deleteMatchesForSessionStmt,
		deletePossibleNextItemsForSessionStmt:     q.deletePossibleNextItemsForSessionStmt,
		deleteSessionStmt:                         q.deleteSessionStmt,
		getAllWinnersForUserStmt:                  q.getAllWinnersForUserStmt,
		getCurrentRoundStmt:                       q.getCurrentRoundStmt,
		getItemIdsForPlaylistStmt:                 q.getItemIdsForPlaylistStmt,
		getNextPairStmt:                           q.getNextPairStmt,
		getNonActiveUserSessionsStmt:              q.getNonActiveUserSessionsStmt,
		getNumberOfMatchesCompletedStmt:           q.getNumberOfMatchesCompletedStmt,
		getPlaylistStmt:                           q.getPlaylistStmt,
		getPlaylistItemStmt:                       q.getPlaylistItemStmt,
		getPlaylistsForUserStmt:                   q.getPlaylistsForUserStmt,
		getSessionStmt:                            q.getSessionStmt,
		getStatistics1Stmt:                        q.getStatistics1Stmt,
		getUserStmt:                               q.getUserStmt,
		getWinnerStmt:                             q.getWinnerStmt,
		initializePossibleNextItemsForSessionStmt: q.initializePossibleNextItemsForSessionStmt,
		setCurrentRoundStmt:                       q.setCurrentRoundStmt,
		setUserSessionStmt:                        q.setUserSessionStmt,
		setWinnerStmt:                             q.setWinnerStmt,
	}
}
