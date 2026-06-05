package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	// Ensure these match your actual go.mod module name!
	"github.com/adijunek/kine-track-go/internal/api/controllers"
	"github.com/adijunek/kine-track-go/internal/api/middleware"
	"github.com/adijunek/kine-track-go/internal/database"
)

func main() {
	log.Println("🚢 Booting Kine-Track Spatial Engine...")

	if err := godotenv.Load(); err != nil {
		log.Println("ℹ️  No .env file found, relying on system environment variables")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// THIS is what stops VS Code from deleting the database import!
	db := database.ConnectPostGIS()

	// (Optional) We will use db later, but logging it prevents a "declared and not used" error
	log.Printf("Database connected: %v", db != nil)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.SetTrustedProxies([]string{"127.0.0.1"})

	r.StaticFile("/", "./web/index.html")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "engine operational"})
	})

	// Inject the database into the controller
	ingestCtrl := controllers.NewIngestController(db)

	ingestAPI := r.Group("/api/v1/ingest")
	ingestAPI.Use(middleware.APIKeyAuth())
	{
		// Map the POST route directly to your new handler function
		ingestAPI.POST("/batch", ingestCtrl.HandleBatch)
	}

	// Inject the new controller
	mapCtrl := controllers.NewMapController(db)
	publicAPI := r.Group("/api/v1")
	{
		// Map data is public so the frontend can read it without an API key
		publicAPI.GET("/map-data", mapCtrl.GetMapDashboard)
	}

	hostIP := os.Getenv("BIND_IP")
	if hostIP == "" {
		hostIP = "127.0.0.1"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8010"
	}

	srv := &http.Server{
		Addr:    hostIP + ":" + port,
		Handler: r,
	}

	log.Printf("🔒 Spatial Engine listening on %s", srv.Addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Fatal server error: %s\n", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("🛑 Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("✅ Server exited cleanly.")
}
