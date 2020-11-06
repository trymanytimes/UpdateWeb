package handler

import (
	"fmt"
	"net"
	"sort"
	"strconv"

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

var (
	TablePdPool = restdb.ResourceDBType(&resource.PdPool{})
)

type PdPoolHandler struct {
}

func NewPdPoolHandler() *PdPoolHandler {
	return &PdPoolHandler{}
}

func (s *PdPoolHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := checkPdPoolValid(pdpool); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		_, ipcidr, _ := net.ParseCIDR(subnet.Ipnet)
		if ones, _ := ipcidr.Mask.Size(); uint32(ones) > pdpool.PrefixLen {
			return fmt.Errorf("pdpool %s prefix len %d should bigger than subnet mask len %d", pdpool.Prefix, pdpool.PrefixLen, ones)
		}

		if checkIPsBelongsToSubnet(subnet.Ipnet, pdpool.Prefix) == false {
			return fmt.Errorf("pdpool %s not belongs to subnet %s", pdpool.Prefix, subnet.Ipnet)
		}

		if conflictPool, conflict, err := checkPdPoolConflictWithSubnetPools(tx, subnet.GetID(), pdpool); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("pdpool %s conflict with pool %s in subnet %s", pdpoolToSubnetStr(pdpool), conflictPool, subnet.GetID())
		}

		pdpool.Subnet = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return err
		}

		return sendCreatePDPoolCmdToDDIAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("create pdpool %s-%d with subnet %s failed: %s",
			pdpoolToSubnetStr(pdpool), pdpool.DelegatedLen, subnet.GetID(), err.Error()))
	}

	return pdpool, nil
}

func checkPdPoolValid(pdpool *resource.PdPool) error {
	prefix := net.ParseIP(pdpool.Prefix)
	if prefix == nil || prefix.To4() != nil {
		return fmt.Errorf("pdpool prefix %s is invalid", pdpool.Prefix)
	}

	if pdpool.PrefixLen >= 128 {
		return fmt.Errorf("pdpool prefix len %d should not bigger than 128)", pdpool.PrefixLen)
	}

	if pdpool.DelegatedLen < pdpool.PrefixLen || pdpool.DelegatedLen > 128 {
		return fmt.Errorf("pdpool delegated len %d not in (%d, 128]", pdpool.DelegatedLen, pdpool.PrefixLen)
	}

	if err := checkIPsValid(pdpool.DomainServers, resource.Version6); err != nil {
		return fmt.Errorf("pdpool domain servers invalid: %s", err.Error())
	}

	if err := checkClientClassValid(pdpool.ClientClass); err != nil {
		return fmt.Errorf("pdpool client class %s invalid: %s", pdpool.ClientClass, err.Error())
	}

	pdpool.Prefix = prefix.String()
	return nil
}

func checkPdPoolConflictWithSubnetPools(tx restdb.Transaction, subnetID string, pdpool *resource.PdPool) (string, bool, error) {
	subnet := pdpoolToSubnetStr(pdpool)
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pdpools); err != nil {
		return "", false, fmt.Errorf("get pdpools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, pdpool_ := range pdpools {
		subnet_ := pdpoolToSubnetStr(pdpool_)
		if checkIPsBelongsToSubnet(subnet_, pdpool.Prefix) || checkIPsBelongsToSubnet(subnet, pdpool_.Prefix) {
			return subnet_, true, nil
		}
	}

	var pools []*resource.Pool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pools); err != nil {
		return "", false, fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, pool := range pools {
		if pool.Version == resource.Version6 {
			if checkIPsBelongsToSubnet(subnet, pool.BeginAddress) {
				return poolToString(pool), true, nil
			}
		}
	}

	var reservations []*resource.Reservation
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &reservations); err != nil {
		return "", false, fmt.Errorf("get reservations with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, reservation := range reservations {
		if reservation.Version == resource.Version6 {
			if checkIPsBelongsToSubnet(subnet, reservation.IpAddress) {
				return reservationToString(reservation), true, nil
			}
		}
	}

	var staticAddresses []*resource.StaticAddress
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &staticAddresses); err != nil {
		return "", false, fmt.Errorf("get static addresses with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, staticAddress := range staticAddresses {
		if staticAddress.Version == resource.Version6 {
			if checkIPsBelongsToSubnet(subnet, staticAddress.IpAddress) {
				return staticAddressToString(staticAddress), true, nil
			}
		}
	}

	return "", false, nil
}

func pdpoolToSubnetStr(pdpool *resource.PdPool) string {
	return pdpool.Prefix + "/" + strconv.Itoa(int(pdpool.PrefixLen))
}

func sendCreatePDPoolCmdToDDIAgent(subnetID uint32, pdpool *resource.PdPool) error {
	req, err := proto.Marshal(&pb.CreatePDPoolRequest{
		Header:       &pb.DDIRequestHead{Method: "Create", Resource: pdpool.GetType()},
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
		DnsServers:   pdpool.DomainServers,
		ClientClass:  pdpool.ClientClass,
	})

	if err != nil {
		return fmt.Errorf("marshal create pdpool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.CreatePDPool, req)
}

func (s *PdPoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pdpools resource.PdPools
	if err := db.GetResources(map[string]interface{}{"subnet": subnetID}, &pdpools); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list pdpools with subnet %s from db failed: %s",
			subnetID, err.Error()))
	}

	sort.Sort(pdpools)
	return pdpools, nil
}

func (s *PdPoolHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpoolID := ctx.Resource.(*resource.PdPool).GetID()
	var pdpools []*resource.PdPool
	pdpool, err := restdb.GetResourceWithID(db.GetDB(), pdpoolID, &pdpools)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get pdpool %s with subnet %s from db failed: %s",
			pdpoolID, subnetID, err.Error()))
	}

	return pdpool.(*resource.PdPool), nil
}

func (s *PdPoolHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := checkPdPoolValid(pdpool); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("update pdpool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		if _, err := tx.Update(TablePdPool, map[string]interface{}{
			"domainServers": pdpool.DomainServers,
			"clientClass":   pdpool.ClientClass,
		}, map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendUpdatePdPoolCmdToDDIAgent(subnetID, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update pdpool %s with subnet %s failed: %s",
			pdpoolToSubnetStr(pdpool), subnetID, err.Error()))
	}

	return pdpool, nil
}

func setPdPoolFromDB(tx restdb.Transaction, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	pdpoolInterface, err := getResourceWithIDTx(tx, pdpool.GetID(), &pdpools)
	if err != nil {
		return fmt.Errorf("get pool from db failed: %s", err.Error())
	}

	pd := pdpoolInterface.(*resource.PdPool)
	pdpool.Prefix = pd.Prefix
	return nil
}

func sendUpdatePdPoolCmdToDDIAgent(subnetID string, pdpool *resource.PdPool) error {
	req, err := proto.Marshal(&pb.UpdatePDPoolRequest{
		Header:      &pb.DDIRequestHead{Method: "Update", Resource: pdpool.GetType()},
		SubnetId:    subnetIDStrToUint32(subnetID),
		Prefix:      pdpool.Prefix,
		DnsServers:  pdpool.DomainServers,
		ClientClass: pdpool.ClientClass,
	})

	if err != nil {
		return fmt.Errorf("marshal update pdpool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.UpdatePDPool, req)
}

func (s *PdPoolHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		if _, err := tx.Delete(TablePdPool, map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendDeletePdPoolCmdToDDIAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
			pdpoolToSubnetStr(pdpool), subnet.GetID(), err.Error()))
	}

	return nil
}

func sendDeletePdPoolCmdToDDIAgent(subnetID uint32, pdpool *resource.PdPool) error {
	req, err := proto.Marshal(&pb.DeletePDPoolRequest{
		Header:   &pb.DDIRequestHead{Method: "Delete", Resource: pdpool.GetType()},
		SubnetId: subnetID,
		Prefix:   pdpool.Prefix,
	})

	if err != nil {
		return fmt.Errorf("marshal delete pdpool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.DeletePDPool, req)
}
