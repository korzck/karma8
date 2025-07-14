//go:build integration

package test

import (
	"gateway/internal/handlers"
	"gateway/internal/repository"
	"gateway/internal/service"

	"github.com/jmoiron/sqlx"
)

type TestService struct {
	gatewayHandler *handlers.GatewayHandler
	chunkerService *service.ChunkerService
}

func newTestService(db *sqlx.DB) *TestService {
	repository := repository.NewRepository(db)

	chunkerService := service.NewChunkerService(repository, nil)
	gatewayHandler := handlers.NewGatewayHandler(chunkerService)

	return &TestService{
		gatewayHandler: gatewayHandler,
		chunkerService: chunkerService,
	}
}
