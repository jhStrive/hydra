package server

import (
	"strings"

	"github.com/micro-plat/hydra/conf"
)

//CNode 集群节点
type CNode struct {
	name     string
	root     string
	path     string
	host     string
	index    int
	serverID string
	mid      string
}

//NewCNode 构建集群节点信息
func NewCNode(name string, mid string, index int) *CNode {
	items := strings.Split(name, "_")
	return &CNode{
		name:     name,
		mid:      mid,
		index:    index,
		host:     items[0],
		serverID: items[1],
	}
}

//IsAvailable 当前节点是否是可用的
func (c *CNode) IsAvailable() bool {
	return c.name != ""
}

//GetHost 获取服务器信息
func (c *CNode) GetHost() string {
	return c.host
}

//GetServerID 获取节点编号
func (c *CNode) GetServerID() string {
	return c.serverID
}

//GetName 获取当前服务名称
func (c *CNode) GetName() string {
	return c.name
}

//IsCurrent 是否是当前服务
func (c *CNode) IsCurrent() bool {
	return c.serverID == c.mid
}

//GetIndex 获取当前节点的索引编号
func (c *CNode) GetIndex() int {
	return c.index
}

//IsMaster 判断当前节点是否是主节点
func (c *CNode) IsMaster(i int) bool {
	return c.index < i
}

//Clone 克隆当前对象
func (c *CNode) Clone() conf.ICNode {
	node := *c
	return &node
}
