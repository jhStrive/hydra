package global

import (
	"fmt"
	"net"
	"strings"

	"github.com/asaskevich/govalidator"
)

const (
	ProtoZK      = "zk"
	ProtoRPC     = "rpc"
	ProtoHTTP    = "http"
	ProtoLM      = "lm"
	ProtoFS      = "fs"
	ProtoLMQ     = "lmq"
	ProtoREDIS   = "redis"
	ProtoMQTT    = "mqtt"
	ProtoInvoker = "ivk"
)

//ParseProto 解析协议信息
func ParseProto(address string) (string, string, error) {
	address = strings.Trim(address, " ")

	addr := strings.Split(address, "://")
	if len(addr) != 2 {
		return "", "", fmt.Errorf("%s协议格式错误,正确格式(proto://addr)", addr)
	}
	proto := addr[0]
	if proto == "" {
		return "", "", fmt.Errorf("%s缺少协议proto,正确格式(proto://addr)", address)
	}
	raddr := addr[1]
	if raddr == "" {
		return "", "", fmt.Errorf("%s缺少地址addr,正确格式(proto://addr)", address)
	}
	if !strings.HasPrefix(raddr, "/") {
		if isIP(raddr) {
			return proto, raddr, nil
		}
		raddr = fmt.Sprintf("/%s", raddr)
	}
	return proto, raddr, nil
}
func isIP(addr string) bool {
	if !strings.Contains(addr, ":") {
		if govalidator.IsIP(addr) {
			return true
		}
		return false
	}
	a, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	return govalidator.IsIP(a)
}

//IsProto 是否是指定的协议
func IsProto(addr string, proto ...string) (string, bool) {
	for _, prt := range proto {
		p, addrs, _ := ParseProto(addr)
		if p == prt {
			return addrs, true
		}
	}
	return "", false
}

//IsLocal 是否是本地服务
func IsLocal(proto string) bool {
	return strings.EqualFold(proto, ProtoLM) || strings.EqualFold(proto, ProtoLMQ)
}
