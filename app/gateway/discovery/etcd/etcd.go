package etcd

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/CycleZero/ley/app/gateway/discovery"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func init() {
	discovery.Register("etcd", New)
}

// New creates a new etcd registry from DSN
// DSN format: etcd://host1,host2,host3/prefix?dial_timeout=5s&username=user&password=pass
func New(dsn *url.URL) (registry.Discovery, error) {
	// Parse endpoints from host (supports multiple hosts separated by comma)
	endpoints := strings.Split(dsn.Host, ",")
	if len(endpoints) == 0 || (len(endpoints) == 1 && endpoints[0] == "") {
		endpoints = []string{"127.0.0.1:2379"}
	}

	fmt.Println("etcd endpoints:", endpoints)
	// Build etcd client config
	config := clientv3.Config{
		Endpoints: endpoints,
	}

	// Parse query parameters
	query := dsn.Query()

	config.DialTimeout = time.Second * 10

	// Username and password for authentication
	if username := query.Get("username"); username != "" {
		config.Username = username
	}
	if password := query.Get("password"); password != "" {
		config.Password = password
	}

	// Create etcd client
	client, err := clientv3.New(config)
	if err != nil {
		return nil, err
	}

	// Create etcd registry
	return etcd.New(client), nil
}
