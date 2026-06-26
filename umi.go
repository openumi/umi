package umi

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type App interface {
	Start() error
	Stop() error
}

func Run(cfg *Config) error {
	ctx, err := provisionContext(cfg)
	if err != nil {
		return err
	}

	started := make([]string, 0, len(ctx.cfg.apps))
	for name, a := range ctx.cfg.apps {
		if err := a.Start(); err != nil {
			for _, otherName := range started {
				_ = ctx.cfg.apps[otherName].Stop()
			}
			return fmt.Errorf("%s app module: start: %v", name, err)
		}
		started = append(started, name)
	}

	fmt.Println("[umi] all apps started successfully")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n[umi] shutting down...")

	for i := len(started); i >= 0; i-- {
		name := started[i]
		_ = ctx.cfg.apps[name].Stop()
	}

	ctx.cancelFunc(fmt.Errorf("shutting down"))
	return nil
}

func provisionContext(cfg *Config) (Context, error) {
	if cfg == nil {
		cfg = new(Config)
	}

	ctx, cancelCause := NewContextWithCause(Context{Context: context.Background(), cfg: cfg})
	cfg.cancelFunc = cancelCause

	cfg.apps = make(map[string]App)

	for appName := range cfg.AppsRaw {
		if _, err := ctx.App(appName); err != nil {
			cancelCause(fmt.Errorf("provision error: %w", err))
		}
	}

	return ctx, nil
}
