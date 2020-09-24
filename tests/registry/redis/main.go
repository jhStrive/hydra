package main

import (
	"fmt"

	"github.com/micro-plat/hydra"
	"github.com/micro-plat/hydra/conf/server/api"
	"github.com/micro-plat/hydra/conf/server/apm"
	"github.com/micro-plat/hydra/conf/server/cron"
	"github.com/micro-plat/hydra/conf/server/mqc"
	squeue "github.com/micro-plat/hydra/conf/server/queue"
	"github.com/micro-plat/hydra/conf/server/task"
	"github.com/micro-plat/hydra/conf/vars/cache"
	"github.com/micro-plat/hydra/conf/vars/db"
	"github.com/micro-plat/hydra/conf/vars/queue"
	"github.com/micro-plat/hydra/context"
	scron "github.com/micro-plat/hydra/hydra/servers/cron"
	"github.com/micro-plat/hydra/hydra/servers/http"
	smqc "github.com/micro-plat/hydra/hydra/servers/mqc"
	"github.com/micro-plat/hydra/hydra/servers/rpc"

	_ "github.com/micro-plat/hydra/components/pkgs/mq/redis"
)

var app = hydra.NewApp(
	hydra.WithServerTypes(http.API, rpc.RPC, smqc.MQC, scron.CRON),
	hydra.WithPlatName("taosytest"),
	hydra.WithSystemName("test-rgtredis"),
	hydra.WithClusterName("taosy"),
	hydra.WithRegistry("redis://192.168.5.79:6379"),
	// hydra.WithRegistry("redis://192.168.0.111:6379,192.168.0.112:6379,192.168.0.113:6379,192.168.0.114:6379,192.168.0.115:6379,192.168.0.116:6379"),
	// hydra.WithRegistry("zk://192.168.0.101:2181"),
)

func init() {
	hydra.Conf.Vars().DB("taosy_db", db.New("oracle", "connstring", db.WithConnect(10, 10, 10)))
	hydra.Conf.Vars().Cache("cache", cache.New("redis", []byte(`{"proto":"redis","addrs":["192.168.5.79:6379"],"db":0,"dial_timeout":10,"read_timeout":10,"write_time":10,"pool_size":10}`)))
	hydra.Conf.Vars().Queue("queue", queue.New("redis", []byte(`{"proto":"redis","addrs":["192.168.5.79:6379"],"db":0,"dial_timeout":10,"read_timeout":10,"write_time":10,"pool_size":10}`)))
	hydra.Conf.RPC(":8071")
	queues := &squeue.Queues{}
	queues = queues.Append(squeue.NewQueue("queuename1", "/testmqc"))
	mqser := hydra.Conf.MQC("redis://queue", mqc.WithTrace(), mqc.WithTimeout(10))
	// mqser.Sub("server", `{"proto":"redis","addrs":["192.168.5.79:6379"],"db":0,"dial_timeout":10,"read_timeout":10,"write_time":10,"pool_size":10}`)
	mqser.Queue(queues.Queues...)
	tasks := task.Tasks{}
	tasks.Append(task.NewTask(fmt.Sprintf("@every %ds", 10), "/testcron"))
	hydra.Conf.CRON(cron.WithEnable(), cron.WithTrace(), cron.WithTimeout(10), cron.WithSharding(1)).Task(tasks.Tasks...)
	hydra.Conf.API(":8070", api.WithTimeout(10, 10), api.WithEnable()).APM("skywalking", apm.WithDisable())
	app.API("/taosy/testapi", func(ctx context.IContext) (r interface{}) {
		ctx.Log().Info("api 接口服务测试")
		return nil
	})

	app.RPC("/taosy/testrpc", func(ctx context.IContext) (r interface{}) {
		ctx.Log().Info("rpc 接口服务测试")
		return nil
	})

	app.MQC("/testmqc", func(ctx context.IContext) (r interface{}) {
		ctx.Log().Info("mqc 接口服务测试")
		return nil
	})

	app.CRON("/testcron", func(ctx context.IContext) (r interface{}) {
		ctx.Log().Info("cron 接口服务测试")
		return nil
	})
}

func main() {
	app.Start()
}
