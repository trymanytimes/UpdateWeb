package main

import (
	"flag"
	"strings"

	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/auth"
	"github.com/trymanytimes/UpdateWeb/pkg/business"
	"github.com/trymanytimes/UpdateWeb/pkg/db"
	"github.com/trymanytimes/UpdateWeb/pkg/grpcclient"
	auditlog "github.com/trymanytimes/UpdateWeb/pkg/log"
	"github.com/trymanytimes/UpdateWeb/pkg/metric"
	restserver "github.com/trymanytimes/UpdateWeb/server"
)

var (
	configFile string
)

func main() {
	flag.StringVar(&configFile, "c", "web-controller.conf", "configure file path")
	flag.Parse()

	log.InitLogger(log.Debug)
	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s", err.Error())
	}
	db.RegisterResources(auditlog.PersistentResources()...)
	db.RegisterResources(auth.PersistentResources()...)
	if err := db.Init(conf); err != nil {
		log.Fatalf("init db failed: %s", err.Error())
	}
	conn, err := grpc.Dial(conf.APIServer.GrpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dail grpc failed: %s", err.Error())
	}
	defer conn.Close()
	if len(strings.Split(conf.Server.GrpcAddr, ":")) < 2 {
		log.Fatalf("grpc address is not correct")
	}
	grpcclient.NewGrpcClient(conn)

	server, err := restserver.NewServer()
	if err != nil {
		log.Fatalf("new server failed: %s", err.Error())
	}

	server.RegisterHandler(restserver.HandlerRegister(metric.RegisterHandler))
	if err := server.RegisterHandler(restserver.HandlerRegister(auth.RegisterHandler)); err != nil {
		log.Fatalf("register auth failed: %s", err.Error())
	}
	server.RegisterHandler(restserver.HandlerRegister(auditlog.RegisterHandler))
	server.RegisterHandler(restserver.HandlerRegister(business.RegisterHandler))

	if err := server.Run(conf); err != nil {
		log.Fatalf("server run failed: %s", err.Error())
	}
}
