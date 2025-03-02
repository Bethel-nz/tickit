package handlers

import (
	"net/http"

	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/internal/env"
)

func HealthCheck(c *router.Context) {
	c.JSON(http.StatusOK, map[string]string{
		"status":      "healthy",
		"version":     "1.0.0",
		"environment": env.String("Environment", "development", env.Optional).Get(),
	})
}
