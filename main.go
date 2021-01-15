package main

import (
	"net/url"
	"os"

	cli "github.com/jawher/mow.cli"
	"go.uber.org/zap"
)

var log *zap.Logger

func main() {
	log, _ = zap.NewDevelopment()
	defer func() { _ = log.Sync() }()

	app := cli.App("proxy-protocol-tester", "tests the proxy protocol")
	app.Command("client", "runs a proxy protocol client", cmdClient)
	app.Command("server", "runs a proxy protocol server", cmdServer)
	_ = app.Run(os.Args)
}

func cmdClient(cmd *cli.Cmd) {
	cmd.Spec = "URL [ --proxy-protocol-version ]"
	dst := cmd.StringArg("URL", "", "the URL to reach")
	version := cmd.IntOpt("proxy-protocol-version", 0, "the HAProxy protocol version to use")
	cmd.Action = func() {
		dstURL, err := url.Parse(*dst)
		if err != nil {
			log.Fatal("invalid destination URL", zap.Error(err))
		}
		runClient(byte(*version), dstURL)
	}
}

func cmdServer(cmd *cli.Cmd) {
	cmd.Spec = "ADDRESS"
	addr := cmd.StringArg("ADDRESS", "", "the host:port address to listen on")
	cmd.Action = func() {
		runServer(*addr)
	}
}
