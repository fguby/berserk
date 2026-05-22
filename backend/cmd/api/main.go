package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pianke-ticket/backend/internal/config"
	"pianke-ticket/backend/internal/database"
	"pianke-ticket/backend/internal/httpapi"
	"pianke-ticket/backend/internal/store"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("connecting database")
	connectCtx, cancelConnect := context.WithTimeout(ctx, 20*time.Second)
	pool, err := database.Open(connectCtx, cfg.DatabaseURL)
	cancelConnect()
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	logger.Info("checking database schema")
	schemaCtx, cancelSchema := context.WithTimeout(ctx, 2*time.Minute)
	if err := database.EnsureSchema(schemaCtx, pool); err != nil {
		cancelSchema()
		logger.Error("ensure database schema", "error", err)
		os.Exit(1)
	}
	cancelSchema()
	logger.Info("database schema checked")

	ticketStore := store.NewPostgres(pool)
	server := httpapi.NewServer(httpapi.ServerConfig{
		Addr:                   cfg.HTTPAddr,
		PublicBaseURL:          cfg.PublicBaseURL,
		WebAppID:               cfg.WebAppID,
		EmailCodeTTLSeconds:    cfg.EmailCodeTTLSeconds,
		SMTPHost:               cfg.SMTPHost,
		SMTPPort:               cfg.SMTPPort,
		SMTPUsername:           cfg.SMTPUsername,
		SMTPPassword:           cfg.SMTPPassword,
		SMTPFromEmail:          cfg.SMTPFromEmail,
		SMTPFromName:           cfg.SMTPFromName,
		SMTPTLSMode:            cfg.SMTPTLSMode,
		XAIAPIKey:              cfg.XAIAPIKey,
		XAIBaseURL:             cfg.XAIBaseURL,
		XAIResponsesPath:       cfg.XAIResponsesPath,
		XAIMainModel:           cfg.XAIMainModel,
		XAIImageModel:          cfg.XAIImageModel,
		GeneratedImageDir:      cfg.GeneratedImageDir,
		OSSBucket:              cfg.OSSBucket,
		OSSRegion:              cfg.OSSRegion,
		OSSEndpoint:            cfg.OSSEndpoint,
		OSSAccessKeyID:         cfg.OSSAccessKeyID,
		OSSAccessKeySecret:     cfg.OSSAccessKeySecret,
		OSSSecurityToken:       cfg.OSSSecurityToken,
		OSSObjectPrefix:        cfg.OSSObjectPrefix,
		OSSSignedURLTTLSeconds: cfg.OSSSignedURLTTLSeconds,
		R2Bucket:               cfg.R2Bucket,
		R2Endpoint:             cfg.R2Endpoint,
		R2AccessKeyID:          cfg.R2AccessKeyID,
		R2AccessKeySecret:      cfg.R2AccessKeySecret,
		R2ObjectPrefix:         cfg.R2ObjectPrefix,
		R2SignedURLTTLSeconds:  cfg.R2SignedURLTTLSeconds,
		Store:                  ticketStore,
		Logger:                 logger,
	})

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		errCh <- server.Start()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown api", "error", err)
		}
	case err := <-errCh:
		if err != nil {
			logger.Error("api stopped", "error", err)
			os.Exit(1)
		}
	}
}
