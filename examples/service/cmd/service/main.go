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
	"github.com/ruko1202/xlog/xfield"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"golang.org/x/sync/errgroup"

	"github.com/ruko1202/goque"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := config.InitLogger()
	xlog.ReplaceGlobalLogger(logger)
	ctx = xlog.ContextWithLogger(ctx, logger)

	xlog.Info(ctx, "Starting Goque example service")
	cfg, err := config.Load()
	if err != nil {
		xlog.Fatal(ctx, "Failed to load configuration", xfield.Error(err))
	}

	// Configure metrics
	stop := initExporters(ctx, cfg)
	defer stop()

	db := config.NewDB(ctx, cfg.Database.Driver, cfg.Database.DSN)
	defer func() {
		_ = db.Close()
	}()

	storage := initStorage(ctx, db)

	goqueInst := initGoque(cfg, storage)
	if err := goqueInst.Run(ctx); err != nil {
		cancel()
		xlog.Fatal(ctx, "Failed to run goque", xfield.Error(err))
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

	server := initHTTPServer(ctx, application, cfg)
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		xlog.Info(ctx, "Starting HTTP server", xfield.String("address", addr))
		if err := server.Start(addr); err != nil {
			cancel()
			xlog.Fatal(ctx, "Server error", xfield.Error(err))
		}
	}()

	waitForShutdown(ctx, server)
}

func initStorage(ctx context.Context, db *sqlx.DB) goque.TaskStorage {
	storage, err := goque.NewStorage(db)
	if err != nil {
		xlog.Fatal(ctx, "Failed to create storage", xfield.Error(err))
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
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	notificationProcessor := processors.NewNotificationProcessor()
	goqueInst.RegisterProcessor(
		models.TaskTypeNotification,
		notificationProcessor,
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	reportProcessor := processors.NewReportProcessor()
	goqueInst.RegisterProcessor(
		models.TaskTypeReport,
		reportProcessor,
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	webhookProcessor := processors.NewWebhookProcessor()
	goqueInst.RegisterProcessor(
		models.TaskTypeWebhook,
		webhookProcessor,
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	return goqueInst
}

func initHTTPServer(ctx context.Context, application *app.Application, cfg *config.Config) *echo.Echo {
	e := echo.New()
	e.HidePort = true
	e.HideBanner = true

	// Middleware
	logger := xlog.LoggerFromContext(ctx)
	e.Use(middleware.RequestID())
	e.Use(app.XlogMiddleware(logger))
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(otelecho.Middleware(cfg.AppName))

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
		xlog.Error(ctx, "Error shutting down HTTP server", xfield.Error(err))
	}

	xlog.Info(ctx, "Server stopped successfully")
}

func initExporters(ctx context.Context, cfg *config.Config) func() {
	xlog.ReplaceTracerName(cfg.AppName)
	goque.SetMetricsServiceName(cfg.AppName)

	otelRes, err := config.InitTracerResource(cfg)
	if err != nil {
		xlog.Panic(ctx, "failed to initialize OpenTelemetry", xfield.Error(err))
	}

	tracerProvider, err := config.InitTracerProvider(ctx, cfg, otelRes)
	if err != nil {
		xlog.Panic(ctx, "failed to initialize OpenTelemetry", xfield.Error(err))
	}

	meterProvider, err := config.InitMetricExporter(otelRes)
	if err != nil {
		xlog.Panic(ctx, "failed to initialize OpenTelemetry metric exporter", xfield.Error(err))
	}

	return func() {
		eg := errgroup.Group{}
		eg.Go(func() error {
			return tracerProvider.Shutdown(ctx)
		})
		eg.Go(func() error {
			return meterProvider.Shutdown(ctx)
		})

		_ = eg.Wait()
	}
}
