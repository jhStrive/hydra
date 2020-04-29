package conf

import (
	"fmt"

	"github.com/micro-plat/hydra/registry"
)

//IMainConf 主配置信息
type IMainConf interface {
	IPub
	IsStarted() bool
	IsTrace() bool
	GetClusterNodes() CNodes
	GetMainConf() *JSONConf
	GetMainObject(v interface{}) (int32, error)
	GetSubConf(name string) (*JSONConf, error)
	GetSubObject(name string, v interface{}) (int32, error)
	GetRegistry() registry.IRegistry
	GetVersion() int32
	Has(names ...string) bool
	Iter(f func(path string, conf *JSONConf) bool)
}

//MainConf 服务器主配置
type MainConf struct {
	mainConf *JSONConf
	version  int32
	subConfs map[string]JSONConf
	registry registry.IRegistry
	IPub
}

//NewMainConf 管理服务器的主配置信息
func NewMainConf(platName string, systemName string, serverType string, clusterName string, rgst registry.IRegistry) (s *MainConf, err error) {
	s = &MainConf{
		registry: rgst,
		IPub:     NewPub(platName, systemName, serverType, clusterName),
		subConfs: make(map[string]JSONConf),
	}
	if err = s.load(); err != nil {
		return
	}
	return s, nil
}

//load 加载配置
func (c *MainConf) load() (err error) {

	var mainConfRaw []byte
	mainConfRaw, c.version, err = c.registry.GetValue(c.GetMainPath())
	if err != nil {
		return err
	}
	rdata, err := decrypt(mainConfRaw)
	if err != nil {
		return err
	}
	//初始化主配置
	if c.mainConf, err = NewJSONConf(rdata, c.version); err != nil {
		err = fmt.Errorf("%s配置有误:%v", c.GetMainPath(), err)
		return err
	}
	confs, _, err := c.registry.GetChildren(c.GetMainPath())
	if err != nil {
		return err
	}
	for _, p := range confs {
		childConfPath := registry.Join(c.GetMainPath(), p)
		data, version, err := c.registry.GetValue(childConfPath)
		if err != nil {
			return err
		}
		rdata, err := decrypt(data)
		if err != nil {
			return err
		}
		childConf, err := NewJSONConf(rdata, version)
		if err != nil {
			err = fmt.Errorf("%s配置有误:%v", childConfPath, err)
			return err
		}
		c.subConfs[p] = *childConf
	}
	return nil
}

//IsTrace 是否跟踪请求或响应
func (c *MainConf) IsTrace() bool {
	return c.mainConf.GetString("trace", "true") == "true"
}

//GetRegistry 获取注册中心
func (c *MainConf) GetRegistry() registry.IRegistry {
	return c.registry
}

//IsStarted 当前服务是否已启动
func (c *MainConf) IsStarted() bool {
	return c.mainConf.GetString("status", "start") == "start"
}

//GetVersion 获取版本号
func (c *MainConf) GetVersion() int32 {
	return c.version
}

//GetClusterNodes 获取集群中的所有节点
func (c *MainConf) GetClusterNodes() CNodes {
	cnodes := make([]*CNode, 0, 2)
	path := c.GetServerPubPath()
	children, _, err := c.registry.GetChildren(path)
	if err != nil {
		return nil
	}

	for i, name := range children {
		cnodes = append(cnodes, NewCNode(name, c.GetClusterID(), i))
	}
	return cnodes
}

//GetMainConf 获取当前主配置
func (c *MainConf) GetMainConf() *JSONConf {
	return c.mainConf
}

//GetMainObject 获取主配置信息
func (c *MainConf) GetMainObject(v interface{}) (int32, error) {
	conf := c.GetMainConf()
	if err := conf.Unmarshal(&v); err != nil {
		err = fmt.Errorf("获取主配置失败:%v", err)
		return 0, err
	}

	return conf.version, nil
}

//GetSubConf 指定子配置
func (c *MainConf) GetSubConf(name string) (*JSONConf, error) {
	if v, ok := c.subConfs[name]; ok {
		return &v, nil
	}
	return nil, ErrNoSetting
}

//GetSubObject 获取子配置信息
func (c *MainConf) GetSubObject(name string, v interface{}) (int32, error) {
	conf, err := c.GetSubConf(name)
	if err != nil {
		return 0, err
	}

	if err := conf.Unmarshal(&v); err != nil {
		err = fmt.Errorf("获取%s配置失败:%v", name, err)
		return 0, err
	}
	return conf.version, nil
}

//Has 是否存在子配置
func (c *MainConf) Has(names ...string) bool {
	for _, name := range names {
		_, ok := c.subConfs[name]
		if ok {
			return true
		}
	}
	return false
}

//Iter 迭代所有配置
func (c *MainConf) Iter(f func(path string, conf *JSONConf) bool) {
	for path, v := range c.subConfs {
		if !f(path, &v) {
			break
		}
	}
}
