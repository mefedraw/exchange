package app

import (
	grpcapp "grpcAuth/internal/app/grpc"
	"log/slog"
	"time"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(
	log *slog.Logger,
	grpcPort int,
	storagePath string,
	tokenTTL time.Duration) *App {
	// TODO: auth storage

	// TODO: init auth service

	grpcApp := grpcapp.New(log, grpcPort)

	return &App{grpcApp}
}
