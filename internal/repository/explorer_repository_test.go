package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	explorerdb "github.com/backend-interview-task/db/gen/explorer"
	"github.com/backend-interview-task/internal/repository"
	"github.com/backend-interview-task/utils"
)

type ExplorerRepositoryTestSuite struct {
	suite.Suite
	mock pgxmock.PgxPoolIface
	repo repository.ExplorerRepository
	ctx  context.Context
}

func TestExplorerRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(ExplorerRepositoryTestSuite))
}

func (s *ExplorerRepositoryTestSuite) SetupTest() {
	s.ctx = context.Background()

	var err error
	s.mock, err = pgxmock.NewPool()
	s.Require().NoError(err)

	logger := zaptest.NewLogger(s.T())
	s.repo = repository.NewExplorerRepository(s.mock, logger)
}

func (s *ExplorerRepositoryTestSuite) TearDownTest() {
	s.mock.Close()
}

func (s *ExplorerRepositoryTestSuite) TestGetLikers_Success_NoPagination() {
	recipientUserID := "user123"
	paginationToken := ""

	// Empty token means default cursor with limit 10
	expectedSQL := `SELECT .* FROM decisions WHERE .*`

	rows := pgxmock.NewRows([]string{"actor_user_id", "timestamp"}).
		AddRow("actor1", int64(1234))

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true).
		WillReturnRows(rows)

	likers, nextToken, err := s.repo.GetLikers(s.ctx, recipientUserID, paginationToken)

	s.NoError(err)
	s.Len(likers, 1)
	s.Equal("actor1", likers[0].ActorID)
	s.Equal(int64(1234), likers[0].Timestamp)
	s.Empty(nextToken)
	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestGetLikers_Success_WithPagination() {
	recipientUserID := "user123"
	cursor := &utils.Cursor{
		LastCreatedAt: 123,
		Limit:         2,
	}
	paginationToken, _ := cursor.Encode()

	expectedSQL := `SELECT .* FROM decisions WHERE .*`

	rows := pgxmock.NewRows([]string{"actor_user_id", "timestamp"}).
		AddRow("actor1", int64(12345)).
		AddRow("actor2", int64(123456)).
		AddRow("actor3", int64(1234567))

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true, int64(123)).
		WillReturnRows(rows)

	likers, nextToken, err := s.repo.GetLikers(s.ctx, recipientUserID, paginationToken)

	s.NoError(err)
	s.Len(likers, 2)
	s.Equal("actor1", likers[0].ActorID)
	s.Equal(int64(12345), likers[0].Timestamp)
	s.Equal("actor2", likers[1].ActorID)
	s.Equal(int64(123456), likers[1].Timestamp)
	s.NotEmpty(nextToken)

	// Verify next token contains correct timestamp
	decodedCursor, decodeErr := utils.DecodeCursor(nextToken)
	s.NoError(decodeErr)
	s.Equal(int64(123456), decodedCursor.LastCreatedAt)
	s.Equal(2, decodedCursor.Limit)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestGetLikers_EmptyResult() {
	recipientUserID := "user123"
	paginationToken := ""

	expectedSQL := `SELECT .* FROM decisions WHERE .*`

	rows := pgxmock.NewRows([]string{"actor_user_id", "timestamp"})

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true).
		WillReturnRows(rows)

	likers, nextToken, err := s.repo.GetLikers(s.ctx, recipientUserID, paginationToken)

	s.NoError(err)
	s.Empty(likers)
	s.Empty(nextToken)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestGetLikers_InvalidPaginationToken() {
	recipientUserID := "user123"
	invalidToken := "invalid_token"

	likers, nextToken, err := s.repo.GetLikers(s.ctx, recipientUserID, invalidToken)

	s.Error(err)
	s.Contains(err.Error(), "invalid paginationToken")
	s.Nil(likers)
	s.Empty(nextToken)
}

func (s *ExplorerRepositoryTestSuite) TestGetLikers_QueryError() {
	recipientUserID := "user123"
	paginationToken := ""

	expectedSQL := `SELECT .* FROM decisions WHERE .*`

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true).
		WillReturnError(errors.New("database connection failed"))

	likers, nextToken, err := s.repo.GetLikers(s.ctx, recipientUserID, paginationToken)

	s.Error(err)
	s.Contains(err.Error(), "failed to get likers")
	s.Nil(likers)
	s.Empty(nextToken)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestGetNewLikers_Success_NoPagination() {
	recipientUserID := "user123"
	paginationToken := ""

	expectedSQL := `SELECT .* FROM decisions d1 LEFT JOIN decisions d2 ON d1.actor_user_id = d2.recipient_user_id WHERE .*`

	rows := pgxmock.NewRows([]string{"actor_user_id", "timestamp"}).
		AddRow("newactor1", int64(1234)).
		AddRow("newactor2", int64(12345))

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true).
		WillReturnRows(rows)

	likers, nextToken, err := s.repo.GetNewLikers(s.ctx, recipientUserID, paginationToken)

	s.NoError(err)
	s.Len(likers, 2)
	s.Equal("newactor1", likers[0].ActorID)
	s.Equal(int64(1234), likers[0].Timestamp)
	s.Equal("newactor2", likers[1].ActorID)
	s.Equal(int64(12345), likers[1].Timestamp)
	s.Empty(nextToken) // No next token since results <= limit

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestGetNewLikers_Success_WithPagination() {
	recipientUserID := "user123"
	cursor := &utils.Cursor{
		LastCreatedAt: 123,
		Limit:         2,
	}
	paginationToken, _ := cursor.Encode()

	expectedSQL := `SELECT .* FROM decisions d1 LEFT JOIN decisions d2 ON d1.actor_user_id = d2.recipient_user_id WHERE .*`

	rows := pgxmock.NewRows([]string{"actor_user_id", "timestamp"}).
		AddRow("newactor1", int64(1234)).
		AddRow("newactor2", int64(12345)).
		AddRow("newactor3", int64(123456))

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true, int64(123)).
		WillReturnRows(rows)

	likers, nextToken, err := s.repo.GetNewLikers(s.ctx, recipientUserID, paginationToken)

	s.NoError(err)
	s.Len(likers, 2)
	s.Equal("newactor1", likers[0].ActorID)
	s.Equal("newactor2", likers[1].ActorID)
	s.NotEmpty(nextToken)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestGetNewLikers_EmptyResult() {
	recipientUserID := "user123"
	paginationToken := ""

	expectedSQL := `SELECT .* FROM decisions d1 LEFT JOIN decisions d2 ON d1.actor_user_id = d2.recipient_user_id WHERE .*`

	rows := pgxmock.NewRows([]string{"actor_user_id", "timestamp"})

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true).
		WillReturnRows(rows)

	likers, nextToken, err := s.repo.GetNewLikers(s.ctx, recipientUserID, paginationToken)

	s.NoError(err)
	s.Empty(likers)
	s.Empty(nextToken)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestGetNewLikers_InvalidPaginationToken() {
	recipientUserID := "user123"
	invalidToken := "invalid_token"

	likers, nextToken, err := s.repo.GetNewLikers(s.ctx, recipientUserID, invalidToken)

	s.Error(err)
	s.Contains(err.Error(), "invalid paginationToken")
	s.Nil(likers)
	s.Empty(nextToken)
}

func (s *ExplorerRepositoryTestSuite) TestGetNewLikers_QueryError() {
	recipientUserID := "user123"
	paginationToken := ""

	expectedSQL := `SELECT .* FROM decisions d1 LEFT JOIN decisions d2 ON d1.actor_user_id = d2.recipient_user_id WHERE .*`

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID, true).
		WillReturnError(errors.New("database connection failed"))

	likers, nextToken, err := s.repo.GetNewLikers(s.ctx, recipientUserID, paginationToken)

	s.Error(err)
	s.Contains(err.Error(), "failed to get new likers")
	s.Nil(likers)
	s.Empty(nextToken)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestCountLikes_Success() {
	recipientUserID := "user123"
	expectedCount := int64(42)

	expectedSQL := `SELECT .* FROM decisions WHERE .*`

	rows := pgxmock.NewRows([]string{"count"}).AddRow(expectedCount)

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID).
		WillReturnRows(rows)

	count, err := s.repo.CountLikes(s.ctx, recipientUserID)

	s.NoError(err)
	s.Equal(expectedCount, count)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestCountLikes_ZeroCount() {
	recipientUserID := "user123"

	expectedSQL := `SELECT .* FROM decisions WHERE .*`

	rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID).
		WillReturnRows(rows)

	count, err := s.repo.CountLikes(s.ctx, recipientUserID)

	s.NoError(err)
	s.Equal(int64(0), count)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestCountLikes_QueryError() {
	recipientUserID := "user123"

	expectedSQL := `SELECT .* FROM decisions WHERE .*`

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(recipientUserID).
		WillReturnError(errors.New("database connection failed"))

	count, err := s.repo.CountLikes(s.ctx, recipientUserID)

	s.Error(err)
	s.Equal(int64(0), count)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestCreateDecision_Success() {
	params := explorerdb.CreateDecisionParams{
		ActorUserID:     "actor123",
		RecipientUserID: "recipient456",
		LikedRecipient:  true,
	}

	expectedSQL := `INSERT INTO decisions .* VALUES .* ON CONFLICT .* DO UPDATE .*`

	s.mock.ExpectExec(expectedSQL).
		WithArgs(params.ActorUserID, params.RecipientUserID, params.LikedRecipient).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := s.repo.CreateDecision(s.ctx, params)

	s.NoError(err)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestCreateDecision_DislikeDecision() {
	params := explorerdb.CreateDecisionParams{
		ActorUserID:     "actor123",
		RecipientUserID: "recipient456",
		LikedRecipient:  false, // Dislike
	}

	expectedSQL := `INSERT INTO decisions .* VALUES .* ON CONFLICT .* DO UPDATE .*`

	s.mock.ExpectExec(expectedSQL).
		WithArgs(params.ActorUserID, params.RecipientUserID, params.LikedRecipient).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := s.repo.CreateDecision(s.ctx, params)

	s.NoError(err)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestCreateDecision_Error() {
	params := explorerdb.CreateDecisionParams{
		ActorUserID:     "actor123",
		RecipientUserID: "recipient456",
		LikedRecipient:  true,
	}

	expectedSQL := `INSERT INTO decisions .* VALUES .* ON CONFLICT .* DO UPDATE .*`

	s.mock.ExpectExec(expectedSQL).
		WithArgs(params.ActorUserID, params.RecipientUserID, params.LikedRecipient).
		WillReturnError(errors.New("constraint violation"))

	err := s.repo.CreateDecision(s.ctx, params)

	s.Error(err)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestHasMutualLike_True() {
	params := explorerdb.HasMutualLikeParams{
		ActorUserID:     "actor123",
		RecipientUserID: "recipient456",
	}

	expectedSQL := `SELECT .*`

	mutualLike := true
	rows := pgxmock.NewRows([]string{"exists"}).AddRow(&mutualLike)

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(params.ActorUserID, params.RecipientUserID).
		WillReturnRows(rows)

	result, err := s.repo.HasMutualLike(s.ctx, params)

	s.NoError(err)
	s.NotNil(result)
	s.True(*result)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestHasMutualLike_False() {
	params := explorerdb.HasMutualLikeParams{
		ActorUserID:     "actor123",
		RecipientUserID: "recipient456",
	}

	expectedSQL := `SELECT .*`

	mutualLike := false
	rows := pgxmock.NewRows([]string{"exists"}).AddRow(&mutualLike)

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(params.ActorUserID, params.RecipientUserID).
		WillReturnRows(rows)

	result, err := s.repo.HasMutualLike(s.ctx, params)

	s.NoError(err)
	s.NotNil(result)
	s.False(*result)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestHasMutualLike_Null() {
	params := explorerdb.HasMutualLikeParams{
		ActorUserID:     "actor123",
		RecipientUserID: "recipient456",
	}

	expectedSQL := `SELECT .*`

	rows := pgxmock.NewRows([]string{"exists"}).AddRow(nil)

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(params.ActorUserID, params.RecipientUserID).
		WillReturnRows(rows)

	result, err := s.repo.HasMutualLike(s.ctx, params)

	s.NoError(err)
	s.Nil(result)

	s.NoError(s.mock.ExpectationsWereMet())
}

func (s *ExplorerRepositoryTestSuite) TestHasMutualLike_Error() {
	params := explorerdb.HasMutualLikeParams{
		ActorUserID:     "actor123",
		RecipientUserID: "recipient456",
	}

	expectedSQL := `SELECT .*`

	s.mock.ExpectQuery(expectedSQL).
		WithArgs(params.ActorUserID, params.RecipientUserID).
		WillReturnError(errors.New("database connection failed"))

	result, err := s.repo.HasMutualLike(s.ctx, params)

	s.Error(err)
	s.Nil(result)

	s.NoError(s.mock.ExpectationsWereMet())
}
