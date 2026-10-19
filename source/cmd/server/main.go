package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"log"
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
	"github.com/sollie/matchmark/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("matchmark: %v", err)
	}
}

func run() error {
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
	auth, err := service.NewAdminAuth(repo, cfg.AdminPassword)
	if err != nil {
		return err
	}
	api := handler.NewRouter(service.New(repo, svcOpts...),
		handler.WithAdminAuth(auth, cfg.BaseURL),
		handler.WithBackupDir(filepath.Dir(cfg.DBPath)),
	)
	root := trustedProxyRealIP(cfg.TrustedProxy)(spaHandler(web.Dist(), api))
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
	replicator := service.NewReplicator(syncDB, cfg.ReplicationInterval, log.Printf)
	go replicator.Run(ctx)
	log.Printf("replication enabled: pushing to %s every %s", cfg.ReplicationURL, cfg.ReplicationInterval)
	return conn, []service.Option{service.WithReplicator(replicator)}, nil
}

func serve(ctx context.Context, srv *http.Server, cfg Config) error {
	switch cfg.TLSMode {
	case TLSStatic:
		log.Printf("listening on %s (static certificate)", srv.Addr)
		return srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	case TLSACME:
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
		log.Printf("listening on %s (ACME for %s)", srv.Addr, cfg.ACMEDomain)
		return srv.ListenAndServeTLS("", "")
	}
	log.Printf("listening on %s (plain HTTP)", srv.Addr)
	return srv.ListenAndServe()
}
