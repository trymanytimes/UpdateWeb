package handler

import (
	"fmt"
	"net"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dns/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

var (
	TableAcl  = restdb.ResourceDBType(&resource.Acl{})
	NoneACL   = "none"
	AnyACL    = "any"
	forbidden = "forbidden"
)

type ACLHandler struct{}

func NewACLHandler() (*ACLHandler, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := initAclIfNonExists(tx, NoneACL); err != nil {
			return err
		}

		return initAclIfNonExists(tx, AnyACL)
	}); err != nil {
		return nil, err
	}

	return &ACLHandler{}, nil
}

func initAclIfNonExists(tx restdb.Transaction, aclName string) error {
	if exists, err := tx.Exists(TableAcl, map[string]interface{}{"id": aclName}); err != nil {
		return err
	} else if exists == false {
		acl := &resource.Acl{Name: aclName, Status: "allow"}
		acl.SetID(aclName)
		_, err = tx.Insert(acl)
		return err
	}
	return nil
}

func (h *ACLHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	acl := ctx.Resource.(*resource.Acl)
	if len(acl.Ips) == 0 && acl.Isp == "" {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("isp or ip is null"))
	}
	acl.Name = strings.ToLower(acl.Name)
	if err := util.CheckNameValid(acl.Name); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("check the acl %s's name fail: %s", acl.Name, err.Error()))
	}
	if err := h.checkACLIPs(acl.Ips); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("check the acl %s's ip fail: %s", acl.Name, err.Error()))
	}
	acl.SetID(acl.Name)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(acl); err != nil {
			return fmt.Errorf("add acl %s to db failed: %s", acl.Name, err.Error())
		}

		var ipsData []string
		if acl.Isp == "" {
			ipsData = acl.Ips
		} else {
			ipsData = append(ipsData, acl.Isp)
		}

		return SendKafkaMessage(acl.ID, kafkaconsumer.CreateACL,
			&pb.CreateACLReq{
				Header: &pb.DDIRequestHead{Method: "Create", Resource: acl.GetType()},
				Id:     acl.GetID(),
				Name:   acl.Name,
				Ips:    h.getStatusIPs(acl.Status, ipsData)})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("Create acl %s failed: %s", acl.Name, err.Error()))
	}

	return acl, nil
}

func (h *ACLHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	aclID := ctx.Resource.(*resource.Acl).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(TableAcl, map[string]interface{}{restdb.IDField: aclID}); err != nil {
			return fmt.Errorf("delete acl %s from db failed: %s", aclID, err.Error())
		}

		return SendKafkaMessage(aclID, kafkaconsumer.DeleteACL,
			&pb.DeleteACLReq{
				Header: &pb.DDIRequestHead{Method: "Delete", Resource: ctx.Resource.(*resource.Acl).GetType()},
				Id:     aclID})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete acl failed: %s", err.Error()))
	}

	return nil
}

func (h *ACLHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	acl := ctx.Resource.(*resource.Acl)
	if err := checkDeleteACLValid(acl); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update acl %s from db failed: %s", acl.Name, err.Error()))
	}
	if err := h.checkACLIPs(acl.Ips); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("check the acl %s's ip fail: %s", acl.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableAcl, map[string]interface{}{
			"ips":     acl.Ips,
			"isp":     acl.Isp,
			"status":  acl.Status,
			"comment": acl.Comment,
		}, map[string]interface{}{restdb.IDField: acl.GetID()}); err != nil {
			return fmt.Errorf("update acl %s from db failed: %s", acl.Name, err.Error())
		}

		return SendKafkaMessage(acl.ID, kafkaconsumer.UpdateACL, &pb.UpdateACLReq{
			Header: &pb.DDIRequestHead{Method: "Update", Resource: acl.GetType()},
			Id:     acl.GetID(),
			Name:   acl.Name,
			Ips:    h.getStatusIPs(acl.Status, acl.Ips),
		})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update the acl %s fail: %s", acl.Name, err.Error()))
	}

	return acl, nil
}

func (h *ACLHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	aclid := ctx.Resource.(*resource.Acl).GetID()
	var acls []*resource.Acl
	acl, err := restdb.GetResourceWithID(db.GetDB(), aclid, &acls)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get acl %s from db failed: %s", aclid, err.Error()))
	}
	return acl.(*resource.Acl), nil
}

func (h *ACLHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var acls []*resource.Acl
	if err := db.GetResources(map[string]interface{}{"orderby": "create_time"}, &acls); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list acl from db failed: %s", err.Error()))
	}

	return acls, nil
}

func checkDeleteACLValid(acl *resource.Acl) error {
	if acl.ID == NoneACL || acl.ID == AnyACL {
		return fmt.Errorf("It's not allow to modify the default any or none acl\n")
	}
	return nil
}

func (h *ACLHandler) getStatusIPs(status string, originIPs []string) []string {
	var ips []string
	if status == forbidden {
		for _, ip := range originIPs {
			ips = append(ips, "!"+ip)
		}
	} else {
		ips = originIPs
	}
	return ips
}

func (h *ACLHandler) checkACLIPs(ips []string) error {
	for _, ip := range ips {
		if len(strings.Split(ip, "/")) == 2 {
			_, _ip, err := net.ParseCIDR(ip)
			if err != nil {
				return err
			} else if _ip == nil {
				return fmt.Errorf("ip address is not correct")
			}
			if _ip.String() != ip {
				return fmt.Errorf("ip address is not correct,could be %s", _ip.String())
			}
		} else {
			if _ip := net.ParseIP(ip); _ip == nil {
				return fmt.Errorf("ip address is not correct")
			}
		}
	}
	return nil
}
