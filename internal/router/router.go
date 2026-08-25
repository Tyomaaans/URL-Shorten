package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"url-shorten/internal/middleware"
	shorten "url-shorten/internal/shortens"
	user "url-shorten/internal/users"
)

func NewUserRouter(
	userHandler     *user.UserHandler,
	shortenHandler  *shorten.ShortenHandler,
	authMiddleware  *middleware.AuthMiddleware,
) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	v1 := r.Group("/api/v1")
	{	
		// User Auth
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.RegisterUser)
			auth.POST("/login",    userHandler.LoginUser)
			auth.POST("/refresh",  userHandler.RefreshToken)
			auth.POST("/logout",   authMiddleware.Authenticate(), userHandler.LogoutUser)
		}
		
		// User CRUD
		users := v1.Group("/users/me", authMiddleware.Authenticate())
		{
			users.GET("",    userHandler.GetMyProfile)
			users.PATCH("",  userHandler.UpdateMyProfile)
			users.PUT("",    userHandler.UpdateMyPassword)
			users.DELETE("", userHandler.DeleteMyProfile)
		}

		// User Session
		sessions := v1.Group("users/me/sessions", authMiddleware.Authenticate())
		{
			sessions.GET("",           userHandler.GetActiveMyProfileSessions)
			sessions.DELETE("/:sid",   userHandler.RevokeMyProfileSession)
			sessions.DELETE("/others", userHandler.RevokeAllMyProfileOtherSessions)
			sessions.DELETE("",        userHandler.RevokeAllMyProfileSessions)
		}

		shortens := v1.Group("")
		{
			shortens.POST("/s",      shortenHandler.CreateShorten)
			shortens.GET("/s/:code", shortenHandler.Redirect)

			myShorten := shortens.Group("/shortens/me", authMiddleware.Authenticate())
			{
				myShorten.POST("",            shortenHandler.CreateMyShorten)
				myShorten.PATCH("/:shid",     shortenHandler.UpdateMyShorten)
				myShorten.GET("",             shortenHandler.GetMyShortens)
				myShorten.GET("/:shid",       shortenHandler.GetMyShortenByID)
				myShorten.PUT(":shid/status", shortenHandler.SetURLStatus)
				myShorten.DELETE("/:shid",    shortenHandler.DeleteMyShorten)
			}
		}

		admin := v1.Group("/admin", authMiddleware.Authenticate(), authMiddleware.AdminSecretMiddleware())
		{
			// User CRUD for Admin
			admin.GET("/users",                      userHandler.GetUsers)
			admin.GET("/users/:sub",                 userHandler.GetUserByID)
			admin.PATCH("/users/:sub",               userHandler.UpdateUser)
			admin.PUT("/users/:sub",                 userHandler.UpdatePasswordUser)
			admin.DELETE("/users/:sub",              userHandler.DeleteUser)
			
			// User Session for Admin
			admin.GET("/users/:sub/sessions",        userHandler.GetActiveSessionsUser)
			admin.DELETE("users/:sub/sessions/:sid", userHandler.RevokeSessionUser)
			admin.DELETE("users/:sub/sessions",      userHandler.RevokeAllSessionsUser)

			// User Shorten for Admin
			admin.GET("/shortens",                     shortenHandler.GetUserShortens)
			admin.GET("/shortens/:shid",               shortenHandler.GetUserShortenByID)
			admin.GET("/users/:sub/shortens",          shortenHandler.GetShortensByUserID)
			admin.PATCH("/users/:sub/shortens/:shid",  shortenHandler.UpdateUserShorten)
			admin.DELETE("/users/:sub/shortens/:shid", shortenHandler.DeleteUserShorten)
		}
	}

	return r
}