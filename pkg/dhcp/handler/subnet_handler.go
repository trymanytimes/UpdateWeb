package handler

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/linkingthing/ddi-controller/pkg/auth/authentification"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	"github.com/linkingthing/ddi-controller/pkg/grpcclient"
	"github.com/linkingthing/ddi-controller/pkg/kafkaproducer"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	FileNameIpnet = "ipnet"
)

var (
	TableSubnet = restdb.ResourceDBType(&resource.Subnet{})
)

type SubnetHandler struct {
}

func NewSubnetHandler() *SubnetHandler {
	return &SubnetHandler{}
}

func (s *SubnetHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet)
	if err := checkSubnetValid(subnet); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("create subnet params invalid: %s", err.Error()))
	}

	if !authentification.PrefixFilter(ctx, subnet.Ipnet) {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, fmt.Sprintf("the subnet is not allow for creating"))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var subnets []*resource.Subnet
		if err := tx.Fill(map[string]interface{}{"offset": 0, "limit": 1, "orderby": "subnet_id desc"}, &subnets); err != nil {
			return fmt.Errorf("get max subnet id from db failed: %s\n", err.Error())
		}

		subnet.SubnetId = 1
		if len(subnets) == 1 {
			subnet.SubnetId = subnets[0].SubnetId + 1
		}

		subnet.SetID(strconv.Itoa(int(subnet.SubnetId)))
		if _, err := tx.Insert(subnet); err != nil {
			return err
		}

		return sendCreateSubnetCmdToDDIAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create subnet %s failed: %s", subnet.Ipnet, err.Error()))
	}

	return subnet, nil
}

func checkSubnetValid(subnet *resource.Subnet) error {
	ip, ipcidr, err := net.ParseCIDR(subnet.Ipnet)
	if err != nil {
		return fmt.Errorf("subnet %s invalid: %s", subnet.Ipnet, err.Error())
	} else if ip.To4() != nil {
		subnet.Version = resource.Version4
		if ones, _ := ipcidr.Mask.Size(); ones > 24 {
			return fmt.Errorf("subnet %s invalid: ip mask size %d is bigger than 24", subnet.Ipnet, ones)
		}
	} else {
		subnet.Version = resource.Version6
		if ones, _ := ipcidr.Mask.Size(); ones > 64 {
			return fmt.Errorf("subnet %s invalid: ip mask size %d is bigger than 64", subnet.Ipnet, ones)
		}
	}

	subnet.Ipnet = ipcidr.String()
	if err := setSubnetDefaultValue(subnet); err != nil {
		return err
	}

	return checkSubnetParamsValid(subnet)
}

func setSubnetDefaultValue(subnet *resource.Subnet) error {
	var configs []*resource.DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &configs)
	}); err != nil {
		return fmt.Errorf("get dhcp global config failed: %s", err.Error())
	}

	defaultValidLifetime := DefaultValidLifetime
	defaultMinLifetime := DefaultMinValidLifetime
	defaultMaxLifetime := DefaultMaxValidLifetime
	var defaultDomains []string
	if len(configs) != 0 {
		defaultValidLifetime = configs[0].ValidLifetime
		defaultMinLifetime = configs[0].MinValidLifetime
		defaultMaxLifetime = configs[0].MaxValidLifetime
		for _, domain := range configs[0].DomainServers {
			if ip := net.ParseIP(domain); ip != nil {
				if subnet.Version == resource.Version4 && ip.To4() != nil {
					defaultDomains = append(defaultDomains, domain)
				} else if subnet.Version == resource.Version6 && ip.To4() == nil {
					defaultDomains = append(defaultDomains, domain)
				}
			}
		}
	}

	if subnet.ValidLifetime == 0 {
		subnet.ValidLifetime = defaultValidLifetime
	}

	if subnet.MinValidLifetime == 0 {
		subnet.MinValidLifetime = defaultMinLifetime
	}

	if subnet.MaxValidLifetime == 0 {
		subnet.MaxValidLifetime = defaultMaxLifetime
	}

	if len(subnet.DomainServers) == 0 {
		subnet.DomainServers = defaultDomains
	}

	return nil
}

func checkSubnetParamsValid(subnet *resource.Subnet) error {
	if err := checkIPsValid(subnet.DomainServers, subnet.Version); err != nil {
		return fmt.Errorf("subnet domain servers invalid: %s", err.Error())
	}

	if err := checkIPsValid(subnet.Routers, subnet.Version); err != nil {
		return fmt.Errorf("subnet routers invalid: %s", err.Error())
	}

	if err := checkIPsValid(subnet.RelayAgentAddresses, subnet.Version); err != nil {
		return fmt.Errorf("subnet relay agent addresses invalid: %s", err.Error())
	}

	if err := checkClientClassValid(subnet.ClientClass); err != nil {
		return fmt.Errorf("subnet client class %s invalid: %s", subnet.ClientClass, err.Error())
	}

	return checkLifetimeValid(subnet.ValidLifetime, subnet.MinValidLifetime, subnet.MaxValidLifetime)
}

func checkIPsValid(ips []string, version resource.Version) error {
	for _, ip := range ips {
		if ip_ := net.ParseIP(ip); ip_ == nil {
			return fmt.Errorf("ip %s is invalid", ip)
		} else if version == resource.Version4 && ip_.To4() == nil {
			return fmt.Errorf("ip %s is invalid, it should be ipv4", ip)
		} else if version == resource.Version6 && ip_.To4() != nil {
			return fmt.Errorf("ip %s is invalid, it should be ipv6", ip)
		}
	}

	return nil
}

func checkClientClassValid(clientClass string) error {
	if clientClass == "" {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(TableClientClass, map[string]interface{}{"name": clientClass}); err != nil {
			return err
		} else if exists == false {
			return fmt.Errorf("no found client class %s in db", clientClass)
		} else {
			return nil
		}
	})
}

func sendCreateSubnetCmdToDDIAgent(subnet *resource.Subnet) error {
	cmd := kafkaconsumer.CreateSubnet4
	var req []byte
	var err error
	if subnet.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.CreateSubnet4Request{
			Header:              &pb.DDIRequestHead{Method: "Create", Resource: subnet.GetType()},
			Id:                  subnet.SubnetId,
			Ipnet:               subnet.Ipnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			DomainServers:       subnet.DomainServers,
			Routers:             subnet.Routers,
			ClientClass:         subnet.ClientClass,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
	} else {
		cmd = kafkaconsumer.CreateSubnet6
		req, err = proto.Marshal(&pb.CreateSubnet6Request{
			Header:              &pb.DDIRequestHead{Method: "Create", Resource: subnet.GetType()},
			Id:                  subnet.SubnetId,
			Ipnet:               subnet.Ipnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			DnsServers:          subnet.DomainServers,
			ClientClass:         subnet.ClientClass,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal create subnet request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func (s *SubnetHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var conditions map[string]interface{}
	if ipnet, ok := util.GetFilterValueWithEqModifierFromFilters(FileNameIpnet, ctx.GetFilters()); ok {
		conditions = map[string]interface{}{FileNameIpnet: ipnet}
	}

	var subnets resource.Subnets
	if err := db.GetResources(conditions, &subnets); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	for _, subnet := range subnets {
		if err := setSubnetLeasesUsedRatio(subnet); err != nil {
			log.Warnf("get subnet %s lease used ratio failed: %s", subnet.GetID(), err.Error())
		}
	}

	sort.Sort(subnets)
	var visible resource.Subnets
	for _, subnet := range subnets {
		if authentification.PrefixFilter(ctx, subnet.Ipnet) {
			visible = append(visible, subnet)
		}
	}
	return visible, nil
}

func (s *SubnetHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.(*resource.Subnet).GetID()
	var subnets []*resource.Subnet
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s from db failed: %s", subnetID, err.Error()))
	}

	subnet := subnetInterface.(*resource.Subnet)
	if err := setSubnetLeasesUsedRatio(subnet); err != nil {
		log.Warnf("get subnet %s leases used ratio failed: %s", subnetID, err.Error())
	}

	return subnet, nil
}

func setSubnetLeasesUsedRatio(subnet *resource.Subnet) error {
	if subnet.Capacity == 0 {
		return nil
	}

	var resp *pb.GetLeasesCountResponse
	var err error
	if subnet.Version == resource.Version4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet4LeasesCount(context.TODO(),
			&pb.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet6LeasesCount(context.TODO(),
			&pb.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
	}

	if err != nil {
		return err
	}

	if resp.Succeed {
		subnet.UsedRatio = fmt.Sprintf("%.4f", float64(resp.GetLeasesCount())/float64(subnet.Capacity))
	}

	return nil
}

func (s *SubnetHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet)
	if err := checkSubnetParamsValid(subnet); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("update subnet params invalid: %s", err.Error()))
	}

	if !authentification.PrefixFilter(ctx, subnet.Ipnet) {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, fmt.Sprintf("the subnet is not allow for updating"))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(TableSubnet, map[string]interface{}{
			"validLifetime":       subnet.ValidLifetime,
			"maxValidLifetime":    subnet.MaxValidLifetime,
			"minValidLifetime":    subnet.MinValidLifetime,
			"domainServers":       subnet.DomainServers,
			"routers":             subnet.Routers,
			"clientClass":         subnet.ClientClass,
			"relayAgentAddresses": subnet.RelayAgentAddresses,
			"tags":                subnet.Tags,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendUpdateSubnetCmdToDDIAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return subnet, nil
}

func setSubnetFromDB(tx restdb.Transaction, subnet *resource.Subnet) error {
	var subnets []*resource.Subnet
	subnetInterface, err := getResourceWithIDTx(tx, subnet.GetID(), &subnets)
	if err != nil {
		return fmt.Errorf("get subnet %s from db failed: %s", subnet.GetID(), err.Error())
	}

	s := subnetInterface.(*resource.Subnet)
	subnet.Version = s.Version
	subnet.SubnetId = s.SubnetId
	subnet.Capacity = s.Capacity
	subnet.Ipnet = s.Ipnet
	return nil
}

func sendUpdateSubnetCmdToDDIAgent(subnet *resource.Subnet) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.UpdateSubnet4
	if subnet.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.UpdateSubnet4Request{
			Header:              &pb.DDIRequestHead{Method: "Update", Resource: subnet.GetType()},
			Id:                  subnet.SubnetId,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			DomainServers:       subnet.DomainServers,
			Routers:             subnet.Routers,
			ClientClass:         subnet.ClientClass,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
	} else {
		cmd = kafkaconsumer.UpdateSubnet6
		req, err = proto.Marshal(&pb.UpdateSubnet6Request{
			Header:              &pb.DDIRequestHead{Method: "Update", Resource: subnet.GetType()},
			Id:                  subnet.SubnetId,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			DnsServers:          subnet.DomainServers,
			ClientClass:         subnet.ClientClass,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal update subnet request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func (s *SubnetHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Delete(TableSubnet, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendDeleteSubnetCmdToDDIAgent(subnet)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return nil
}

func sendDeleteSubnetCmdToDDIAgent(subnet *resource.Subnet) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.DeleteSubnet4
	if subnet.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.DeleteSubnet4Request{
			Header: &pb.DDIRequestHead{Method: "Delete", Resource: subnet.GetType()},
			Id:     subnet.SubnetId})
	} else {
		cmd = kafkaconsumer.DeleteSubnet6
		req, err = proto.Marshal(&pb.DeleteSubnet6Request{
			Header: &pb.DDIRequestHead{Method: "Delete", Resource: subnet.GetType()},
			Id:     subnet.SubnetId})
	}

	if err != nil {
		return fmt.Errorf("marshal delete subnet %s request failed: %s", subnet.GetID(), err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}
