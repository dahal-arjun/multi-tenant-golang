package infrastructure

import (
	"clean-architecture/pkg/framework"
	"net/http"

	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Router -> Gin Router
type Router struct {
	*gin.Engine
}

// NewRouter : all the routes are defined here
func NewRouter(
	env *framework.Env,
	logger framework.Logger,
) Router {

	gin.DefaultWriter = logger.GetGinLogger()
	appEnv := env.Environment
	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	httpRouter := gin.Default()

	httpRouter.MaxMultipartMemory = env.MaxMultipartMemory

	httpRouter.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "POST", "OPTIONS", "DELETE"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))

	// Attach sentry middleware
	httpRouter.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))

	httpRouter.GET("/health-check", healthCheck)

	if appEnv != "production" {
		httpRouter.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	return Router{
		httpRouter,
	}
}

// healthCheck godoc
// @Summary      Health check
// @Description  Liveness probe
// @Tags         system
// @Success      200  {object}  map[string]interface{}
// @Router       /health-check [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": "clean architecture 📺 API Up and Running"})
}
