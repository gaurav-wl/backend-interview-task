package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	coremock "github.com/backend-interview-task/mocks/core"
	pb "github.com/backend-interview-task/proto"
	"github.com/backend-interview-task/utils"
)

type ExploreServiceTestSuite struct {
	suite.Suite
	mockCore *coremock.ExplorerCore
	service  *ExploreService
	ctx      context.Context
}

func TestExploreServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ExploreServiceTestSuite))
}

func (s *ExploreServiceTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.mockCore = new(coremock.ExplorerCore)
	logger := zaptest.NewLogger(s.T())
	s.service = NewExploreService(s.mockCore, logger)
}

func (s *ExploreServiceTestSuite) TearDownTest() {
	s.mockCore.AssertExpectations(s.T())
}

func (s *ExploreServiceTestSuite) TestListLikedYou_Success() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "user123",
		PaginationToken: utils.ToPointer("token456"),
	}

	expectedResp := &pb.ListLikedYouResponse{
		Likers: []*pb.ListLikedYouResponse_Liker{
			{ActorId: "actor1", UnixTimestamp: 1640995200},
			{ActorId: "actor2", UnixTimestamp: 1640995100},
		},
		NextPaginationToken: utils.ToPointer("next_token"),
	}

	s.mockCore.EXPECT().ListLikers(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.ListLikedYou(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
}

func (s *ExploreServiceTestSuite) TestListLikedYou_Success_NoPaginationToken() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "user123",
		PaginationToken: nil,
	}

	expectedResp := &pb.ListLikedYouResponse{
		Likers: []*pb.ListLikedYouResponse_Liker{
			{ActorId: "actor1", UnixTimestamp: 1640995200},
		},
	}

	s.mockCore.EXPECT().ListLikers(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.ListLikedYou(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
}

func (s *ExploreServiceTestSuite) TestListLikedYou_EmptyRecipientUserId() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "",
		PaginationToken: utils.ToPointer("token456"),
	}

	resp, err := s.service.ListLikedYou(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.InvalidArgument, status.Code(err))
	s.Contains(err.Error(), "recipient_user_id is required")
	s.mockCore.AssertNotCalled(s.T(), "ListLikers")
}

func (s *ExploreServiceTestSuite) TestListLikedYou_CoreError() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "user123",
		PaginationToken: utils.ToPointer("token456"),
	}

	coreErr := errors.New("database connection failed")
	s.mockCore.EXPECT().ListLikers(mock.Anything, req).Return(nil, coreErr).Once()

	resp, err := s.service.ListLikedYou(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to get likers")
}

func (s *ExploreServiceTestSuite) TestListNewLikedYou_Success() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "user123",
		PaginationToken: utils.ToPointer("token456"),
	}

	expectedResp := &pb.ListLikedYouResponse{
		Likers: []*pb.ListLikedYouResponse_Liker{
			{ActorId: "newactor1", UnixTimestamp: 1640995200},
			{ActorId: "newactor2", UnixTimestamp: 1640995100},
		},
		NextPaginationToken: utils.ToPointer("next_token"),
	}

	s.mockCore.EXPECT().ListNewLikers(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.ListNewLikedYou(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
}

func (s *ExploreServiceTestSuite) TestListNewLikedYou_Success_EmptyResult() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "user123",
		PaginationToken: nil,
	}

	expectedResp := &pb.ListLikedYouResponse{
		Likers: []*pb.ListLikedYouResponse_Liker{},
	}

	s.mockCore.EXPECT().ListNewLikers(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.ListNewLikedYou(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
	s.Empty(resp.Likers)
}

func (s *ExploreServiceTestSuite) TestListNewLikedYou_EmptyRecipientUserId() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "",
		PaginationToken: utils.ToPointer("token456"),
	}

	resp, err := s.service.ListNewLikedYou(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.InvalidArgument, status.Code(err))
	s.Contains(err.Error(), "recipient_user_id is required")
	s.mockCore.AssertNotCalled(s.T(), "ListNewLikers")
}

func (s *ExploreServiceTestSuite) TestListNewLikedYou_CoreError() {
	req := &pb.ListLikedYouRequest{
		RecipientUserId: "user123",
		PaginationToken: utils.ToPointer("token456"),
	}

	coreErr := errors.New("database timeout")
	s.mockCore.EXPECT().ListNewLikers(mock.Anything, req).Return(nil, coreErr).Once()

	resp, err := s.service.ListNewLikedYou(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to get new likers")
}

func (s *ExploreServiceTestSuite) TestCountLikedYou_Success() {
	req := &pb.CountLikedYouRequest{
		RecipientUserId: "user123",
	}

	expectedResp := &pb.CountLikedYouResponse{
		Count: 42,
	}

	s.mockCore.EXPECT().CountLikers(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.CountLikedYou(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
	s.Equal(uint64(42), resp.Count)
}

func (s *ExploreServiceTestSuite) TestCountLikedYou_Success_ZeroCount() {
	req := &pb.CountLikedYouRequest{
		RecipientUserId: "user123",
	}

	expectedResp := &pb.CountLikedYouResponse{
		Count: 0,
	}

	s.mockCore.EXPECT().CountLikers(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.CountLikedYou(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
	s.Equal(uint64(0), resp.Count)
}

func (s *ExploreServiceTestSuite) TestCountLikedYou_EmptyRecipientUserId() {
	req := &pb.CountLikedYouRequest{
		RecipientUserId: "",
	}

	resp, err := s.service.CountLikedYou(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.InvalidArgument, status.Code(err))
	s.Contains(err.Error(), "recipient_user_id is required")
	s.mockCore.AssertNotCalled(s.T(), "CountLikers")
}

func (s *ExploreServiceTestSuite) TestCountLikedYou_CoreError() {
	req := &pb.CountLikedYouRequest{
		RecipientUserId: "user123",
	}

	coreErr := errors.New("cache unavailable")
	s.mockCore.EXPECT().CountLikers(mock.Anything, req).Return(nil, coreErr).Once()

	resp, err := s.service.CountLikedYou(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to count likers")
}

func (s *ExploreServiceTestSuite) TestPutDecision_Success_LikedRecipient() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	expectedResp := &pb.PutDecisionResponse{
		MutualLikes: true,
	}

	s.mockCore.EXPECT().CreateDecision(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.PutDecision(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
	s.True(resp.MutualLikes)
}

func (s *ExploreServiceTestSuite) TestPutDecision_Success_DislikedRecipient() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  false,
	}

	expectedResp := &pb.PutDecisionResponse{
		MutualLikes: false,
	}

	s.mockCore.EXPECT().CreateDecision(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.PutDecision(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
	s.False(resp.MutualLikes)
}

func (s *ExploreServiceTestSuite) TestPutDecision_Success_NoMutualLike() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	expectedResp := &pb.PutDecisionResponse{
		MutualLikes: false,
	}

	s.mockCore.EXPECT().CreateDecision(mock.Anything, req).Return(expectedResp, nil).Once()

	resp, err := s.service.PutDecision(s.ctx, req)

	s.NoError(err)
	s.Equal(expectedResp, resp)
	s.False(resp.MutualLikes)
}

func (s *ExploreServiceTestSuite) TestPutDecision_EmptyActorUserId() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	resp, err := s.service.PutDecision(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.InvalidArgument, status.Code(err))
	s.Contains(err.Error(), "actor_user_id is required")
	s.mockCore.AssertNotCalled(s.T(), "CreateDecision")
}

func (s *ExploreServiceTestSuite) TestPutDecision_EmptyRecipientUserId() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "",
		LikedRecipient:  true,
	}

	resp, err := s.service.PutDecision(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.InvalidArgument, status.Code(err))
	s.Contains(err.Error(), "recipient_user_id is required")
	s.mockCore.AssertNotCalled(s.T(), "CreateDecision")
}

func (s *ExploreServiceTestSuite) TestPutDecision_SameActorAndRecipient() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "sameuser123",
		RecipientUserId: "sameuser123",
		LikedRecipient:  true,
	}

	resp, err := s.service.PutDecision(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.InvalidArgument, status.Code(err))
	s.Contains(err.Error(), "actor and recipient cannot be the same user")
	s.mockCore.AssertNotCalled(s.T(), "CreateDecision")
}

func (s *ExploreServiceTestSuite) TestPutDecision_CoreError() {
	req := &pb.PutDecisionRequest{
		ActorUserId:     "actor123",
		RecipientUserId: "recipient456",
		LikedRecipient:  true,
	}

	coreErr := errors.New("database constraint violation")
	s.mockCore.EXPECT().CreateDecision(mock.Anything, req).Return(nil, coreErr).Once()

	resp, err := s.service.PutDecision(s.ctx, req)

	s.Nil(resp)
	s.Error(err)
	s.Equal(codes.Internal, status.Code(err))
	s.Contains(err.Error(), "failed to create decision")
}
