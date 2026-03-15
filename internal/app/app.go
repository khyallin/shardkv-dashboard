package app

import "github.com/gin-gonic/gin"

type Application struct {
	r *gin.Engine
}

func New() *Application {
	app := &Application{
		r: gin.Default(),
	}
	registerRoutes(app.r)
	return app
}

func (app *Application) Run() {
	app.r.Run(":8080")
}