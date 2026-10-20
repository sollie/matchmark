package main

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type TLSMode string

const (
	TLSPlain  TLSMode = "plain"
	TLSStatic TLSMode = "static"
	TLSACME   TLSMode = "acme"
)

type Config struct {
	AdminPassword string
	DBPath        string
	ListenAddr    string
	BaseURL       string
	TrustedProxy  []netip.Prefix
	TLSCertFile   string
	TLSKeyFile    string
	ACMEDomain    string
	ACMEEmail     string
	TLSMode       TLSMode

	DisableCompression bool

	ReplicationURL      string
	ReplicationInterval time.Duration
}

func LoadConfig(getenv func(string) string) (Config, error) {
	c := Config{
		AdminPassword: getenv("ADMIN_PASSWORD"),
		DBPath:        getenv("DB_PATH"),
		ListenAddr:    getenv("LISTEN_ADDR"),
		BaseURL:       getenv("BASE_URL"),
		TLSCertFile:   getenv("TLS_CERT_FILE"),
		TLSKeyFile:    getenv("TLS_KEY_FILE"),
		ACMEDomain:    getenv("ACME_DOMAIN"),
		ACMEEmail:     getenv("ACME_EMAIL"),

		ReplicationURL:      getenv("REPLICATION_URL"),
		ReplicationInterval: time.Minute,
	}
	if raw := getenv("REPLICATION_INTERVAL"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("REPLICATION_INTERVAL: invalid positive duration %q", raw)
		}
		c.ReplicationInterval = d
	}
	if c.DBPath == "" {
		c.DBPath = "data/matchmark.db"
	}
	proxies, err := parseTrustedProxies(getenv("TRUSTED_PROXY"))
	if err != nil {
		return Config{}, err
	}
	c.TrustedProxy = proxies
	disable, err := parseBoolEnv(getenv("DISABLE_COMPRESSION"))
	if err != nil {
		return Config{}, fmt.Errorf("DISABLE_COMPRESSION: %w", err)
	}
	c.DisableCompression = disable
	c.TLSMode, err = selectTLSMode(c)
	if err != nil {
		return Config{}, err
	}
	if c.ListenAddr == "" {
		c.ListenAddr = defaultListenAddr(c.TLSMode)
	}
	return c, nil
}

func selectTLSMode(c Config) (TLSMode, error) {
	static := c.TLSCertFile != "" || c.TLSKeyFile != ""
	acme := c.ACMEDomain != ""
	switch {
	case static && acme:
		return "", errors.New("TLS_CERT_FILE/TLS_KEY_FILE and ACME_DOMAIN are mutually exclusive")
	case static:
		if c.TLSCertFile == "" || c.TLSKeyFile == "" {
			return "", errors.New("TLS_CERT_FILE and TLS_KEY_FILE must be set together")
		}
		return TLSStatic, nil
	case acme:
		if c.ACMEEmail == "" {
			return "", errors.New("ACME_EMAIL is required with ACME_DOMAIN")
		}
		return TLSACME, nil
	}
	return TLSPlain, nil
}

func defaultListenAddr(mode TLSMode) string {
	if mode == TLSPlain {
		return ":8080"
	}
	return ":8443"
}

func parseTrustedProxies(raw string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if p, err := netip.ParsePrefix(part); err == nil {
			out = append(out, p.Masked())
			continue
		}
		addr, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXY: invalid address or CIDR %q", part)
		}
		out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
	}
	return out, nil
}

func peerAddr(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func parseBoolEnv(raw string) (bool, error) {
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}
