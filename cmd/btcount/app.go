package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ferux/btcount/internal/api"
	"github.com/ferux/btcount/internal/btcount"
	"github.com/ferux/btcount/internal/bthttp"
	"github.com/ferux/btcount/internal/btlog"
	"github.com/ferux/btcount/internal/cache"
	"github.com/ferux/btcount/internal/postgres"
	"github.com/ferux/btcount/internal/worker"

	"go.uber.org/zap"
)

func app(ctx context.Context, cfg btcount.Config) (err error) {
	log, err := btlog.NewLog(cfg.LogLevel, btlog.LogFormat(cfg.LogFormat), isDevelopment())
	if err != nil {
		return fmt.Errorf("making new log: %w", err)
	}

	log.Info("app runned",
		zap.String("revision", revision),
		zap.Int("pid", os.Getpid()),
		zap.Bool("development", isDevelopment()),
	)

	var db btcount.Database
	db, err = postgres.Open(ctx, cfg.DBAddr, postgres.Config{
		MinConns: cfg.DBMinConn,
		MaxConns: cfg.DBMaxConn,
	})
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}

	hstore := postgres.NewHistoryStore()
	tstore := postgres.NewTransactionStore()

	httpapi := bthttp.NewServer(bthttp.Config{
		WriteTimeout: cfg.HTTPTimeout,
		ReadTimeout:  cfg.HTTPTimeout,
		IdleTimeout:  cfg.HTTPTimeout,
	}, log)
	httpapi.MountDebug()

	statcache, err := cache.InitHistoryStatCollector(ctx, cache.HistoryStatParams{
		HStore: hstore,
		TStore: tstore,
		DB:     db,
	}, log)
	if err != nil {
		log.Warn("unable to init cache", zap.Error(err))
		statcache = nil
	}
	walletAPI := api.NewWalletAPI(db, hstore, tstore, statcache)
	httpapi.MountWalletAPI(walletAPI)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer panicRecover(log)
		defer wg.Done()
		defer log.Info("http server finished")

		err = httpapi.Run(ctx, cfg.HTTPAddr)
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("unable to run server", zap.Error(err))
		}
	}()

	go func() {
		<-ctx.Done()
		sdctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		err = httpapi.Shutdown(sdctx)
		if err != nil {
			log.Error("shutting down http", zap.Error(err))
		}
	}()

	wg.Add(1)
	go func() {
		defer panicRecover(log)
		defer wg.Done()
		defer log.Info("worker finished")

		wcfg := worker.StatMakerWorkerConfig{
			TStore:     tstore,
			HStore:     hstore,
			DB:         db,
			RetryDelay: cfg.StatWorkerRetryDelay,
		}

		worker.RunStatMakerWorker(ctx, wcfg, log)
	}()

	wg.Wait()
	return nil
}

func panicRecover(log *zap.Logger) {
	if rec := recover(); rec != nil {
		log.Error("panic captured", zap.Any("panic", rec))
		debug.PrintStack()
	}
}
