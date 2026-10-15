package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hitraa/siyi-sdk-go/pkg/protocol"
	"github.com/hitraa/siyi-sdk-go/web"
)

func main() {
	port := flag.Int("port", 8082, "Web UI HTTP listening port")
	cameraIP := flag.String("ip", protocol.DefaultIP, "Target SIYI gimbal camera IP address")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	srv := web.NewServer(web.Config{
		Port:     *port,
		CameraIP: *cameraIP,
		Logger:   logger,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		if err := srv.Start(ctx); err != nil {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	logger.Info("SIYI SDK Web UI running", "url", "http://localhost:"+flag.Lookup("port").Value.String(), "camera_ip", *cameraIP)

	<-ctx.Done()
	logger.Info("shutting down server...")
	_ = srv.Stop(context.Background())
}
