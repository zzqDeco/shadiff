package cmd

import (
	"context"
	"fmt"
	"strings"

	"shadiff/internal/capture/dbhook"
	"shadiff/internal/config"
	"shadiff/internal/model"
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

func resolveReplayDBProxies(flagChanged bool, flagValues []string, cfg *config.AppConfig) ([]config.DBProxyConfig, error) {
	if !flagChanged {
		return append([]config.DBProxyConfig(nil), cfg.Replay.DBProxies...), nil
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

func startDBHooks(ctx context.Context, sink chan<- model.SideEffect, proxies []config.DBProxyConfig) (*dbhook.Group, error) {
	hooks := make([]dbhook.DBHook, 0, len(proxies))
	for _, proxy := range proxies {
		hook, err := newDBHook(dbhook.Config{
			DBType:     proxy.Type,
			ListenAddr: proxy.ListenAddr,
			TargetAddr: proxy.TargetAddr,
		})
		if err != nil {
			stopHookSlice(hooks)
			return nil, err
		}
		if err := hook.Start(ctx); err != nil {
			stopHookSlice(hooks)
			return nil, err
		}

		hooks = append(hooks, hook)
	}

	return dbhook.NewGroup(ctx, hooks, sink), nil
}

func stopDBHooks(hooks interface{ Stop() error }) {
	if hooks == nil {
		return
	}
	_ = hooks.Stop()
}

func stopHookSlice(hooks []dbhook.DBHook) {
	for _, hook := range hooks {
		_ = hook.Stop()
	}
}
