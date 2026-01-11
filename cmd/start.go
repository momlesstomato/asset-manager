package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"asset-manager/core/config"
	"asset-manager/core/database"
	"asset-manager/core/loader"
	"asset-manager/core/logger"
	"asset-manager/core/middleware/auth"
	"asset-manager/core/middleware/rayid"
	"asset-manager/core/storage"

	"asset-manager/feature/furniture"
	"asset-manager/feature/integrity"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gorm.io/gorm"

	_ "asset-manager/docs/swagger"
)

// @title Asset Manager API
// @version 1.0
// @description API for managing Habbo assets.
// @host localhost:8080
// @BasePath /

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the asset manager server",
	Long:  `Starts the HTTP server and initializes all enabled features.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Load Configuration
		cfg, err := config.LoadConfig(".")
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		// 2. Initialize Logger
		logg, err := logger.New(&cfg.Log)
		if err != nil {
			log.Fatalf("Failed to initialize logger: %v", err)
		}
		defer logg.Sync()
		zap.ReplaceGlobals(logg)

		// 3. Connect to Database (Optional)
		// We use the emulator name as the "server" field.
		var db *gorm.DB
		if conn, err := database.Connect(cfg.Database); err != nil {
			logg.Warn("Optional database connection failed", zap.Error(err))
		} else {
			db = conn
			// If succeeded, inject "server" field into logger
			logg = logg.With(zap.String("server", cfg.Server.Emulator))
			logg.Info("Connected to emulator database")
		}

		// 3. Initialize Fiber App
		app := fiber.New(fiber.Config{
			DisableStartupMessage: true, // We will log our own startup message
		})

		// 3. Initialize Storage
		store, err := storage.NewClient(cfg.Storage)
		if err != nil {
			logg.Fatal("Failed to create storage client", zap.Error(err))
		}

		// 4. Initialize Feature Loader
		mgr := loader.NewManager()

		// Register Features
		mgr.Register(integrity.NewFeature(store, cfg.Storage.Bucket, logg, db, cfg.Server.Emulator))
		mgr.Register(furniture.NewFeature(store, cfg.Storage.Bucket, logg, db, cfg.Server.Emulator))

		// Middleware Registration
		// 1. RayID (Must be first to trace everything)
		app.Use(rayid.New())

		// 2. Logging Middleware (Custom to use Zap + RayID)
		app.Use(func(c *fiber.Ctx) error {
			// Attach logger to locals? or just log request here?
			// Let's log the incoming request
			l := logger.WithRayID(logg, c)
			l.Info("Request started",
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.String("ip", c.IP()),
			)
			err := c.Next()
			// Log error if happened
			if err != nil {
				l.Error("Request error", zap.Error(err))
			}
			return err
		})

		// 2.5 Swagger Documentation (Public)
		app.Get("/swagger/*", swagger.HandlerDefault)

		// 3. Auth (Protect API)
		// We protect everything for now as requested ("protect every request")
		app.Use(auth.New(auth.Config{ApiKey: cfg.Server.ApiKey}))

		// 5. Load Features
		if err := mgr.LoadAll(app); err != nil {
			logg.Fatal("Failed to load features", zap.Error(err))
		}

		// 7. Start Server
		go func() {
			logg.Info("Starting server", zap.String("port", cfg.Server.Port))
			if err := app.Listen(":" + cfg.Server.Port); err != nil {
				logg.Fatal("Server failed to start", zap.Error(err))
			}
		}()

		// 7. Graceful Shutdown
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		logg.Info("Shutting down server...")
		_ = app.Shutdown()
	},
}

func init() {
	RootCmd.AddCommand(startCmd)
}
