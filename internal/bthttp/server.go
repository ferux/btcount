package bthttp

import (
	"context"
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/ferux/btcount/internal/api"
	"github.com/ferux/btcount/internal/btcount"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Config struct {
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	IdleTimeout  time.Duration
}

type Server struct {
	mux        *mux.Router
	httpserver *http.Server
}

// NewServer creates a new server.
func NewServer(cfg Config, log *zap.Logger) *Server {
	mux := mux.NewRouter()
	mux.Use(
		middlewareRequestID,
		middlewareServer,
		middlewareLogging(log),
	)

	mux.NotFoundHandler = rootHandler()

	httpserver := &http.Server{
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	srv := &Server{
		mux:        mux,
		httpserver: httpserver,
	}

	return srv
}

// MountWalletAPI mounts wallet API for handling requests related to
// the wallet.
func (srv *Server) MountWalletAPI(wapi api.WalletAPI) {
	v1 := srv.mux.PathPrefix("/api/v1").Subrouter()

	v1.Handle("/wallet/transaction", saveTransaction(wapi)).
		Methods(http.MethodPost)

	v1.Handle("/wallet/history", getHistory(wapi)).
		Methods(http.MethodPost)
}

// MountDebug mounts debug related handlers.
func (srv *Server) MountDebug() {
	root := srv.mux

	root.Handle("/debug/vars", expvar.Handler()).Methods(http.MethodGet)
	root.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index)).Methods(http.MethodGet)
	root.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline)).Methods(http.MethodGet)
	root.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile)).Methods(http.MethodGet)
	root.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol)).Methods(http.MethodGet)
	root.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace)).Methods(http.MethodGet)
	root.Handle("/debug/pprof/allocs", pprof.Handler("allocs")).Methods(http.MethodGet)
	root.Handle("/debug/pprof/block", pprof.Handler("block")).Methods(http.MethodGet)
	root.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine")).Methods(http.MethodGet)
	root.Handle("/debug/pprof/heap", pprof.Handler("heap")).Methods(http.MethodGet)
	root.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate")).Methods(http.MethodGet)
	root.Handle("/debug/pprof/mutex", pprof.Handler("mutex")).Methods(http.MethodGet)
}

// Run starts the server and blocks until the server closed.
func (srv *Server) Run(_ context.Context, addr string) (err error) {
	if srv.httpserver == nil {
		return btcount.ErrServerNotInited
	}

	srv.httpserver.Addr = addr
	srv.httpserver.Handler = srv.mux

	err = srv.httpserver.ListenAndServe()
	if err != nil {
		return fmt.Errorf("starting http server: %w", err)
	}

	return nil
}

// Shutdown gracefuly shutdows the server.
func (srv *Server) Shutdown(ctx context.Context) (err error) {
	if srv.httpserver == nil {
		return nil
	}

	return srv.httpserver.Shutdown(ctx)
}
