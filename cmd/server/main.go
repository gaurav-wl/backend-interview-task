package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/backend-interview-task/config"
	"github.com/backend-interview-task/internal/core"
	"github.com/backend-interview-task/internal/providers/cache"
	"github.com/backend-interview-task/internal/providers/database"
	"github.com/backend-interview-task/internal/repository"
	"github.com/backend-interview-task/internal/service"
	pb "github.com/backend-interview-task/proto"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}
	logger, err := initLogger(cfg.Logger)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Explore Service",
		zap.String("version", "1.0.0"),
		zap.String("host", cfg.Server.Host),
		zap.String("port", cfg.Server.Port))

	pgxPool, err := database.NewDBProvider(cfg.Database, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer pgxPool.Close()

	database.RunMigrations(cfg.Database)

	cacheProvider, err := cache.NewRedisCacheProvider(context.Background(), cfg.Redis.Address, cfg.Redis.Password, logger)
	if err != nil {
		logger.Warn("Failed to initialize redis cache", zap.Error(err))
	}

	// Initialize repositories
	repo := repository.NewExplorerRepository(pgxPool, logger)

	// Initialize cores
	exploreCore := core.NewExploreCore(repo, cacheProvider, logger)

	// Initialize gRPC services
	exploreService := service.NewExploreService(exploreCore, logger)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryLoggingInterceptor(logger)),
	)
	pb.RegisterExploreServiceServer(grpcServer, exploreService)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("explore.ExploreService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)
	address := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatal("Failed to listen", zap.String("address", address), zap.Error(err))
	}

	go func() {
		logger.Info("gRPC server starting", zap.String("address", address))
		if err := grpcServer.Serve(listener); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down gracefully...")

	// Graceful shutdown
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	grpcServer.GracefulStop()

	logger.Info("Server shutdown complete")
}

// initLogger initializes the logger based on configuration
func initLogger(cfg config.LoggerConfig) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.Set(cfg.Level); err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

// unaryLoggingInterceptor is a gRPC interceptor for logging unary RPCs
func unaryLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
		}

		if err != nil {
			fields = append(fields, zap.Error(err))
			logger.Error("gRPC call failed", fields...)
		} else {
			logger.Info("gRPC call completed", fields...)
		}

		return resp, err
	}
}
