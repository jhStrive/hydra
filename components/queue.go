package components

import (
	"github.com/micro-plat/hydra/conf"
	"github.com/micro-plat/lib4go/queue"
	"github.com/micro-plat/lib4go/types"
)

//queueTypeNode queue在var配置中的类型名称
const queueTypeNode = "queue"

//queueNameNode queue名称在var配置中的末节点名称
const queueNameNode = "queue"

//IQueue 消息队列
type IQueue = queue.IQueue

//IComponentQueue Component Queue
type IComponentQueue interface {
	GetRegularQueue(names ...string) (c IQueue)
	GetQueue(names ...string) (q IQueue, err error)
}

//StandardQueue queue
type StandardQueue struct {
	c IComponents
}

//NewStandardQueue 创建queue
func NewStandardQueue(c IComponents) *StandardQueue {
	return &StandardQueue{c: c}
}

//GetRegularQueue 获取正式的没有异常Queue实例
func (s *StandardQueue) GetRegularQueue(names ...string) (c IQueue) {
	c, err := s.GetQueue(names...)
	if err != nil {
		panic(err)
	}
	return c
}

//GetQueue GetQueue
func (s *StandardQueue) GetQueue(names ...string) (q IQueue, err error) {
	name := types.GetStringByIndex(names, 0, dbNameNode)
	obj, err := s.c.GetOrCreate(dbTypeNode, name, func(c conf.IConf) (interface{}, error) {
		return queue.NewQueue(c.GetString("proto"), string(c.GetRaw()))
	})
	if err != nil {
		return nil, err
	}
	return obj.(IQueue), nil
}