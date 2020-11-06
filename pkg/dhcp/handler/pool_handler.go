package handler

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	"github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	"github.com/linkingthing/ddi-controller/pkg/grpcclient"
	"github.com/linkingthing/ddi-controller/pkg/kafkaproducer"
)

const (
	MaxUint64 uint64 = 1844674407370955165
)

var (
	TablePool = restdb.ResourceDBType(&resource.Pool{})
)

type PoolHandler struct {
}

func NewPoolHandler() *PoolHandler {
	return &PoolHandler{}
}

func (s *PoolHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pool := ctx.Resource.(*resource.Pool)
	if err := checkPoolValid(pool); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("create pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnetIfCanCreateDynamicPool(subnet); err != nil {
			return err
		}

		if checkIPsBelongsToSubnet(subnet.Ipnet, pool.BeginAddress, pool.EndAddress) == false {
			return fmt.Errorf("pool %s not belongs to subnet %s", poolToString(pool), subnet.Ipnet)
		}

		if conflictPool, conflict, err := checkPoolConflictWithSubnetPools(tx, subnet.GetID(), pool); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("pool %s conflicts with pool %s in subnet %s", poolToString(pool), conflictPool, subnet.GetID())
		}

		if staticAddress, conflict, err := checkPoolConflictWithSubnetStaticAddress(tx, subnet.GetID(), pool); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("pool %s conflicts with static address %s in subnet %s", poolToString(pool), staticAddress, subnet.GetID())
		}

		if err := recalculatePoolCapacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(TableSubnet, map[string]interface{}{
			"capacity": subnet.Capacity + pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		pool.Subnet = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreatePoolCmdToDDIAgent(subnet.SubnetId, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("create pool %s with subnet %s failed: %s",
			poolToString(pool), subnet.GetID(), err.Error()))
	}

	return pool, nil
}

func checkPoolValid(pool *resource.Pool) error {
	begin := net.ParseIP(pool.BeginAddress)
	if begin == nil {
		return fmt.Errorf("pool begin address %s is invalid", pool.BeginAddress)
	}

	end := net.ParseIP(pool.EndAddress)
	if end == nil {
		return fmt.Errorf("pool end address %s is invalid", pool.EndAddress)
	}

	if (begin.To4() == nil && end.To4() != nil) || (begin.To4() != nil && end.To4() == nil) {
		return fmt.Errorf("pool begin address %s type is diff from end address %s", pool.BeginAddress, pool.EndAddress)
	}

	if begin.To4() != nil {
		pool.Version = resource.Version4
		pool.Capacity = ipv4PoolCapacity(begin, end)
	} else {
		pool.Version = resource.Version6
		pool.Capacity = ipv6PoolCapacity(begin, end)
	}

	if pool.Capacity <= 0 {
		return fmt.Errorf("invalid pool capacity with begin-address %s and end-address %s", pool.BeginAddress, pool.EndAddress)
	}

	pool.BeginAddress = begin.String()
	pool.EndAddress = end.String()
	return checkPoolParamsValid(pool)
}

func checkPoolParamsValid(pool *resource.Pool) error {
	if err := checkIPsValid(pool.DomainServers, pool.Version); err != nil {
		return fmt.Errorf("pool domain servers invalid: %s", err.Error())
	}

	if err := checkIPsValid(pool.Routers, pool.Version); err != nil {
		return fmt.Errorf("pool routers invalid: %s", err.Error())
	}

	if err := checkClientClassValid(pool.ClientClass); err != nil {
		return fmt.Errorf("pool client class %s invalid: %s", pool.ClientClass, err.Error())
	}

	return nil
}

func ipv4PoolCapacity(begin, end net.IP) uint64 {
	return uint64(util.Ipv4ToUint32(end) - util.Ipv4ToUint32(begin) + 1)
}

func ipv6PoolCapacity(begin, end net.IP) uint64 {
	beginBigInt := util.Ipv6ToBigInt(begin)
	endBigInt := util.Ipv6ToBigInt(end)
	capacity := big.NewInt(0).Sub(endBigInt, beginBigInt)
	if capacity_ := big.NewInt(0).Add(capacity, big.NewInt(1)); capacity_.IsUint64() {
		return capacity_.Uint64()
	} else {
		return MaxUint64
	}

}

func getResourceWithIDTx(tx restdb.Transaction, id string, out interface{}) (interface{}, error) {
	if err := tx.Fill(map[string]interface{}{restdb.IDField: id}, out); err != nil {
		return nil, err
	}

	sliceVal := reflect.ValueOf(out).Elem()
	if sliceVal.Len() == 1 {
		return sliceVal.Index(0).Interface(), nil
	} else {
		return nil, fmt.Errorf("no found")
	}
}

func checkSubnetIfCanCreateDynamicPool(subnet *resource.Subnet) error {
	if subnet.Version == resource.Version6 {
		_, ipcidr, _ := net.ParseCIDR(subnet.Ipnet)
		if ones, _ := ipcidr.Mask.Size(); ones != 64 {
			return fmt.Errorf("only subnet which mask len is 64 can create dynamic pool, current mask len is %d", ones)
		}
	}

	return nil
}

func checkPoolConflictWithSubnetPools(tx restdb.Transaction, subnetID string, pool *resource.Pool) (string, bool, error) {
	conflictPool, conflict, err := checkPoolConflictWithSubnetPool(tx, subnetID, pool)
	if err != nil || conflict {
		return poolToString(conflictPool), conflict, err
	}

	return checkIPConflictWithSubnetPDPool(tx, subnetID, pool.BeginAddress)
}

func checkPoolConflictWithSubnetPool(tx restdb.Transaction, subnetID string, pool *resource.Pool) (*resource.Pool, bool, error) {
	var pools []*resource.Pool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pools); err != nil {
		return nil, false, fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, p := range pools {
		if p.CheckConflictWithAnother(pool) {
			return p, true, nil
		}
	}

	return nil, false, nil
}

func poolToString(pool *resource.Pool) string {
	return pool.BeginAddress + "-" + pool.EndAddress
}

func checkIPConflictWithSubnetPDPool(tx restdb.Transaction, subnetID, ip string) (string, bool, error) {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pdpools); err != nil {
		return "", false, fmt.Errorf("get pdpools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, pdpool := range pdpools {
		subnet := pdpoolToSubnetStr(pdpool)
		if checkIPsBelongsToSubnet(subnet, ip) {
			return subnet, true, nil
		}
	}

	return "", false, nil
}

func checkIPsBelongsToSubnet(subnet string, ips ...string) bool {
	_, ipcidr, _ := net.ParseCIDR(subnet)
	for _, ip := range ips {
		if ipcidr.Contains(net.ParseIP(ip)) == false {
			return false
		}
	}

	return true
}

func checkPoolConflictWithSubnetStaticAddress(tx restdb.Transaction, subnetID string, pool *resource.Pool) (string, bool, error) {
	var staticAddresses []*resource.StaticAddress
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &staticAddresses); err != nil {
		return "", false, err
	}

	for _, staticAddress := range staticAddresses {
		if pool.Contains(staticAddress.IpAddress) {
			return staticAddressToString(staticAddress), true, nil
		}
	}

	return "", false, nil
}

func recalculatePoolCapacity(tx restdb.Transaction, subnetID string, pool *resource.Pool) error {
	var reservations []*resource.Reservation
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &reservations); err != nil {
		return err
	}

	for _, reservation := range reservations {
		if pool.Contains(reservation.IpAddress) {
			pool.Capacity -= reservation.Capacity
		}
	}

	return nil
}

func sendCreatePoolCmdToDDIAgent(subnetID uint32, pool *resource.Pool) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.CreatePool4
	if pool.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.CreatePool4Request{
			Header:        &pb.DDIRequestHead{Method: "Create", Resource: pool.GetType()},
			SubnetId:      subnetID,
			BeginAddress:  pool.BeginAddress,
			EndAddress:    pool.EndAddress,
			DomainServers: pool.DomainServers,
			Routers:       pool.Routers,
			ClientClass:   pool.ClientClass,
		})
	} else {
		cmd = kafkaconsumer.CreatePool6
		req, err = proto.Marshal(&pb.CreatePool6Request{
			Header:       &pb.DDIRequestHead{Method: "Create", Resource: pool.GetType()},
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			DnsServers:   pool.DomainServers,
			ClientClass:  pool.ClientClass,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal create pool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func subnetIDStrToUint32(subnetID string) uint32 {
	id, _ := strconv.Atoi(subnetID)
	return uint32(id)
}

func (s *PoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pools resource.Pools
	if err := db.GetResources(map[string]interface{}{"subnet": subnetID}, &pools); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pools with subnet %s from db failed: %s", subnetID, err.Error()))
	}

	for _, pool := range pools {
		if err := setPoolLeasesUsedRatio(pool); err != nil {
			log.Warnf("get pool %s with subnet %s leases used ratio failed: %s", poolToString(pool), subnetID, err.Error())
		}
	}

	sort.Sort(pools)
	return pools, nil
}

func (s *PoolHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.(*resource.Pool).GetID()
	var pools []*resource.Pool
	poolInterface, err := restdb.GetResourceWithID(db.GetDB(), poolID, &pools)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get pool %s with subnet %s from db failed: %s",
			poolID, subnetID, err.Error()))
	}

	pool := poolInterface.(*resource.Pool)
	if err := setPoolLeasesUsedRatio(pool); err != nil {
		log.Warnf("get pool %s with subnet %s leases used ratio failed: %s", poolToString(pool), subnetID, err.Error())
	}

	return pool, nil
}

func setPoolLeasesUsedRatio(pool *resource.Pool) error {
	if pool.Capacity == 0 {
		return nil
	}

	var resp *pb.GetLeasesCountResponse
	var err error
	if pool.Version == resource.Version4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetPool4LeasesCount(context.TODO(),
			&pb.GetPool4LeasesCountRequest{
				SubnetId:     subnetIDStrToUint32(pool.Subnet),
				BeginAddress: pool.BeginAddress,
				EndAddress:   pool.EndAddress,
			})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetPool6LeasesCount(context.TODO(),
			&pb.GetPool6LeasesCountRequest{
				SubnetId:     subnetIDStrToUint32(pool.Subnet),
				BeginAddress: pool.BeginAddress,
				EndAddress:   pool.EndAddress,
			})
	}

	if err != nil {
		return err
	}

	if resp.Succeed {
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(resp.GetLeasesCount())/float64(pool.Capacity))
	}

	return nil
}

func (s *PoolHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pool := ctx.Resource.(*resource.Pool)
	if err := checkPoolParamsValid(pool); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("update pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setPoolFromDB(tx, pool); err != nil {
			return err
		}

		if _, err := tx.Update(TablePool, map[string]interface{}{
			"domainServers": pool.DomainServers,
			"routers":       pool.Routers,
			"clientClass":   pool.ClientClass,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendUpdatePoolCmdToDDIAgent(subnetID, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update pool %s with subnet %s failed: %s",
			poolToString(pool), subnetID, err.Error()))
	}

	return pool, nil
}

func setPoolFromDB(tx restdb.Transaction, pool *resource.Pool) error {
	var pools []*resource.Pool
	poolInterface, err := getResourceWithIDTx(tx, pool.GetID(), &pools)
	if err != nil {
		return fmt.Errorf("get pool from db failed: %s", err.Error())
	}

	p := poolInterface.(*resource.Pool)
	pool.BeginAddress = p.BeginAddress
	pool.EndAddress = p.EndAddress
	pool.Version = p.Version
	pool.Capacity = p.Capacity
	return nil
}

func sendUpdatePoolCmdToDDIAgent(subnetID string, pool *resource.Pool) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.UpdatePool4
	if pool.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.UpdatePool4Request{
			Header:        &pb.DDIRequestHead{Method: "Update", Resource: pool.GetType()},
			SubnetId:      subnetIDStrToUint32(subnetID),
			BeginAddress:  pool.BeginAddress,
			EndAddress:    pool.EndAddress,
			DomainServers: pool.DomainServers,
			Routers:       pool.Routers,
			ClientClass:   pool.ClientClass,
		})
	} else {
		cmd = kafkaconsumer.UpdatePool6
		req, err = proto.Marshal(&pb.UpdatePool6Request{
			Header:       &pb.DDIRequestHead{Method: "Update", Resource: pool.GetType()},
			SubnetId:     subnetIDStrToUint32(subnetID),
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			DnsServers:   pool.DomainServers,
			ClientClass:  pool.ClientClass,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal update pool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func (s *PoolHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pool := ctx.Resource.(*resource.Pool)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPoolFromDB(tx, pool); err != nil {
			return err
		}

		if _, err := tx.Update(TableSubnet, map[string]interface{}{
			"capacity": subnet.Capacity - pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		if _, err := tx.Delete(TablePool, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeletePoolCmdToDDIAgent(subnet.SubnetId, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete pool %s with subnet %s failed: %s",
			poolToString(pool), subnet.GetID(), err.Error()))
	}

	return nil
}

func sendDeletePoolCmdToDDIAgent(subnetID uint32, pool *resource.Pool) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.DeletePool4
	if pool.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.DeletePool4Request{
			Header:       &pb.DDIRequestHead{Method: "Delete", Resource: pool.GetType()},
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
	} else {
		cmd = kafkaconsumer.DeletePool6
		req, err = proto.Marshal(&pb.DeletePool6Request{
			Header:       &pb.DDIRequestHead{Method: "Delete", Resource: pool.GetType()},
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal delete pool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}
