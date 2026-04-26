package infra

import (
	"strconv"

	"github.com/nats-io/nats.go"
)

type NatsMQ struct {
	nc *nats.Conn
}

func NewNatsMQ(host string, port int) *NatsMQ {
	portStr := strconv.FormatInt(int64(port), 10)
	nc, err := nats.Connect(host + ":" + portStr)
	if err != nil {
		panic(err)
	}
	nc.Subscribe("test", func(msg *nats.Msg) {})
	return &NatsMQ{
		nc: nc,
	}
}

func (n *NatsMQ) GetConn() *nats.Conn {
	return n.nc
}

func (n *NatsMQ) Close() {
	n.nc.Close()
}
