// @title           Multi-tenant API
// @version         1.0
// @description     Base template: PostgreSQL RLS, JWT auth, clean architecture.
// @host            localhost:5000
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description     Type "Bearer" followed by a space and the access token.
package main

import (
	_ "clean-architecture/docs"
	"clean-architecture/bootstrap"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = bootstrap.RootApp.Execute()
}
