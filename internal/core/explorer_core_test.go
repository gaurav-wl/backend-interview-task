package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	explorerdb "github.com/backend-interview-task/db/gen/explorer"
	"github.com/backend-interview-task/internal/models"
	cachemock "github.com/backend-interview-task/mocks/providers/cache"
	repomock "github.com/backend-interview-task/mocks/repository"
	pb "github.com/backend-interview-task/proto"
	"github.com/backend-interview-task/utils"
)

type ExplorerCoreTestSuite struct {
	suite.Suite
	mockExplorerRepo *repomock.ExplorerRepository
	mockCache        *cachemock.CacheProvider
	explorerCore     ExplorerCore
	logger           *zap.Logger
}

func TestExplorerCoreTestSuite(t *testing.T) {
	suite.Run(t, new(ExplorerCoreTestSuite))
}

func (s *ExplorerCoreTestSuite) SetupTest() {
	s.logger = zap.NewNop()
	s.mockExplorerRepo = new(repomock.ExplorerRepository)
	s.mockCache = new(cachemock.CacheProvider)
	s.explorerCore = NewExploreCore(s.mockExplorerRepo, s.mockCache, s.logger)
}

func (s *ExplorerCoreTestSuite) TearDownTest() {
	s.mockExplorerRepo.AssertExpectations(s.T())
	s.mockCache.AssertExpectations(s.T())
}

func (s *ExplorerCoreTestSuite) TestListLikers_CacheHit() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: utils.ToPointer("eyJsYXN0X2NyZWF0ZWRfYXQiOiAxNzU2Mzc3NjU0LCAibGltaXQiOiAxMH0="),
	}
	cacheKey := utils.LikersKey(req.RecipientUserId, req.GetPaginationToken())

	cachedEmptyResp := &pb.ListLikedYouResponse{}
	cachedFinalResp := pb.ListLikedYouResponse{
		Likers: []*pb.ListLikedYouResponse_Liker{
			{ActorId: "testActor1", UnixTimestamp: 100},
			{ActorId: "testActor2", UnixTimestamp: 200},
		},
	}

	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, cachedEmptyResp).
		Run(func(ctx context.Context, key string, out interface{}) {
			obj := out.(*pb.ListLikedYouResponse)
			*obj = cachedFinalResp
		}).Return(true, nil).Once()

	resp, err := s.explorerCore.ListLikers(context.Background(), req)

	s.NoError(err)
	s.Equal(&cachedFinalResp, resp)
	s.mockExplorerRepo.AssertNotCalled(s.T(), "GetLikers")
}

func (s *ExplorerCoreTestSuite) TestListLikers_CacheMiss_DatabaseSuccess() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: utils.ToPointer("token123"),
	}
	cacheKey := utils.LikersKey(req.RecipientUserId, req.GetPaginationToken())

	// Mock cache miss
	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, &pb.ListLikedYouResponse{}).
		Return(false, nil).Once()

	// Mock repository response
	likers := []models.Liker{
		{ActorID: "actor1", Timestamp: 100},
		{ActorID: "actor2", Timestamp: 200},
	}
	nextToken := "nextPageToken"

	s.mockExplorerRepo.EXPECT().GetLikers(mock.Anything, req.RecipientUserId, req.GetPaginationToken()).
		Return(likers, nextToken, nil).Once()

	// Mock cache set (async goroutine)
	s.mockCache.EXPECT().SetJSON(mock.Anything, cacheKey, mock.Anything, utils.LikersTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.ListLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.Likers, 2)
	s.Equal("actor1", resp.Likers[0].ActorId)
	s.Equal(uint64(100), resp.Likers[0].UnixTimestamp)
	s.Equal("actor2", resp.Likers[1].ActorId)
	s.Equal(uint64(200), resp.Likers[1].UnixTimestamp)
	s.Equal(nextToken, *resp.NextPaginationToken)
}

func (s *ExplorerCoreTestSuite) TestListLikers_CacheMiss_DatabaseSuccess_NoNextToken() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: nil, // No pagination token
	}
	cacheKey := utils.LikersKey(req.RecipientUserId, req.GetPaginationToken())

	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, &pb.ListLikedYouResponse{}).
		Return(false, nil).Once()

	likers := []models.Liker{
		{ActorID: "actor1", Timestamp: 100},
	}

	s.mockExplorerRepo.EXPECT().GetLikers(mock.Anything, req.RecipientUserId, req.GetPaginationToken()).
		Return(likers, "", nil).Once() // Empty next token

	s.mockCache.EXPECT().SetJSON(mock.Anything, cacheKey, mock.Anything, utils.LikersTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.ListLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.Likers, 1)
	s.Nil(resp.NextPaginationToken) // Should be nil when no next token
}

func (s *ExplorerCoreTestSuite) TestListLikers_CacheMiss_DatabaseError() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: utils.ToPointer("token123"),
	}
	cacheKey := utils.LikersKey(req.RecipientUserId, req.GetPaginationToken())

	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, &pb.ListLikedYouResponse{}).
		Return(false, nil).Once()

	s.mockExplorerRepo.EXPECT().GetLikers(mock.Anything, req.RecipientUserId, req.GetPaginationToken()).
		Return(nil, "", errors.New("database connection failed")).Once()

	resp, err := s.explorerCore.ListLikers(context.Background(), req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to get likers")
}

func (s *ExplorerCoreTestSuite) TestListLikers_CacheError_DatabaseSuccess() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: utils.ToPointer("token123"),
	}
	cacheKey := utils.LikersKey(req.RecipientUserId, req.GetPaginationToken())

	// Mock cache error
	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, &pb.ListLikedYouResponse{}).
		Return(false, errors.New("cache unavailable")).Once()

	likers := []models.Liker{
		{ActorID: "actor1", Timestamp: 100},
	}

	s.mockExplorerRepo.EXPECT().GetLikers(mock.Anything, req.RecipientUserId, req.GetPaginationToken()).
		Return(likers, "", nil).Once()

	s.mockCache.EXPECT().SetJSON(mock.Anything, cacheKey, mock.Anything, utils.LikersTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.ListLikers(context.Background(), req)

	s.NoError(err) // Should still succeed despite cache error
	s.NotNil(resp)
	s.Len(resp.Likers, 1)
}

func (s *ExplorerCoreTestSuite) TestListNewLikers_CacheHit() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: utils.ToPointer("newtoken123"),
	}
	cacheKey := utils.NewLikersKey(req.RecipientUserId, req.GetPaginationToken())

	cachedEmptyResp := &pb.ListLikedYouResponse{}
	cachedFinalResp := pb.ListLikedYouResponse{
		Likers: []*pb.ListLikedYouResponse_Liker{
			{ActorId: "newActor1", UnixTimestamp: 300},
		},
	}

	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, cachedEmptyResp).
		Run(func(ctx context.Context, key string, out interface{}) {
			obj := out.(*pb.ListLikedYouResponse)
			*obj = cachedFinalResp
		}).Return(true, nil).Once()

	resp, err := s.explorerCore.ListNewLikers(context.Background(), req)

	s.NoError(err)
	s.Equal(&cachedFinalResp, resp)
	s.mockExplorerRepo.AssertNotCalled(s.T(), "GetNewLikers")
}

func (s *ExplorerCoreTestSuite) TestListNewLikers_CacheMiss_DatabaseSuccess() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: utils.ToPointer("newtoken123"),
	}
	cacheKey := utils.NewLikersKey(req.RecipientUserId, req.GetPaginationToken())

	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, &pb.ListLikedYouResponse{}).
		Return(false, nil).Once()

	likers := []models.Liker{
		{ActorID: "newactor1", Timestamp: 300},
		{ActorID: "newactor2", Timestamp: 400},
	}
	nextToken := "newNextToken"

	s.mockExplorerRepo.EXPECT().GetNewLikers(mock.Anything, req.RecipientUserId, req.GetPaginationToken()).
		Return(likers, nextToken, nil).Once()

	s.mockCache.EXPECT().SetJSON(mock.Anything, cacheKey, mock.Anything, utils.NewLikersTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.ListNewLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.Likers, 2)
	s.Equal("newactor1", resp.Likers[0].ActorId)
	s.Equal(uint64(300), resp.Likers[0].UnixTimestamp)
	s.Equal(nextToken, *resp.NextPaginationToken)
}

func (s *ExplorerCoreTestSuite) TestListNewLikers_DatabaseError() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: utils.ToPointer("newtoken123"),
	}
	cacheKey := utils.NewLikersKey(req.RecipientUserId, req.GetPaginationToken())

	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, &pb.ListLikedYouResponse{}).
		Return(false, nil).Once()

	s.mockExplorerRepo.EXPECT().GetNewLikers(mock.Anything, req.RecipientUserId, req.GetPaginationToken()).
		Return(nil, "", errors.New("database timeout")).Once()

	resp, err := s.explorerCore.ListNewLikers(context.Background(), req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to get new likers")
}

func (s *ExplorerCoreTestSuite) TestCountLikers_CacheHit() {
	req := &pb.CountLikedYouRequest{RecipientUserId: "testuser"}
	cacheKey := utils.LikersCountKey(req.RecipientUserId)

	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return("42", nil).Once()

	resp, err := s.explorerCore.CountLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Equal(uint64(42), resp.Count)
	s.mockExplorerRepo.AssertNotCalled(s.T(), "CountLikes")
}

func (s *ExplorerCoreTestSuite) TestCountLikers_CacheHit_ZeroCount() {
	req := &pb.CountLikedYouRequest{RecipientUserId: "testuser"}
	cacheKey := utils.LikersCountKey(req.RecipientUserId)

	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return("0", nil).Once()

	resp, err := s.explorerCore.CountLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Equal(uint64(0), resp.Count)
	s.mockExplorerRepo.AssertNotCalled(s.T(), "CountLikes")
}

func (s *ExplorerCoreTestSuite) TestCountLikers_CacheInvalidValue_DatabaseSuccess() {
	req := &pb.CountLikedYouRequest{RecipientUserId: "testuser"}
	cacheKey := utils.LikersCountKey(req.RecipientUserId)

	// Cache returns invalid value
	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return("invalid_number", nil).Once()

	s.mockExplorerRepo.EXPECT().CountLikes(mock.Anything, req.RecipientUserId).
		Return(int64(15), nil).Once()

	s.mockCache.EXPECT().Set(mock.Anything, cacheKey, "15", utils.LikersCountTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.CountLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Equal(uint64(15), resp.Count)
}

func (s *ExplorerCoreTestSuite) TestCountLikers_CacheMiss_DatabaseSuccess() {
	req := &pb.CountLikedYouRequest{RecipientUserId: "testuser"}
	cacheKey := utils.LikersCountKey(req.RecipientUserId)

	// Cache miss (empty string)
	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return("", errors.New("cache miss")).Once()

	s.mockExplorerRepo.EXPECT().CountLikes(mock.Anything, req.RecipientUserId).
		Return(int64(25), nil).Once()

	s.mockCache.EXPECT().Set(mock.Anything, cacheKey, "25", utils.LikersCountTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.CountLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Equal(uint64(25), resp.Count)
}

func (s *ExplorerCoreTestSuite) TestCountLikers_CacheError_DatabaseSuccess() {
	req := &pb.CountLikedYouRequest{RecipientUserId: "testuser"}
	cacheKey := utils.LikersCountKey(req.RecipientUserId)

	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return("", errors.New("cache unavailable")).Once()

	s.mockExplorerRepo.EXPECT().CountLikes(mock.Anything, req.RecipientUserId).
		Return(int64(35), nil).Once()

	s.mockCache.EXPECT().Set(mock.Anything, cacheKey, "35", utils.LikersCountTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.CountLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Equal(uint64(35), resp.Count)
}

func (s *ExplorerCoreTestSuite) TestCountLikers_DatabaseError() {
	req := &pb.CountLikedYouRequest{RecipientUserId: "testuser"}
	cacheKey := utils.LikersCountKey(req.RecipientUserId)

	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return("", errors.New("cache miss")).Once()

	s.mockExplorerRepo.EXPECT().CountLikes(mock.Anything, req.RecipientUserId).
		Return(int64(0), errors.New("database connection failed")).Once()

	resp, err := s.explorerCore.CountLikers(context.Background(), req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to count likers")
}

func (s *ExplorerCoreTestSuite) TestCreateDecision_LikedRecipient_MutualLike() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	createParams := explorerdb.CreateDecisionParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
		LikedRecipient:  req.LikedRecipient,
	}

	mutualParams := explorerdb.HasMutualLikeParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
	}

	s.mockExplorerRepo.EXPECT().CreateDecision(mock.Anything, createParams).Return(nil).Once()

	// Return pointer to true for mutual like
	mutualLike := true
	s.mockExplorerRepo.EXPECT().HasMutualLike(mock.Anything, mutualParams).
		Return(&mutualLike, nil).Once()

	resp, err := s.explorerCore.CreateDecision(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.True(resp.MutualLikes)
}

func (s *ExplorerCoreTestSuite) TestCreateDecision_LikedRecipient_NoMutualLike() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	createParams := explorerdb.CreateDecisionParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
		LikedRecipient:  req.LikedRecipient,
	}

	mutualParams := explorerdb.HasMutualLikeParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
	}

	s.mockExplorerRepo.EXPECT().CreateDecision(mock.Anything, createParams).Return(nil).Once()

	// Return pointer to false for no mutual like
	mutualLike := false
	s.mockExplorerRepo.EXPECT().HasMutualLike(mock.Anything, mutualParams).
		Return(&mutualLike, nil).Once()

	resp, err := s.explorerCore.CreateDecision(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.False(resp.MutualLikes)
}

func (s *ExplorerCoreTestSuite) TestCreateDecision_LikedRecipient_MutualLikeNil() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	createParams := explorerdb.CreateDecisionParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
		LikedRecipient:  req.LikedRecipient,
	}

	mutualParams := explorerdb.HasMutualLikeParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
	}

	s.mockExplorerRepo.EXPECT().CreateDecision(mock.Anything, createParams).Return(nil).Once()

	// Return nil for mutual like (no result)
	s.mockExplorerRepo.EXPECT().HasMutualLike(mock.Anything, mutualParams).
		Return(nil, nil).Once()

	resp, err := s.explorerCore.CreateDecision(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.False(resp.MutualLikes) // Should be false when nil
}

func (s *ExplorerCoreTestSuite) TestCreateDecision_NotLikedRecipient() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  false, // Not liked
	}

	createParams := explorerdb.CreateDecisionParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
		LikedRecipient:  req.LikedRecipient,
	}

	s.mockExplorerRepo.EXPECT().CreateDecision(mock.Anything, createParams).Return(nil).Once()

	// Should NOT call HasMutualLike when not liked
	resp, err := s.explorerCore.CreateDecision(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.False(resp.MutualLikes)
	s.mockExplorerRepo.AssertNotCalled(s.T(), "HasMutualLike")
}

func (s *ExplorerCoreTestSuite) TestCreateDecision_CreateDecisionError() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	createParams := explorerdb.CreateDecisionParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
		LikedRecipient:  req.LikedRecipient,
	}

	s.mockExplorerRepo.EXPECT().CreateDecision(mock.Anything, createParams).
		Return(errors.New("database constraint violation")).Once()

	resp, err := s.explorerCore.CreateDecision(context.Background(), req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to create decision")
	s.mockExplorerRepo.AssertNotCalled(s.T(), "HasMutualLike")
}

func (s *ExplorerCoreTestSuite) TestCreateDecision_HasMutualLikeError() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	createParams := explorerdb.CreateDecisionParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
		LikedRecipient:  req.LikedRecipient,
	}

	mutualParams := explorerdb.HasMutualLikeParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
	}

	s.mockExplorerRepo.EXPECT().CreateDecision(mock.Anything, createParams).Return(nil).Once()

	s.mockExplorerRepo.EXPECT().HasMutualLike(mock.Anything, mutualParams).
		Return(nil, errors.New("database timeout")).Once()

	resp, err := s.explorerCore.CreateDecision(context.Background(), req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to check mutual like")
}

func (s *ExplorerCoreTestSuite) TestListLikers_EmptyResult() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "testuser",
		PaginationToken: nil,
	}
	cacheKey := utils.LikersKey(req.RecipientUserId, req.GetPaginationToken())

	s.mockCache.EXPECT().GetJSON(mock.Anything, cacheKey, &pb.ListLikedYouResponse{}).
		Return(false, nil).Once()

	// Empty likers result
	s.mockExplorerRepo.EXPECT().GetLikers(mock.Anything, req.RecipientUserId, req.GetPaginationToken()).
		Return([]models.Liker{}, "", nil).Once()

	s.mockCache.EXPECT().SetJSON(mock.Anything, cacheKey, mock.Anything, utils.LikersTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.ListLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Empty(resp.Likers)
	s.Nil(resp.NextPaginationToken)
}

func (s *ExplorerCoreTestSuite) TestCountLikers_ZeroCountFromDatabase() {
	req := &pb.CountLikedYouRequest{RecipientUserId: "testuser"}
	cacheKey := utils.LikersCountKey(req.RecipientUserId)

	s.mockCache.EXPECT().Get(mock.Anything, cacheKey).Return("", errors.New("cache miss")).Once()

	s.mockExplorerRepo.EXPECT().CountLikes(mock.Anything, req.RecipientUserId).
		Return(int64(0), nil).Once()

	s.mockCache.EXPECT().Set(mock.Anything, cacheKey, "0", utils.LikersCountTTL).
		Return(nil).Maybe()

	resp, err := s.explorerCore.CountLikers(context.Background(), req)

	s.NoError(err)
	s.NotNil(resp)
	s.Equal(uint64(0), resp.Count)
}
