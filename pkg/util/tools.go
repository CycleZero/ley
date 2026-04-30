package util

import (
	"context"
	"os"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-kratos/kratos/v2/registry"
	grpcx "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// GetContextWithTimeOut 获取一个超时的context,简化写法
func GetContextWithTimeOut(sec int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(sec)*time.Second)
}

// UniqueSlice 数组去重
func UniqueSlice[T comparable](slice []T) []T {
	s := mapset.NewSet[T]()
	s.Append(slice...)
	slice = s.ToSlice()
	return slice
}

// InArray 判断任意可使用==运算符的元素是否在列表中
func InArray[T comparable](val T, array []T) bool {
	for _, v := range array {
		if v == val {
			return true
		}
	}
	return false
}

func DiscoveryEndpoint(serviceName string) string {
	schema := "discovery"
	auth := ""
	endpoint := schema + "://" + auth + "/" + serviceName
	return endpoint
}

func NewGrpcConn(dis registry.Discovery, serviceName string) (*grpc.ClientConn, error) {
	return grpcx.DialInsecure(
		context.Background(),
		grpcx.WithDiscovery(dis),
		grpcx.WithEndpoint(DiscoveryEndpoint(serviceName)),
	)
}

func ServiceId(serviceName string) string {
	i := uuid.New().String()
	host, err := os.Hostname()
	if err != nil {
		return serviceName + "." + i
	}
	return host + "." + serviceName + "." + i
}
