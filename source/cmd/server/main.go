package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/caddyserver/certmagic"

	"github.com/sollie/matchmark/internal/db"
	"github.com/sollie/matchmark/internal/handler"
	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/service"
	"github.com/sollie/matchmark/internal/wa1500"
	"github.com/sollie/matchmark/web"
)

func main() {
	if isVersionRequest(os.Args[1:]) {
		printVersion(os.Stdout)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "restore" {
		if err := runRestore(os.Args[2:], os.Getenv, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "restore:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "reset-password" {
		if err := runResetPassword(os.Args[2:], os.Getenv, os.Stdin, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "reset-password:", err)
			os.Exit(1)
		}
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger.Info("starting matchmark", "version", version, "commit", commit, "built", date)
	if err := run(logger); err != nil {
		logger.Error("matchmark exited", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := LoadConfig(os.Getenv)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o750); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	conn, svcOpts, err := openDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	repo := repository.New(conn)
	audit := service.NewAudit(repo, logger)
	auth := service.NewAuth(repo, service.WithAuthAudit(audit))
	if err := auth.SeedAdmin(cfg.AdminUsername, cfg.AdminPassword); err != nil {
		return err
	}
	svcOpts = append(svcOpts, service.WithScoreLog(logger), service.WithAudit(audit))
	api := handler.NewRouter(service.New(repo, svcOpts...),
		handler.WithAuth(auth, cfg.BaseURL),
		handler.WithVersion(version),
		handler.WithTLS(cfg.TLSMode != TLSPlain),
		handler.WithAudit(audit),
		handler.WithBackupDir(filepath.Dir(cfg.DBPath)),
		handler.WithWA1500(service.NewWA1500Lookup(wa1500.NewClient())),
	)
	root := trustedProxyRealIP(cfg.TrustedProxy)(requestLogger(logger)(compression(!cfg.DisableCompression)(spaHandler(web.Dist(), api))))
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errc := make(chan error, 1)
	go func() { errc <- serve(ctx, srv, cfg) }()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func openDatabase(ctx context.Context, cfg Config) (*sql.DB, []service.Option, error) {
	if cfg.ReplicationURL == "" {
		conn, err := db.Open(cfg.DBPath)
		return conn, nil, err
	}
	conn, syncDB, err := db.OpenReplicated(ctx, cfg.DBPath, cfg.ReplicationURL)
	if err != nil {
		return nil, nil, err
	}
	replicator := service.NewReplicator(syncDB, cfg.ReplicationInterval, logf)
	go replicator.Run(ctx)
	slog.Info("replication enabled", "url", cfg.ReplicationURL, "interval", cfg.ReplicationInterval.String())
	return conn, []service.Option{service.WithReplicator(replicator)}, nil
}

func serve(ctx context.Context, srv *http.Server, cfg Config) error {
	switch cfg.TLSMode {
	case TLSStatic:
		slog.Info("listening", "addr", srv.Addr, "tls", "static")
		return srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	case TLSACME:
		certmagic.Default.Logger = certmagicLogger()
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = cfg.ACMEEmail
		certmagic.DefaultACME.AltHTTPPort = 8080
		certmagic.DefaultACME.DisableTLSALPNChallenge = true
		certmagic.Default.Storage = &certmagic.FileStorage{Path: filepath.Join(filepath.Dir(cfg.DBPath), "certs")}
		magic := certmagic.NewDefault()
		if err := magic.ManageSync(ctx, []string{cfg.ACMEDomain}); err != nil {
			return fmt.Errorf("acme: %w", err)
		}
		srv.TLSConfig = magic.TLSConfig()
		srv.TLSConfig.MinVersion = tls.VersionTLS12
		slog.Info("listening", "addr", srv.Addr, "tls", "acme", "domain", cfg.ACMEDomain)
		return srv.ListenAndServeTLS("", "")
	}
	slog.Info("listening", "addr", srv.Addr, "tls", "none")
	return srv.ListenAndServe()
}
