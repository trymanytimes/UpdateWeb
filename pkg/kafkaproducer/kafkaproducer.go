package kafkaproducer

import (
	"context"
	"time"

	kg "github.com/segmentio/kafka-go"

	dhcpconsumer "github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	dnsconsumer "github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	"github.com/trymanytimes/UpdateWeb/config"
)

type KafkaProducer struct {
	dnsWriter  *kg.Writer
	dhcpWriter *kg.Writer
}

var globalKafkaProducer *KafkaProducer

func GetKafkaProducer() *KafkaProducer {
	return globalKafkaProducer
}

func Init(conf *config.DDIControllerConfig) {
	globalKafkaProducer = &KafkaProducer{
		dnsWriter: kg.NewWriter(kg.WriterConfig{
			Brokers:   conf.Kafka.Addr,
			Topic:     dnsconsumer.DNSTopic,
			BatchSize: 1,
			Dialer: &kg.Dialer{
				Timeout:   time.Second * 10,
				DualStack: true,
				KeepAlive: time.Second * 5},
		}),
		dhcpWriter: kg.NewWriter(kg.WriterConfig{
			Brokers:   conf.Kafka.Addr,
			Topic:     dhcpconsumer.Topic,
			BatchSize: 1,
			Dialer: &kg.Dialer{
				Timeout:   time.Second * 10,
				DualStack: true,
				KeepAlive: time.Second * 5},
		}),
	}
}

func (producer *KafkaProducer) SendDNSCmd(data []byte, cmd string) error {
	return producer.dnsWriter.WriteMessages(context.Background(), kg.Message{Key: []byte(cmd), Value: data})
}

func (producer *KafkaProducer) SendDHCPCmd(cmd string, data []byte) error {
	return producer.dhcpWriter.WriteMessages(context.Background(), kg.Message{Key: []byte(cmd), Value: data})
}
