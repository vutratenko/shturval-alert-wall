package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/aggregate"
	"github.com/vutratenko/shturval-alert-wall/internal/auth"
	"github.com/vutratenko/shturval-alert-wall/internal/config"
	"github.com/vutratenko/shturval-alert-wall/internal/discovery"
	"github.com/vutratenko/shturval-alert-wall/internal/fake"
	"github.com/vutratenko/shturval-alert-wall/internal/httpapi"
	"github.com/vutratenko/shturval-alert-wall/internal/metrics"
	"github.com/vutratenko/shturval-alert-wall/internal/push"
	"github.com/vutratenko/shturval-alert-wall/internal/source"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed static/*
var content embed.FS

func main() {
	cfg := config.Load()
	var addr string
	flag.StringVar(&addr, "http-bind-address", cfg.HTTPBind, "HTTP listen address")
	flag.Parse()

	static, _ := fs.Sub(content, "static")
	hub := httpapi.NewHub()
	pushStore, err := push.Open(cfg.PushStorePath)
	if err != nil {
		slog.Error("push store", "err", err)
		os.Exit(1)
	}
	fakeSrc := prepareFake(cfg)
	refresh := &discovery.Refreshing{Discoverer: discovery.Discoverer{Static: cfg.StaticSources}}
	api := &httpapi.Server{
		Hub:     hub,
		Static:  static,
		Push:    pushStore,
		Silence: cfg.SilenceAllowed(),
		Sources: func() []source.Source { return buildSources(cfg, refresh, fakeSrc) },
	}
	dispatcher := &push.Dispatcher{
		Store: pushStore,
		Send: (&push.WebSender{
			PublicKey:  pushStore.PublicKey(),
			PrivateKey: pushStore.PrivateKey(),
			Subject:    cfg.PushSubject,
		}).Send,
	}
	handler := http.Handler(api.Handler())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var k8s client.Reader
	needK8s := (cfg.DiscoveryEnabled && cfg.SourceMode != "fake" && cfg.SourceMode != "vmselect") || cfg.Auth.Enabled
	if needK8s {
		restCfg, err := rest.InClusterConfig()
		if err != nil {
			restCfg, err = clientcmdRestConfig()
			if err != nil {
				slog.Warn("k8s client unavailable", "err", err)
			}
		}
		if restCfg != nil {
			c, err := client.New(restCfg, client.Options{})
			if err != nil {
				slog.Warn("k8s client", "err", err)
			} else {
				k8s = c
				refresh.Discoverer.Client = k8s
			}
		}
	}

	if cfg.Auth.Enabled {
		if k8s == nil {
			slog.Error("AUTH_ENABLED requires kubernetes client")
			os.Exit(1)
		}
		if cfg.Auth.RedirectURL == "" {
			slog.Error("AUTH_REDIRECT_URL required when auth enabled")
			os.Exit(1)
		}
		authSvc, err := auth.Bootstrap(ctx, k8s, cfg.Auth.Namespace, cfg.Auth.OIDCClientName, cfg.Auth.RedirectURL, cfg.Auth.SessionTTL, cfg.Auth.CookieSecret)
		if err != nil {
			slog.Error("oidc bootstrap", "err", err)
			os.Exit(1)
		}
		handler = authSvc.Wrap(api.Handler())
		slog.Info("oidc auth enabled", "client", cfg.Auth.OIDCClientName)
	}

	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}

	go runPoller(ctx, cfg, refresh, hub, dispatcher, fakeSrc)
	go func() {
		slog.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func runPoller(ctx context.Context, cfg config.Config, refresh *discovery.Refreshing, hub *httpapi.Hub, alerts *push.Dispatcher, fakeSrc *fake.FileSource) {
	_ = refreshSources(ctx, cfg, refresh)
	if cfg.DiscoveryEnabled {
		go func() {
			t := time.NewTicker(cfg.DiscoveryEvery)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					_ = refreshSources(ctx, cfg, refresh)
				}
			}
		}()
	}
	t := time.NewTicker(cfg.PollInterval)
	defer t.Stop()
	pollOnce(ctx, cfg, refresh, hub, alerts, fakeSrc)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pollOnce(ctx, cfg, refresh, hub, alerts, fakeSrc)
		}
	}
}

func refreshSources(ctx context.Context, cfg config.Config, refresh *discovery.Refreshing) error {
	if cfg.SourceMode == "fake" || cfg.SourceMode == "vmselect" {
		return nil
	}
	return refresh.Refresh(ctx)
}

func pollOnce(ctx context.Context, cfg config.Config, refresh *discovery.Refreshing, hub *httpapi.Hub, alerts *push.Dispatcher, fakeSrc *fake.FileSource) {
	start := time.Now()
	sources := buildSources(cfg, refresh, fakeSrc)
	if len(sources) == 0 {
		slog.Warn("no alert sources configured")
	}
	p := &aggregate.Poller{Sources: sources, Timeout: cfg.PollTimeout, Th: cfg.Mood}
	fetches := pollFetches(ctx, cfg, p)
	for _, f := range fetches {
		v := 0.0
		if f.Reachable {
			v = 1
		}
		metrics.SourceUp.WithLabelValues(f.Name).Set(v)
	}
	snap := aggregate.Build(fetches, time.Now().UTC(), cfg.Mood)
	snap.Silence = cfg.SilenceAllowed()
	events := hub.Update(snap)
	if alerts != nil {
		go alerts.Notify(ctx, events)
	}
	metrics.PollDuration.Observe(time.Since(start).Seconds())
}

func pollFetches(ctx context.Context, cfg config.Config, p *aggregate.Poller) []aggregate.ClusterFetch {
	fetches := p.Poll(ctx)
	if cfg.SourceMode == "vmselect" || cfg.SourceMode == "alertmanager" {
		return aggregate.ExpandByClusterLabel(fetches)
	}
	return fetches
}

func buildSources(cfg config.Config, refresh *discovery.Refreshing, fakeSrc *fake.FileSource) []source.Source {
	switch cfg.SourceMode {
	case "fake":
		if fakeSrc == nil {
			return nil
		}
		return []source.Source{fakeSrc}
	case "vmselect":
		if cfg.VMSelectURL == "" {
			slog.Error("VMSELECT_URL required")
			return nil
		}
		return []source.Source{&source.VMSelect{SelectURL: cfg.VMSelectURL}}
	default:
		return refresh.Snapshot()
	}
}

func prepareFake(cfg config.Config) *fake.FileSource {
	if cfg.SourceMode != "fake" {
		return nil
	}
	path := cfg.FakeAlertsPath
	if path == "" {
		for _, p := range []string{"config/samples/fake-alerts.json", "/config/samples/fake-alerts.json"} {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
		if path == "" {
			path = "config/samples/fake-alerts.json"
		}
	}
	return &fake.FileSource{ClusterName: "demo", Path: path}
}

// clientcmdRestConfig loads kubeconfig when not in-cluster.
func clientcmdRestConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	return kubeCfg.ClientConfig()
}
