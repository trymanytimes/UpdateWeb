package handler

import (
	"fmt"
	"sort"

	"github.com/golang/protobuf/proto"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	"github.com/linkingthing/ddi-controller/pkg/kafkaproducer"
)

const (
	ClientClassOption60 = "option[vendor-class-identifier].text == '%s'"
)

var (
	TableClientClass = restdb.ResourceDBType(&resource.ClientClass{})
)

type ClientClassHandler struct {
}

func NewClientClassHandler() *ClientClassHandler {
	return &ClientClassHandler{}
}

func (s *ClientClassHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass)
	if clientclass.Name == "" || clientclass.Regexp == "" {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("clientclass params name %s and regexp %s must not be empty", clientclass.Name, clientclass.Regexp))
	}

	clientclass.SetID(clientclass.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientclass); err != nil {
			return err
		}

		return sendCreateClientClassCmdToDDIAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add clientclass %s failed: %s", clientclass.Name, err.Error()))
	}

	return clientclass, nil
}

func sendCreateClientClassCmdToDDIAgent(clientclass *resource.ClientClass) error {
	req, err := proto.Marshal(&pb.CreateClientClass4Request{
		Header: &pb.DDIRequestHead{Method: "Create", Resource: clientclass.GetType()},
		Name:   clientclass.Name,
		Regexp: fmt.Sprintf(ClientClassOption60, clientclass.Regexp),
	})

	if err != nil {
		return fmt.Errorf("marshal create clientclass request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.CreateClientClass4, req)
}

func (s *ClientClassHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var clientclasses resource.ClientClasses
	if err := db.GetResources(nil, &clientclasses); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list clientclasses from db failed: %s", err.Error()))
	}

	sort.Sort(clientclasses)
	return clientclasses, nil
}

func (s *ClientClassHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclassID := ctx.Resource.(*resource.ClientClass).GetID()
	var clientclasses []*resource.ClientClass
	clientclass, err := restdb.GetResourceWithID(db.GetDB(), clientclassID, &clientclasses)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get clientclass %s from db failed: %s", clientclassID, err.Error()))
	}

	return clientclass.(*resource.ClientClass), nil
}

func (s *ClientClassHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableClientClass, map[string]interface{}{
			"regexp": clientclass.Regexp,
		}, map[string]interface{}{restdb.IDField: clientclass.GetID()}); err != nil {
			return err
		}

		return sendUpdateClientClassCmdToDDIAgent(clientclass)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update clientclass %s failed: %s", clientclass.GetID(), err.Error()))
	}

	return clientclass, nil
}

func sendUpdateClientClassCmdToDDIAgent(clientclass *resource.ClientClass) error {
	req, err := proto.Marshal(&pb.UpdateClientClass4Request{
		Header: &pb.DDIRequestHead{Method: "Update", Resource: clientclass.GetType()},
		Name:   clientclass.Name,
		Regexp: fmt.Sprintf(ClientClassOption60, clientclass.Regexp),
	})

	if err != nil {
		return fmt.Errorf("marshal update clientclass request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.UpdateClientClass4, req)
}

func (s *ClientClassHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	clientclassID := ctx.Resource.(*resource.ClientClass).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(TableClientClass, map[string]interface{}{restdb.IDField: clientclassID}); err != nil {
			return err
		}

		return sendDeleteClientClassCmdToDDIAgent(ctx.Resource.(*resource.ClientClass))
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete clientclass %s failed: %s", clientclassID, err.Error()))
	}

	return nil
}

func sendDeleteClientClassCmdToDDIAgent(clientClass *resource.ClientClass) error {
	req, err := proto.Marshal(&pb.DeleteClientClass4Request{
		Header: &pb.DDIRequestHead{Method: "Delete", Resource: clientClass.GetType()},
		Name:   clientClass.GetID(),
	})

	if err != nil {
		return fmt.Errorf("marshal delete clientclass request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.DeleteClientClass4, req)
}
