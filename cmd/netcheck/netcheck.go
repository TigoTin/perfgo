package main

import (
	"fmt"
	"log"
	"os"

	"perfgo/pkg/utils"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "netcheck",
		Usage:   "网络接口检测工具",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:    "check",
				Aliases: []string{"c"},
				Usage:   "检测网络接口状态",
				Action: func(cCtx *cli.Context) error {
					return checkNetwork()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func checkNetwork() error {
	fmt.Println("正在检测网络接口状态...")

	results, err := utils.CheckAllNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("检测网络接口时出错: %v", err)
	}

	if len(results) == 0 {
		fmt.Println("未找到活动的网络接口")
		return nil
	}

	utils.PrintNetworkCheckResults(results)

	return nil
}
