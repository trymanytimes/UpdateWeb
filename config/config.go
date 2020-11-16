package config

import (
	"github.com/zdnscloud/cement/configure"
)

type DDIControllerConfig struct {
	Path          string            `yaml:"-"`
	DB            DBConf            `yaml:"db"`
	Server        ServerConf        `yaml:"server"`
	Prometheus    PrometheusConf    `yaml:"prometheus"`
	Elasticsearch ElasticsearchConf `yaml:"elasticsearch"`
	MonitorNode   MonitorNodeConf   `yaml:"monitor_node"`
	AuditLog      AuditLogConf      `yaml:"audit_log"`
	APIServer     APIGrpcConf       `yaml:"api_server"`
	VIP           VIPConf           `yaml:"vip"`
}

type DBConf struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
	Host     string `json:"host"`
}

type ServerConf struct {
	IP          string `yaml:"ip"`
	Port        string `yaml:"port"`
	Hostname    string `yaml:"hostname"`
	TlsCertFile string `yaml:"tls_cert_file"`
	TlsKeyFile  string `yaml:"tls_key_file"`
	Master      string `yaml:"master"`
	GrpcAddr    string `yaml:"grpc_addr"`
	RoleConf    string `yaml:"role_conf"`
}

type APIGrpcConf struct {
	GrpcAddr string `yaml:"grpc_addr"`
}

type PrometheusConf struct {
	Addr       string `yaml:"addr"`
	ExportPort int    `yaml:"export_port"`
}

type MonitorNodeConf struct {
	TimeOut int64 `yaml:"time_out"`
}
type ElasticsearchConf struct {
	Addr  string `yaml:"addr"`
	Index string `yaml:"index"`
}

type AuditLogConf struct {
	ValidPeriod uint32 `yaml:"valid_period"`
}

type VIPConf struct {
	BeginVIP string `yaml:"begin_vip"`
	EndVIP   string `yaml:"end_vip"`
	Length   int32  `yaml:"length"`
}

var gConf *DDIControllerConfig

func LoadConfig(path string) (*DDIControllerConfig, error) {
	var conf DDIControllerConfig
	conf.Path = path
	if err := conf.Reload(); err != nil {
		return nil, err
	}

	return &conf, nil
}

func (c *DDIControllerConfig) Reload() error {
	var newConf DDIControllerConfig
	if err := configure.Load(&newConf, c.Path); err != nil {
		return err
	}

	newConf.Path = c.Path
	*c = newConf
	gConf = &newConf
	return nil
}

func GetConfig() *DDIControllerConfig {
	return gConf
}
