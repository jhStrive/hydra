package run

import (
	"os"

	"github.com/lib4dev/cli/cmds"
	"github.com/micro-plat/hydra/hydra/cmds/pkgs"
	"github.com/urfave/cli"
)

func init() {
	cmds.RegisterFunc(func() cli.Command {
		return cli.Command{
			Name:   "run",
			Usage:  "运行服务,以前台方式运行服务。通过终端输出日志，终端关闭后服务自动退出。",
			Flags:  getFlags(),
			Action: doRun,
		}
	})
}

//doRun 服务启动
func doRun(c *cli.Context) (err error) {
	//1.创建本地服务
	hydraSrv, err := pkgs.GetService(c, os.Args[2:]...)
	if err != nil {
		return err
	}
	err = hydraSrv.Run()
	return nil
}
