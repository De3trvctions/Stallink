package nacos

import (
	"fmt"

	"github.com/De3trvctions/Stallink/internal/conf"
	"github.com/De3trvctions/Stallink/internal/logging"
	source "github.com/go-kratos/kratos/contrib/config/nacos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"go.uber.org/zap"
)

func NewClient(n conf.Nacos) (config_client.IConfigClient, error) {
	clientConfig := constant.ClientConfig{
		NamespaceId:         n.Namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              valueOr(n.LogDir, "./logs/nacos/stallink/config"),
		CacheDir:            valueOr(n.CacheDir, "./logs/nacos/stallink/config-cache"),
		LogLevel:            valueOr(n.LogLevel, "error"),
	}
	if n.Username != "" && n.Password != "" {
		clientConfig.Username = n.Username
		clientConfig.Password = n.Password
	}
	client, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: []constant.ServerConfig{*constant.NewServerConfig(n.Host, n.Port)},
	})
	if err != nil {
		return nil, fmt.Errorf("nacos client: %w", err)
	}
	return client, nil
}

type Remote struct {
	Bootstrap conf.Bootstrap
	source    kconfig.Config
}

func Load(client config_client.IConfigClient, remote conf.RemoteConfig) (*Remote, error) {
	c := kconfig.New(kconfig.WithSource(
		source.NewConfigSource(client, source.WithGroup(remote.Group), source.WithDataID(remote.DataID)),
	))
	if err := c.Load(); err != nil {
		closeSource(c)
		return nil, fmt.Errorf("load nacos %s/%s: %w", remote.Group, remote.DataID, err)
	}
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		closeSource(c)
		return nil, fmt.Errorf("scan nacos %s/%s: %w", remote.Group, remote.DataID, err)
	}
	bc.ResolveEnvironment()
	if err := conf.ValidateBootstrap(bc); err != nil {
		closeSource(c)
		return nil, err
	}
	return &Remote{Bootstrap: bc, source: c}, nil
}

func (r *Remote) Close() error {
	if r == nil || r.source == nil {
		return nil
	}
	return r.source.Close()
}

func closeSource(c kconfig.Config) {
	if err := c.Close(); err != nil {
		logging.Error("close nacos config source", zap.Error(err))
	}
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
