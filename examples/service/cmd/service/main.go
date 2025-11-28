package main

import (
	"context"
	"example/internal/app"
	"example/internal/config"
	"example/internal/models"
	"example/internal/processors"
	"example/internal/worker"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = xlog.ContextWithLogger(ctx, config.InitLogger())

	xlog.Info(ctx, "Starting Goque example service")
	cfg, err := config.Load()
	if err != nil {
		xlog.Fatal(ctx, "Failed to load configuration", zap.Error(err))
	}

	db := config.NewDB(ctx, cfg.Database.Driver, cfg.Database.DSN)
	defer db.Close()

	storage := initStorage(ctx, db)

	goqueInst := initGoque(cfg, storage)
	if err := goqueInst.Run(ctx); err != nil {
		cancel()
		xlog.Fatal(ctx, "Failed to run goque", zap.Error(err))
	}

	queueManager := goque.NewTaskQueueManager(storage)
	if cfg.TaskGenerator.Enabled {
		generator := worker.NewTaskGenerator(cfg.TaskGenerator, queueManager, []models.TaskType{
			models.TaskTypeEmail,
			models.TaskTypeNotification,
			models.TaskTypeReport,
			models.TaskTypeWebhook,
		})
		go generator.Run(ctx)
	}

	application := app.New(cfg, queueManager)

	server := initHTTPServer(application)
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		xlog.Info(ctx, "Starting HTTP server", zap.String("address", addr))
		if err := server.Start(addr); err != nil {
			cancel()
			xlog.Fatal(ctx, "Server error", zap.Error(err))
		}
	}()

	waitForShutdown(ctx, server)
}

func initStorage(ctx context.Context, db *sqlx.DB) goque.TaskStorage {
	storage, err := goque.NewStorage(db)
	if err != nil {
		xlog.Fatal(ctx, "Failed to create storage", zap.Error(err))
	}
	return storage
}

func initGoque(cfg *config.Config, storage goque.TaskStorage) *goque.Goque {
	goqueInst := goque.NewGoque(storage)

	emailProcessor := processors.NewEmailProcessor()
	goqueInst.RegisterProcessor(
		models.TaskTypeEmail,
		emailProcessor,
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(int32(cfg.Queue.MaxAttempts)),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	notificationProcessor := processors.NewNotificationProcessor()
	goqueInst.RegisterProcessor(
		models.TaskTypeNotification,
		notificationProcessor,
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(int32(cfg.Queue.MaxAttempts)),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	reportProcessor := processors.NewReportProcessor()
	goqueInst.RegisterProcessor(
		models.TaskTypeReport,
		reportProcessor,
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(int32(cfg.Queue.MaxAttempts)),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	webhookProcessor := processors.NewWebhookProcessor()
	goqueInst.RegisterProcessor(
		models.TaskTypeWebhook,
		webhookProcessor,
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(int32(cfg.Queue.MaxAttempts)),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	return goqueInst
}

func initHTTPServer(application *app.Application) *echo.Echo {
	e := echo.New()
	e.HidePort = true
	e.HideBanner = true

	// Middleware
	logger := xlog.LoggerFromContext(context.Background())
	e.Use(middleware.RequestID())
	e.Use(app.XlogMiddleware(logger))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Setup routes
	app.SetupRoutes(e, application)

	return e
}

func waitForShutdown(ctx context.Context, server *echo.Echo) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	xlog.Info(ctx, "Shutting down server...")

	if err := server.Shutdown(ctx); err != nil {
		xlog.Error(ctx, "Error shutting down HTTP server", zap.Error(err))
	}

	xlog.Info(ctx, "Server stopped successfully")
}
