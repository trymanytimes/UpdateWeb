package handler

import (
	"context"
	"fmt"
	"time"

	ftpconn "github.com/jlaffaye/ftp"
	kg "github.com/segmentio/kafka-go"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
	"google.golang.org/protobuf/proto"

	"github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	agentkafkaproducer "github.com/linkingthing/ddi-agent/pkg/kafkaproducer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/trymanytimes/UpdateWeb/config"
	"github.com/trymanytimes/UpdateWeb/pkg/auth/authentification"
	"github.com/trymanytimes/UpdateWeb/pkg/db"
	"github.com/trymanytimes/UpdateWeb/pkg/kafkaproducer"
	"github.com/trymanytimes/UpdateWeb/pkg/log/resource"
	metricresource "github.com/trymanytimes/UpdateWeb/pkg/metric/resource"
	monitorconfig "github.com/linkingthing/ddi-monitor/config"
)

var (
	TableUploadLog   = restdb.ResourceDBType(&resource.UploadLog{})
	uploadLogHandler *UploadLogHandler
	AuditlogIgnore   = "auditlogIgnore"
)

type UploadLogHandler struct{}

func NewUploadLogHandler() *UploadLogHandler {
	uploadLogHandler = &UploadLogHandler{}

	go uploadLogHandler.runKafkaConsumer()
	return uploadLogHandler
}

func (handler *UploadLogHandler) runKafkaConsumer() {
	kafkaReader := kg.NewReader(kg.ReaderConfig{
		Brokers:        config.GetConfig().Kafka.Addr,
		Topic:          agentkafkaproducer.UploadLogTopic,
		GroupID:        config.GetConfig().Kafka.GroupIdUploadLog,
		MinBytes:       1,
		MaxBytes:       1e6,
		MaxWait:        time.Millisecond * 100,
		SessionTimeout: time.Second * 10,
		Dialer: &kg.Dialer{
			Timeout:   time.Second * 10,
			DualStack: true,
			KeepAlive: time.Second * 5},
	})

	defer kafkaReader.Close()
	for {
		message, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read dns message from agent kafka failed: %s", err.Error())
			continue
		}

		switch string(message.Key) {
		case agentkafkaproducer.UploadLogEvent:
			if err = handler.uploadLogEvent(message.Value); err != nil {
				log.Errorf("uploadLogEvent failed:%s", message.Key, err.Error())
			}
		}
	}
}

func (handler *UploadLogHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var uploadLogs []*resource.UploadLog
	user, ok := ctx.Get(authentification.AuthUser)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("user not found"))
	}

	if err := db.GetResources(map[string]interface{}{restdb.IDField: user}, &uploadLogs); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("List uploadLogs failed: %s", err.Error()))
	}

	for _, uploadLog := range uploadLogs {
		uploadLog.Password = ""
	}

	return uploadLogs, nil
}

func (handler *UploadLogHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionUploadLog:
		return handler.uploadLog(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (handler *UploadLogHandler) uploadLog(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ctx.Set(AuditlogIgnore, nil)
	input, ok := ctx.Resource.GetAction().Input.(*resource.UploadLogInput)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("input invalid"))
	}

	user, ok := ctx.Get(authentification.AuthUser)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("user unknown"))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var uploadLog *resource.UploadLog
		uploadLogs, err := tx.Get(TableUploadLog, nil)
		if err != nil {
			return err
		}
		for _, value := range uploadLogs.([]*resource.UploadLog) {
			if value.Address == input.Address &&
				(value.Status == resource.FtpStatusTransporting ||
					value.Status == resource.FtpStatusConnecting) {
				return fmt.Errorf("the address is in transporting,please try after it has completed")
			}

			if value.ID == user {
				uploadLog = value
				if value.Status == resource.FtpStatusTransporting ||
					value.Status == resource.FtpStatusConnecting {
					return fmt.Errorf("the file is transporting,please try after it has completed")
				}
			}
		}

		if err := handler.checkFtpValid(input); err != nil {
			return err
		}

		if uploadLog != nil {
			if _, err := tx.Update(TableUploadLog,
				map[string]interface{}{
					"status":      resource.FtpStatusConnecting,
					"user_name":   input.UserName,
					"password":    input.Password,
					"address":     input.Address,
					"comment":     "",
					"finish_time": ""},
				map[string]interface{}{restdb.IDField: user}); err != nil {
				return err
			}
		} else {
			uploadLog = &resource.UploadLog{
				UserName: input.UserName,
				Password: input.Password,
				Address:  input.Address,
			}
			uploadLog.SetID(user.(string))
			uploadLog.Status = resource.FtpStatusConnecting
			if _, err = tx.Insert(uploadLog); err != nil {
				return err
			}
		}

		dnsMasterIp, err := handler.getDnsMasterNodeIP(tx)
		if err != nil {
			return err
		}

		data, err := proto.Marshal(&pb.UploadLogReq{
			Id:           uploadLog.ID,
			Address:      uploadLog.Address,
			User:         uploadLog.UserName,
			Password:     uploadLog.Password,
			MasterNodeIp: dnsMasterIp})
		if err != nil {
			return fmt.Errorf("proto mashal failed: %s", err.Error())
		}

		return kafkaproducer.GetKafkaProducer().SendDNSCmd(data, kafkaconsumer.UploadLog)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action uploadLog failed:%s", err.Error()))
	}

	return resource.UploadLogOutput{Status: resource.FtpStatusConnecting}, nil
}

func (handler *UploadLogHandler) checkFtpValid(input *resource.UploadLogInput) error {
	c, err := ftpconn.Dial(input.Address, ftpconn.DialWithTimeout(5*time.Second))
	if err != nil {
		return err
	}

	if err = c.Login(input.UserName, input.Password); err != nil {
		return err
	}

	return c.Quit()
}

func (handler *UploadLogHandler) updateStatus(response pb.UploadLogResponse, status resource.UploadLogStatus) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(TableUploadLog,
			map[string]interface{}{
				"status":      status,
				"comment":     response.Message,
				"file_name":   response.FileName,
				"finish_time": response.FinishTime},
			map[string]interface{}{restdb.IDField: response.Id})
		return err
	})
}

func (handler *UploadLogHandler) uploadLogEvent(message []byte) error {
	var response pb.UploadLogResponse
	if err := proto.Unmarshal(message, &response); err != nil {
		return fmt.Errorf("uploadLogEvent Unmarshal message error:%s", err.Error())
	}

	status := resource.FtpStatusConnecting
	switch response.Status {
	case pb.UploadLogResponse_STATUS_CONN_FAILED:
		status = resource.FtpStatusConnFailed
	case pb.UploadLogResponse_STATUS_TRANSPORTING:
		status = resource.FtpStatusTransporting
	case pb.UploadLogResponse_STATUS_TRANSPORT_FAILED:
		status = resource.FtpStatusTransportFailed
	case pb.UploadLogResponse_STATUS_TRANSPORT_DONE:
		status = resource.FtpStatusTransportCompleted
	default:
		return fmt.Errorf("uploadLogEvent Status %s unknown", response.Status)
	}

	return handler.updateStatus(response, status)
}

func (handler *UploadLogHandler) getDnsMasterNodeIP(tx restdb.Transaction) (string, error) {
	var nodes []*metricresource.Node
	if err := tx.Fill(map[string]interface{}{"master": ""}, &nodes); err != nil {
		return "", fmt.Errorf("get nodes failed:%s", err.Error())
	}

	for _, node := range nodes {
		if slice.SliceIndex(node.Roles, string(monitorconfig.ServiceRoleDNS)) != -1 {
			return node.Ip, nil
		}
	}

	return "", fmt.Errorf("dns node not found")
}
