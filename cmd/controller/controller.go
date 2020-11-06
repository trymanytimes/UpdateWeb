package main

import (
	"flag"
	"strings"

	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/agentevent"
	"github.com/linkingthing/ddi-controller/pkg/alarm"
	"github.com/linkingthing/ddi-controller/pkg/auth"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dhcp"
	"github.com/linkingthing/ddi-controller/pkg/dns"
	"github.com/linkingthing/ddi-controller/pkg/grpcclient"
	"github.com/linkingthing/ddi-controller/pkg/grpcserver"
	"github.com/linkingthing/ddi-controller/pkg/ipam"
	"github.com/linkingthing/ddi-controller/pkg/kafkaproducer"
	auditlog "github.com/linkingthing/ddi-controller/pkg/log"
	"github.com/linkingthing/ddi-controller/pkg/metric"
	restserver "github.com/linkingthing/ddi-controller/server"
)

var (
	configFile string
)

func main() {
	flag.StringVar(&configFile, "c", "controller.conf", "configure file path")
	flag.Parse()

	log.InitLogger(log.Debug)
	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s", err.Error())
	}

	db.RegisterResources(dhcp.PersistentResources()...)
	db.RegisterResources(ipam.PersistentResources()...)
	db.RegisterResources(metric.PersistentResources()...)
	db.RegisterResources(dns.PersistentResources()...)
	db.RegisterResources(alarm.PersistentResources()...)
	db.RegisterResources(auditlog.PersistentResources()...)
	db.RegisterResources(auth.PersistentResources()...)
	db.RegisterResources(agentevent.PersistentResources()...)
	if err := db.Init(conf); err != nil {
		log.Fatalf("init db failed: %s", err.Error())
	}

	s, err := grpcserver.New(conf)
	if err != nil {
		log.Fatalf("grpc server start fail: %s", err.Error())
	}
	go s.Run()
	kafkaproducer.Init(conf)

	conn, err := grpc.Dial(conf.DDIAgent.GrpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dail grpc failed: %s", err.Error())
	}
	defer conn.Close()
	if len(strings.Split(conf.Server.GrpcAddr, ":")) < 2 {
		log.Fatalf("grpc address is not correct")
	}
	grpcclient.NewDhcpClient(conn)

	server, err := restserver.NewServer()
	if err != nil {
		log.Fatalf("new server failed: %s", err.Error())
	}

	server.RegisterHandler(restserver.HandlerRegister(ipam.RegisterHandler))
	server.RegisterHandler(restserver.HandlerRegister(dhcp.RegisterHandler))
	server.RegisterHandler(restserver.HandlerRegister(metric.RegisterHandler))
	if err := server.RegisterHandler(restserver.HandlerRegister(dns.RegisterHandler)); err != nil {
		log.Fatalf("register dns failed: %s", err.Error())
	}
	if err := server.RegisterHandler(restserver.HandlerRegister(alarm.RegisterHandler)); err != nil {
		log.Fatalf("register alarm failed: %s", err.Error())
	}
	if err := server.RegisterHandler(restserver.HandlerRegister(auth.RegisterHandler)); err != nil {
		log.Fatalf("register auth failed: %s", err.Error())
	}
	server.RegisterHandler(restserver.HandlerRegister(auditlog.RegisterHandler))
	if err := server.RegisterHandler(restserver.HandlerRegister(agentevent.RegisterHandler)); err != nil {
		log.Fatalf("register agentevent failed: %s", err.Error())
	}

	if err := server.Run(conf); err != nil {
		log.Fatalf("server run failed: %s", err.Error())
	}
}
