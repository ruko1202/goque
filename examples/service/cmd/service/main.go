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
	"github.com/robfig/cron/v3"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/ruko1202/goque"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure logger
	logger := config.InitLogger()
	xlog.ReplaceGlobalLogger(logger)
	ctx = xlog.ContextWithLogger(ctx, logger)

	xlog.Info(ctx, "Starting Goque example service")
	cfg, err := config.Load()
	if err != nil {
		xlog.Fatal(ctx, "Failed to load configuration", xfield.Error(err))
	}

	// Configure metrics
	xlog.ReplaceTracerName(cfg.AppName)
	otelRes, err := config.InitTracerResource(cfg)
	if err != nil {
		xlog.Panic(ctx, "failed to initialize OpenTelemetry", xfield.Error(err))
	}

	tracerProvider, err := config.InitTracerProvider(ctx, cfg, otelRes)
	if err != nil {
		xlog.Panic(ctx, "failed to initialize OpenTelemetry", xfield.Error(err))
	}
	defer func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			xlog.Error(ctx, "failed to shutdown tracer provider", xfield.Error(err))
		}
	}()

	meterProvider, err := config.InitMetricExporter(otelRes)
	if err != nil {
		xlog.Panic(ctx, "failed to initialize OpenTelemetry metric exporter", xfield.Error(err))
	}
	defer func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			xlog.Error(ctx, "failed to shutdown meter provider", xfield.Error(err))
		}
	}()

	// Configure db
	db := config.NewDB(ctx, cfg.Database.Driver, cfg.Database.DSN)
	defer func() {
		_ = db.Close()
	}()

	storage := initStorage(ctx, db)

	// Configure goque
	goque.SetTracerProvider(tracerProvider)
	goque.SetMetricsServiceName(cfg.AppName)
	queueManager := goque.NewTaskQueueManager(storage)
	goqueInst := initGoque(ctx, cfg, storage, queueManager)
	if err := goqueInst.Run(ctx); err != nil {
		cancel()
		xlog.Fatal(ctx, "Failed to run goque", xfield.Error(err))
	}
	defer goqueInst.Stop()

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

func initGoque(
	ctx context.Context,
	cfg *config.Config,
	storage goque.TaskStorage,
	queueManager goque.TaskQueueManager,
) *goque.Goque {
	goqueInst := goque.NewGoque(storage)

	goqueInst.RegisterProcessor(
		models.TaskTypeEmail,
		goque.NewTypedTaskProcessor[models.EmailPayload](
			processors.NewEmailProcessor(),
			goque.WithCancelTaskWhenPayloadDecodeError[models.EmailPayload](),
		),
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	goqueInst.RegisterProcessor(
		models.TaskTypeNotification,
		processors.NewNotificationProcessor(),
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	goqueInst.RegisterProcessor(
		models.TaskTypeReport,
		processors.NewReportProcessor(),
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	goqueInst.RegisterProcessor(
		models.TaskTypeWebhook,
		processors.NewWebhookProcessor(),
		goque.WithWorkersCount(cfg.Queue.Workers),
		goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
		goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
	)

	if cfg.TaskGenerator.Enabled {
		goqueInst.RegisterProcessor(
			models.TaskTypeTaskGenerator,
			worker.NewTaskGenerator(cfg.TaskGenerator, queueManager, []models.TaskType{
				models.TaskTypeEmail,
				models.TaskTypeNotification,
				models.TaskTypeReport,
				models.TaskTypeWebhook,
			}),
			goque.WithWorkersCount(10),
			goque.WithTaskProcessingMaxAttempts(cfg.Queue.MaxAttempts),
			goque.WithTaskProcessingTimeout(cfg.Queue.TaskTimeout),
		)

		cronParser := cron.NewParser(
			cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		)
		schedule, err := cronParser.Parse(cfg.TaskGenerator.CronSpec)
		if err != nil {
			xlog.Fatal(ctx, "Failed to create periodic job", xfield.Error(err))
		}

		periodicJob, err := goque.NewPeriodicJob(
			"periodic_job_task_generator",
			schedule,
			func(context.Context) (*goque.Task, error) {
				return goque.NewTask(models.TaskTypeTaskGenerator, goque.NoTaskPayload), nil
			},
			goque.WithPeriodicJobRunOnStart(),
		)
		if err != nil {
			xlog.Fatal(ctx, "Failed to create periodic job", xfield.Error(err))
		}
		goqueInst.RegisterPeriodicJob(periodicJob)
	}

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
	e.Use(middleware.RequestLogger())
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
