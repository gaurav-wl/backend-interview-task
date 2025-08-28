package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"

	explorerdb "github.com/backend-interview-task/db/gen/explorer"
	"github.com/backend-interview-task/internal/models"
	"github.com/backend-interview-task/internal/providers/database"
	"github.com/backend-interview-task/utils"
)

type ExplorerRepository interface {
	GetLikers(ctx context.Context, recipientUserID string, cursor string) ([]models.Liker, string, error)
	GetNewLikers(ctx context.Context, recipientUserID string, cursor string) ([]models.Liker, string, error)
	explorerdb.Querier
}

type explorerStore struct {
	db database.DBProvider
	*explorerdb.Queries
	logger *zap.Logger
}

func NewExplorerRepository(db database.DBProvider, logger *zap.Logger) ExplorerRepository {
	return &explorerStore{
		db:      db,
		logger:  logger,
		Queries: explorerdb.New(db),
	}
}

// GetLikers returns users who liked the recipient with pagination
func (r *explorerStore) GetLikers(ctx context.Context, recipientUserID string, paginationToken string) ([]models.Liker, string, error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	queryBuilder := psql.Select("actor_user_id, EXTRACT(EPOCH FROM created_at)::bigint as timestamp").
		From("decisions").
		Where(squirrel.Eq{"recipient_user_id": recipientUserID}).
		Where(squirrel.Eq{"liked_recipient": true})

	cursor, err := utils.DecodeCursor(paginationToken)
	if err != nil {
		return nil, "", fmt.Errorf("invalid paginationToken: %w", err)
	}

	if cursor == nil || cursor.Limit <= 0 {
		cursor = &utils.Cursor{
			// default limit
			Limit: 20,
		}
	}

	if paginationToken != "" {
		queryBuilder = queryBuilder.Where(squirrel.Lt{"EXTRACT(EPOCH FROM created_at)::bigint": cursor.LastCreatedAt})
	}

	queryBuilder = queryBuilder.
		OrderBy("created_at DESC").
		Limit(uint64(cursor.Limit + 1))

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to get likers",
			zap.String("recipient_user_id", recipientUserID),
			zap.Error(err))
		return nil, "", fmt.Errorf("failed to get likers: %w", err)
	}
	defer rows.Close()

	var likers []models.Liker
	for rows.Next() {
		var liker models.Liker
		if err := rows.Scan(&liker.ActorID, &liker.Timestamp); err != nil {
			return nil, "", fmt.Errorf("failed to scan liker: %w", err)
		}
		likers = append(likers, liker)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating over results: %w", err)
	}

	var nextPaginationToken string
	if len(likers) > cursor.Limit {
		nextCursor := &utils.Cursor{
			LastCreatedAt: likers[cursor.Limit-1].Timestamp,
			Limit:         cursor.Limit,
		}
		nextPaginationToken, err = nextCursor.Encode()
		if err != nil {
			return nil, "", fmt.Errorf("failed to encode next paginationToken: %w", err)
		}
		likers = likers[:cursor.Limit] // Remove the extra item
	}

	return likers, nextPaginationToken, nil
}

// GetNewLikers returns users who liked the recipient but haven't been liked back
func (r *explorerStore) GetNewLikers(ctx context.Context, recipientUserID string, paginationToken string) ([]models.Liker, string, error) {
	args := []interface{}{recipientUserID}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	queryBuilder := psql.Select("d1.actor_user_id, EXTRACT(EPOCH FROM d1.created_at)::bigint as timestamp").
		From("decisions d1").
		LeftJoin("decisions d2 ON d1.actor_user_id = d2.recipient_user_id").
		Where(squirrel.Eq{"d1.recipient_user_id": recipientUserID}).
		Where(squirrel.Eq{"d1.liked_recipient": true}).
		Where(squirrel.Eq{"d2.id": nil})

	cursor, err := utils.DecodeCursor(paginationToken)
	if err != nil {
		return nil, "", fmt.Errorf("invalid paginationToken: %w", err)
	}

	if cursor == nil || cursor.Limit <= 0 {
		cursor = &utils.Cursor{
			Limit: 20,
		}
	}

	if paginationToken != "" {
		queryBuilder = queryBuilder.Where(squirrel.Lt{"EXTRACT(EPOCH FROM d1.created_at)::bigint": cursor.LastCreatedAt})
	}

	queryBuilder = queryBuilder.
		OrderBy("d1.created_at DESC").
		Limit(uint64(cursor.Limit))
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, "", fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to get new likers",
			zap.String("recipient_user_id", recipientUserID),
			zap.Error(err))
		return nil, "", fmt.Errorf("failed to get new likers: %w", err)
	}
	defer rows.Close()

	var likers []models.Liker
	for rows.Next() {
		var liker models.Liker
		if err := rows.Scan(&liker.ActorID, &liker.Timestamp); err != nil {
			return nil, "", fmt.Errorf("failed to scan liker: %w", err)
		}
		likers = append(likers, liker)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating over results: %w", err)
	}

	var nextPaginationToken string
	if len(likers) > cursor.Limit {
		nextCursor := &utils.Cursor{
			LastCreatedAt: likers[cursor.Limit-1].Timestamp,
			Limit:         cursor.Limit,
		}

		nextPaginationToken, err = nextCursor.Encode()
		if err != nil {
			return nil, "", fmt.Errorf("failed to encode next paginationToken: %w", err)
		}

		likers = likers[:cursor.Limit]
	}

	return likers, nextPaginationToken, nil
}
