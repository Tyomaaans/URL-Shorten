package router

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"url-shorten/internal/auth-session/domain"
	"url-shorten/internal/middleware"
	"url-shorten/internal/url-shortener/shortener"
	"url-shorten/internal/url-shortener/user"

	_ "url-shorten/docs/url-shortener"
)

func NewRouter(
	userHandler    *user.UserShortenerHandler,
	shortenHandler *shortener.ShortenHandler,
	authMiddleware *middleware.AuthMiddleware,
	rateLimiter    *middleware.RateLimiter,
) *gin.Engine {
	r     := gin.Default()
	limit := rateLimiter.Limit

	r.GET("/swagger/url-shortener/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.InstanceName("URLShortener"),
		ginSwagger.URL("/swagger/url-shortener/doc.json"),
	))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:4173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PATCH", "PUT", "DELETE"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "X-RateLimit-Policy", "X-RateLimit-Status"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/v1/url-shortener")
	{
		policies := publicShortenCreatePolicies()
		v1.POST("/s", limit(policies[0]), limit(policies[1]), shortenHandler.CreateShorten)
		v1.GET("s/:code", limit(bucket("url-shortener.public.redirect", 180, time.Second/3, middleware.IdentityIP, true)), shortenHandler.Redirect)

		myShorten := v1.Group("shortens/me", authMiddleware.Authenticate())
		{
			myShorten.POST("", limit(bucket("url-shortener.mine.create", 60, 10*time.Second, middleware.IdentityUser, false)), shortenHandler.CreateMyShorten)
			myShorten.GET("", limit(bucket("url-shortener.mine.list", 180, time.Second/3, middleware.IdentityUser, true)), shortenHandler.GetMyShortens)
			myShorten.GET("/:shid", limit(bucket("url-shortener.mine.read", 180, time.Second/3, middleware.IdentityUser, true)), shortenHandler.GetMyShortenByID)
			myShorten.PATCH("/:shid", limit(bucket("url-shortener.mine.update", 60, 5*time.Second, middleware.IdentityUser, false)), shortenHandler.UpdateMyShorten)
			myShorten.PUT("/:shid", limit(bucket("url-shortener.mine.status", 60, 5*time.Second, middleware.IdentityUser, false)), shortenHandler.SetMyShortenStatus)
			myShorten.DELETE("/:shid", limit(bucket("url-shortener.mine.delete", 30, 10*time.Second, middleware.IdentityUser, false)), shortenHandler.DeleteMyShorten)
		}

		auth := v1.Group("/auth")
		{
			auth.POST("/register", limit(bucket("auth.register", 3, 2*time.Hour, middleware.IdentityIP, false)), userHandler.RegisterUser)
			auth.POST("/login", limit(bucket("auth.login", 10, time.Minute, middleware.IdentityIP, false)), userHandler.LoginUser)
			auth.POST("/refresh", limit(bucket("auth.refresh", 20, 15*time.Second, middleware.IdentityIP, false)), userHandler.RefreshToken)
			auth.POST("/heartbeat", limit(bucket("auth.heartbeat", 60, 10*time.Second, middleware.IdentityIP, true)), userHandler.Heartbeat)
			auth.POST("/logout", authMiddleware.Authenticate(), limit(bucket("auth.logout", 20, time.Minute, middleware.IdentityUser, true)), userHandler.LogoutUser)
		}

		users := v1.Group("/users/me", authMiddleware.Authenticate())
		{
			users.GET("", limit(bucket("auth.profile.read", 120, 500*time.Millisecond, middleware.IdentityUser, true)), userHandler.GetMyProfile)
			users.PATCH("", limit(bucket("auth.profile.update", 20, 30*time.Second, middleware.IdentityUser, false)), userHandler.UpdateMyProfile)
			users.DELETE("", limit(bucket("auth.profile.delete", 3, time.Hour, middleware.IdentityUser, false)), userHandler.DeleteMyProfile)
		}

		sessions := v1.Group("/users/me/sessions", authMiddleware.Authenticate())
		{
			sessions.GET("", limit(bucket("auth.sessions.read", 120, time.Second, middleware.IdentityUser, true)), userHandler.GetActiveSessionsMyProfile)
			sessions.DELETE("/:sid", limit(bucket("auth.sessions.revoke-one", 30, 10*time.Second, middleware.IdentityUser, false)), userHandler.RevokeSessionMyProfile)
			sessions.DELETE("/others", limit(bucket("auth.sessions.revoke-others", 10, time.Minute, middleware.IdentityUser, false)), userHandler.RevokeAllOtherSessionsMyProfile)
			sessions.DELETE("", limit(bucket("auth.sessions.revoke-all", 10, time.Minute, middleware.IdentityUser, false)), userHandler.RevokeAllSessionsMyProfile)
		}

		admin := v1.Group("/admin", authMiddleware.Authenticate(), authMiddleware.RequireRole(string(domain.Admin)))
		{
			admin.GET("/users", limit(adminRead("admin.users.list")), userHandler.GetUsers)
			admin.GET("/users/:sub", limit(adminRead("admin.users.read")), userHandler.GetUserByID)
			admin.PATCH("/users/:sub", limit(adminWrite("admin.users.update")), userHandler.UpdateUser)
			admin.DELETE("/users/:sub", limit(adminWrite("admin.users.delete")), userHandler.DeleteUserProfile)
			admin.GET("/users/:sub/sessions", limit(adminRead("admin.users.sessions.read")), userHandler.GetActiveSessionsUser)
			admin.DELETE("users/:sub/sessions/:sid", limit(adminWrite("admin.users.sessions.revoke-one")), userHandler.RevokeSessionUser)
			admin.DELETE("users/:sub/sessions", limit(adminWrite("admin.users.sessions.revoke-all")), userHandler.RevokeAllSessionsUser)

			admin.GET("/shortens", limit(adminRead("admin.shortens.list")), shortenHandler.GetUserShortens)
			admin.GET("/shortens/:shid", limit(adminRead("admin.shortens.read")), shortenHandler.GetUserShortenByID)
			admin.GET("/users/:sub/shortens", limit(adminRead("admin.users.shortens.list")), shortenHandler.GetShortensByUserID)
			admin.PATCH("/users/:sub/shortens/:shid", limit(adminWrite("admin.users.shortens.update")), shortenHandler.UpdateUserShorten)
			admin.PUT("/users/:sub/shortens/:shid", limit(adminWrite("admin.users.shortens.status")), shortenHandler.SetUserShortenStatus)
			admin.DELETE("/users/:sub/shortens/:shid", limit(adminWrite("admin.users.shortens.delete")), shortenHandler.DeleteUserShorten)
		}
	}

	return r
}

func publicShortenCreatePolicies() []middleware.RateLimitPolicy {
	return []middleware.RateLimitPolicy{
		middleware.SlidingPolicy("url-shortener.public.create-ip-abuse", 25, 72*time.Hour, middleware.IdentityIP, false),
		middleware.SlidingPolicy("url-shortener.public.create", 5, 72*time.Hour, middleware.IdentityIPUserAgent, false),
	}
}

func bucket(name string, capacity int, refillEvery time.Duration, identity middleware.RateLimitIdentity, failOpen bool) middleware.RateLimitPolicy {
	return middleware.BucketPolicy(name, capacity, 1/refillEvery.Seconds(), identity, failOpen)
}

func adminRead(name string) middleware.RateLimitPolicy {
	return bucket(name, 300, 200*time.Millisecond, middleware.IdentityUser, true)
}

func adminWrite(name string) middleware.RateLimitPolicy {
	return bucket(name, 100, 500*time.Millisecond, middleware.IdentityUser, false)
}
