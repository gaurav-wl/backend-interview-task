package core

import (
	"context"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	explorerdb "github.com/backend-interview-task/db/gen/explorer"
	"github.com/backend-interview-task/internal/providers/cache"
	"github.com/backend-interview-task/internal/repository"
	pb "github.com/backend-interview-task/proto"
	"github.com/backend-interview-task/utils"
)

type ExplorerCore interface {
	CreateDecision(ctx context.Context, req *pb.PutDecisionRequest) (*pb.PutDecisionResponse, error)
	ListLikers(ctx context.Context, req *pb.ListLikedYouRequest) (*pb.ListLikedYouResponse, error)
	ListNewLikers(ctx context.Context, req *pb.ListLikedYouRequest) (*pb.ListLikedYouResponse, error)
	CountLikers(ctx context.Context, req *pb.CountLikedYouRequest) (*pb.CountLikedYouResponse, error)
}

// exploreCore implements the business logic for the ExploreService
type exploreCore struct {
	repo   repository.ExplorerRepository
	cache  cache.CacheProvider
	logger *zap.Logger
}

// NewExploreCore creates a new ExploreCore to handle the app business logic
func NewExploreCore(repo repository.ExplorerRepository, cache cache.CacheProvider, logger *zap.Logger) ExplorerCore {
	return &exploreCore{
		repo:   repo,
		logger: logger,
		cache:  cache,
	}
}

// ListLikers returns all users who liked the recipient
// First it try from cache, if not found then query from DB
func (s *exploreCore) ListLikers(ctx context.Context, req *pb.ListLikedYouRequest) (*pb.ListLikedYouResponse, error) {
	key := utils.LikersKey(req.GetRecipientUserId(), req.GetPaginationToken())

	var cached pb.ListLikedYouResponse
	if ok, err := s.cache.GetJSON(ctx, key, &cached); err == nil && ok {
		return &cached, nil
	}

	// Get likers with pagination
	likers, nextToken, err := s.repo.GetLikers(ctx, req.RecipientUserId, req.GetPaginationToken())
	if err != nil {
		s.logger.Error("Failed to get likers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get likers")
	}

	// Convert to protobuf format
	pbLikers := make([]*pb.ListLikedYouResponse_Liker, len(likers))
	for i, liker := range likers {
		pbLikers[i] = &pb.ListLikedYouResponse_Liker{
			ActorId:       liker.ActorID,
			UnixTimestamp: uint64(liker.Timestamp),
		}
	}

	response := &pb.ListLikedYouResponse{
		Likers: pbLikers,
	}

	if nextToken != "" {
		response.NextPaginationToken = &nextToken
	}

	go func() {
		err = s.cache.SetJSON(ctx, key, response, utils.LikersTTL)
		if err != nil {
			s.logger.Warn("Failed to cache likers", zap.Error(err))
		}
	}()

	return response, nil
}

// ListNewLikers returns users who liked the recipient but haven't been liked back
// method try from cache, if not found then query from DB
func (s *exploreCore) ListNewLikers(ctx context.Context, req *pb.ListLikedYouRequest) (*pb.ListLikedYouResponse, error) {
	key := utils.NewLikersKey(req.GetRecipientUserId(), req.GetPaginationToken())

	var cached pb.ListLikedYouResponse
	if ok, err := s.cache.GetJSON(ctx, key, &cached); err == nil && ok {
		return &cached, nil
	}

	likers, nextToken, err := s.repo.GetNewLikers(ctx, req.RecipientUserId, req.GetPaginationToken())
	if err != nil {
		s.logger.Error("Failed to get new likers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get new likers")
	}

	pbLikers := make([]*pb.ListLikedYouResponse_Liker, len(likers))
	for i, liker := range likers {
		pbLikers[i] = &pb.ListLikedYouResponse_Liker{
			ActorId:       liker.ActorID,
			UnixTimestamp: uint64(liker.Timestamp),
		}
	}

	response := &pb.ListLikedYouResponse{
		Likers: pbLikers,
	}

	if nextToken != "" {
		response.NextPaginationToken = &nextToken
	}

	go func() {
		_ = s.cache.SetJSON(ctx, key, response, utils.NewLikersTTL)
	}()
	return response, nil
}

// CountLikers returns the count of users who liked the recipient
// First it try from cache, if not found then query from DB
func (s *exploreCore) CountLikers(ctx context.Context, req *pb.CountLikedYouRequest) (*pb.CountLikedYouResponse, error) {
	key := utils.LikersCountKey(req.GetRecipientUserId())
	if raw, err := s.cache.Get(ctx, key); err == nil && raw != "" {
		if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
			return &pb.CountLikedYouResponse{Count: n}, nil
		}
	}

	count, err := s.repo.CountLikes(ctx, req.RecipientUserId)
	if err != nil {
		s.logger.Error("Failed to count likers", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to count likers")
	}

	go func() {
		_ = s.cache.Set(ctx, key, strconv.FormatInt(count, 10), utils.LikersCountTTL)
	}()

	return &pb.CountLikedYouResponse{
		Count: uint64(count),
	}, nil
}

func (s *exploreCore) CreateDecision(ctx context.Context, req *pb.PutDecisionRequest) (*pb.PutDecisionResponse, error) {
	err := s.repo.CreateDecision(ctx, explorerdb.CreateDecisionParams{
		ActorUserID:     req.ActorUserId,
		RecipientUserID: req.RecipientUserId,
		LikedRecipient:  req.LikedRecipient,
	})
	if err != nil {
		s.logger.Error("Failed to create decision", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create decision")
	}

	// Check for mutual like only if this is a like decision
	var mutualLikes bool
	if req.LikedRecipient {
		hasMutualLike, err := s.repo.HasMutualLike(ctx, explorerdb.HasMutualLikeParams{
			ActorUserID:     req.ActorUserId,
			RecipientUserID: req.RecipientUserId,
		})
		if err != nil {
			s.logger.Error("Failed to check mutual like", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to check mutual like")
		}

		if hasMutualLike != nil && *hasMutualLike {
			mutualLikes = true
		}
	}

	return &pb.PutDecisionResponse{
		MutualLikes: mutualLikes,
	}, nil
}
