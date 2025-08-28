package service

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/backend-interview-task/internal/core"
	pb "github.com/backend-interview-task/proto"
)

// ExploreService implements the gRPC service
type ExploreService struct {
	pb.UnimplementedExploreServiceServer
	core   core.ExplorerCore
	logger *zap.Logger
}

func NewExploreService(core core.ExplorerCore, logger *zap.Logger) *ExploreService {
	return &ExploreService{
		core:   core,
		logger: logger,
	}
}

// ListLikedYou returns all users who liked the recipient
func (s *ExploreService) ListLikedYou(ctx context.Context, req *pb.ListLikedYouRequest) (*pb.ListLikedYouResponse, error) {
	if req.RecipientUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_user_id is required")
	}
	resp, err := s.core.ListLikers(ctx, req)
	if err != nil {
		s.logger.Error("Failed to get likers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get likers")
	}

	return resp, nil
}

// ListNewLikedYou returns users who liked the recipient but haven't been liked back
func (s *ExploreService) ListNewLikedYou(ctx context.Context, req *pb.ListLikedYouRequest) (*pb.ListLikedYouResponse, error) {
	if req.RecipientUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_user_id is required")
	}

	// Get new likers with pagination
	resp, err := s.core.ListNewLikers(ctx, req)
	if err != nil {
		s.logger.Error("Failed to get new likers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get new likers")
	}

	return resp, nil
}

// CountLikedYou returns the count of users who liked the recipient
func (s *ExploreService) CountLikedYou(ctx context.Context, req *pb.CountLikedYouRequest) (*pb.CountLikedYouResponse, error) {
	if req.RecipientUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_user_id is required")
	}
	resp, err := s.core.CountLikers(ctx, req)
	if err != nil {
		s.logger.Error("Failed to count likers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to count likers")
	}

	return resp, nil
}

// PutDecision records a decision (like/pass) from actor to recipient
func (s *ExploreService) PutDecision(ctx context.Context, req *pb.PutDecisionRequest) (*pb.PutDecisionResponse, error) {
	if req.ActorUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_user_id is required")
	}
	if req.RecipientUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "recipient_user_id is required")
	}
	if req.ActorUserId == req.RecipientUserId {
		return nil, status.Error(codes.InvalidArgument, "actor and recipient cannot be the same user")
	}
	// Create the decision
	resp, err := s.core.CreateDecision(ctx, req)
	if err != nil {
		s.logger.Error("Failed to create decision", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create decision")
	}

	return resp, nil
}
