package nacos

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/De3trvctions/Stallink/internal/conf"
	source "github.com/go-kratos/kratos/contrib/config/nacos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"gopkg.in/yaml.v3"
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
	watcher   kconfig.Watcher
}

func Load(client config_client.IConfigClient, remote conf.RemoteConfig) (*Remote, error) {
	src := source.NewConfigSource(client, source.WithGroup(remote.Group), source.WithDataID(remote.DataID))
	w, err := src.Watch()
	if err != nil {
		return nil, err
	}
	values, err := src.Load()
	if err == nil {
		var bc conf.Bootstrap
		bc, err = decode(values)
		if err == nil {
			return &Remote{Bootstrap: bc, watcher: w}, nil
		}
	}
	_ = w.Stop()
	return nil, err
}

// Decode every publication from scratch so deleting a field restores its default.
func decode(values []*kconfig.KeyValue) (conf.Bootstrap, error) {
	var bc conf.Bootstrap
	var doc map[string]any
	if len(values) != 1 || yaml.Unmarshal(values[0].Value, &doc) != nil {
		return bc, errors.New("invalid Nacos YAML")
	}
	raw, err := json.Marshal(doc)
	if err != nil || json.Unmarshal(raw, &bc) != nil {
		return bc, errors.New("invalid Nacos configuration fields")
	}
	bc.ResolveEnvironment()
	return bc, conf.ValidateBootstrap(bc)
}

func (r *Remote) Next() (conf.Bootstrap, error) {
	values, err := r.watcher.Next()
	if err != nil {
		return conf.Bootstrap{}, err
	}
	return decode(values)
}

func (r *Remote) Close() error { return r.watcher.Stop() }

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
