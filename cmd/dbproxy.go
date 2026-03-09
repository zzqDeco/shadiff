package cmd

import (
	"context"
	"fmt"
	"strings"

	"shadiff/internal/capture"
	"shadiff/internal/capture/dbhook"
	"shadiff/internal/config"
)

var newDBHook = dbhook.NewHook

func resolveRecordDBProxies(flagChanged bool, flagValues []string, cfg *config.AppConfig) ([]config.DBProxyConfig, error) {
	if !flagChanged {
		return append([]config.DBProxyConfig(nil), cfg.Capture.DBProxies...), nil
	}

	proxies := make([]config.DBProxyConfig, 0, len(flagValues))
	for _, spec := range flagValues {
		proxy, err := parseDBProxySpec(spec)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

func parseDBProxySpec(spec string) (config.DBProxyConfig, error) {
	parts := strings.SplitN(spec, "://", 2)
	if len(parts) != 2 {
		return config.DBProxyConfig{}, fmt.Errorf("invalid db proxy %q", spec)
	}
	addrs := strings.SplitN(parts[1], "->", 2)
	if len(addrs) != 2 {
		return config.DBProxyConfig{}, fmt.Errorf("invalid db proxy %q", spec)
	}
	return config.DBProxyConfig{
		Type:       parts[0],
		ListenAddr: addrs[0],
		TargetAddr: addrs[1],
	}, nil
}

func startDBHooks(ctx context.Context, recorder *capture.Recorder, proxies []config.DBProxyConfig) ([]dbhook.DBHook, error) {
	hooks := make([]dbhook.DBHook, 0, len(proxies))
	for _, proxy := range proxies {
		hook, err := newDBHook(dbhook.Config{
			DBType:     proxy.Type,
			ListenAddr: proxy.ListenAddr,
			TargetAddr: proxy.TargetAddr,
		})
		if err != nil {
			stopDBHooks(hooks)
			return nil, err
		}
		if err := hook.Start(ctx); err != nil {
			stopDBHooks(hooks)
			return nil, err
		}

		go func(h dbhook.DBHook) {
			for effect := range h.SideEffects() {
				recorder.SideEffectChan() <- effect
			}
		}(hook)

		hooks = append(hooks, hook)
	}

	return hooks, nil
}

func stopDBHooks(hooks []dbhook.DBHook) {
	for _, hook := range hooks {
		_ = hook.Stop()
	}
}
