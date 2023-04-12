package main

import (
	"example/mydocker/cgroups/subsystems"
	"example/mydocker/container"
	"example/mydocker/network"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: "Create a container with namespace and cgroups limit mydocker run -ti [command]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "volume",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "specify container name",
		},
		cli.StringFlag{
			Name:  "mem", // 如果Name只有一个字母的话，只需要一个 - 就行，多个字母就需要两个--
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.StringSliceFlag{
			Name: "e",
			Usage: "set environment",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		// context.Args()代表flag处后面的内容
		var cmd []string
		for _, arg := range context.Args() {
			cmd = append(cmd, arg)
		}
		tty := context.Bool("ti")
		detach := context.Bool("d")
		if tty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("mem"),
			CpuSet:      context.String("cpuset"),
			CpuShare:    context.String("cpushare"),
		}
		volume := context.String("v")
		containerName := context.String("name")
		imageName := cmd[0]
		cmd = cmd[1:]
		environment := context.StringSlice("e")
		Run(tty, cmd, resConf, volume, containerName,imageName,environment)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
		log.Info("init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}

var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name or image name")
		}
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		// 此处暂时大小写无所谓，为了统一，都改成大写
		CommitContainer(containerName,imageName)
		return nil
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the container",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

var logCommand = cli.Command{
	Name:  "log",
	Usage: "print container log",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("please input your container name")
		}
		containerName := context.Args().Get(0)
		// 此处暂时大小写无所谓，为了统一，都改成大写
		LogContainer(containerName)
		return nil
	},
}

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		if os.Getenv(ENV_EXEC_PID) != ""{
			log.Infof("pid callback pid %s",os.Getpid())
			return nil;
		}
		// 至少要指定两个参数
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}
		containerName := context.Args().Get(0)
		var commandArray []string
		// 这种方式更简洁
		commandArray = append(commandArray,context.Args().Tail()...)
		// for _,arg := range context.Args().Tail(){
		// 	commandArray=append(commandArray, arg)
		// }
		ExecContainer(containerName,commandArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("please input your container name")
		}
		containerName := context.Args().Get(0)
		StopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("please input your container name")
		}
		containerName := context.Args().Get(0)
		RemoveContainer(containerName)
		return nil
	},
}

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name: "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{ Name: "driver",	Usage: "network driver",},
				cli.StringFlag{ Name: "subnet",Usage: "subnet cidr"},
			},
			Action: func (context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}
				network.Init()
				err := network.CreateNetwork(context.String("driver"),context.String("subnet"),context.Args()[0])
				if err != nil {
					return fmt.Errorf("create network error: %+v",err)
				}
				return nil
			},
		},
		{
			Name: "list",
			Usage: "list container network",
			Action: func (context *cli.Context) error {
				network.Init()
				network.ListNetwork()
				return nil
			},
		},
		{
			Name: "remove",
			Usage: "remove container network",
			Action: func (context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}
				network.Init()
				err := network.DeleteNetwork(context.Args()[0])
				if err != nil {
					return fmt.Errorf("remove network error: %+v", err)
				}
				return nil
			},
		},
	},
}