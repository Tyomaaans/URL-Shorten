package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"url-shorten/internal/auth-session/session"
	"url-shorten/internal/auth-session/token"
	"url-shorten/internal/auth-session/user"
	"url-shorten/internal/config"
	"url-shorten/internal/infrastructure/postgres"
	"url-shorten/internal/infrastructure/redis"
	"url-shorten/internal/middleware"
	"url-shorten/internal/router"
	"url-shorten/internal/url-shortener/shortener"
	shortenerUser "url-shorten/internal/url-shortener/user"
	"url-shorten/pkg"

	_ "url-shorten/docs/url-shortener"
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.NewConfig()

	db, err := postgres.NewPostgresDB(cfg.DatabaseURL, cfg.AdminPassword)
	if err != nil {
		log.Fatalf("failed to connect postgres: %v", err)
	}

	redisClient, err := redis.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer redisClient.Close()

	validate    := pkg.NewValidator()
	
	sessionSvc  := session.NewSessionService(redisClient)
	tokenSvc    := token.NewJWTService(cfg.JWTSecretKey, cfg.JWTExpiry, cfg.DefaultRefreshExpiry, cfg.ShortRefreshExpiry, redisClient, sessionSvc)
	userRepo    := user.NewUserRepository(db)
	userSvc     := user.NewUserService(userRepo, tokenSvc, sessionSvc, validate)
	userHandler := shortenerUser.NewUserHandler(userSvc, cfg.DefaultRefreshExpiry, cfg.ShortRefreshExpiry, cfg.SecureCookies)
	
	shortenRepo    := shortener.NewShortenRepository(db)
	shortenSvc     := shortener.NewShortenService(shortenRepo, redisClient, validate)
	shortenHandler := shortener.NewShortenHandler(shortenSvc)

	authMiddleware := middleware.NewAuthMiddleware(tokenSvc)
	rateLimiter    := middleware.NewRateLimiter(redisClient, logger)
	
	r := router.NewRouter(userHandler, shortenHandler, authMiddleware, rateLimiter)

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	go shortenSvc.StartExpiryWorker(workerCtx)

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Printf("server running on :%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancelWorker()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("postgres close error: %v", err)
		}
	}
}
