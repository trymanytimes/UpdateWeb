package handler

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/ddi-controller/pkg/kafkaproducer"
)

func SendKafkaMessage(id, kafkaCmd string, m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return fmt.Errorf("%s %s's proto mashal failed: %s\n", kafkaCmd, id, err.Error())
	}

	if err := kafkaproducer.GetKafkaProducer().SendDNSCmd(data, kafkaCmd); err != nil {
		return fmt.Errorf("%s %s's send command to kafka failed: %s", kafkaCmd, id, err.Error())
	}

	return nil
}
