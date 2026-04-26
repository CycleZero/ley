package infra

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var globalEtcdClient *clientv3.Client

func GetEtcdClient() *clientv3.Client {
	if globalEtcdClient == nil {
		panic("etcd 客户端未初始化")
	}
	return globalEtcdClient
}

func NewEtcdClient(endpoints []string) *clientv3.Client {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	globalEtcdClient = client
	return client
}
