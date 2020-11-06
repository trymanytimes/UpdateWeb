package handler

import (
	"context"
	"fmt"
	"net"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	"github.com/linkingthing/ddi-controller/pkg/grpcclient"
	"github.com/linkingthing/ddi-controller/pkg/kafkaproducer"
)

var (
	TableReservation = restdb.ResourceDBType(&resource.Reservation{})
)

type ReservationHandler struct {
}

func NewReservationHandler() *ReservationHandler {
	return &ReservationHandler{}
}

func (s *ReservationHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	reservation := ctx.Resource.(*resource.Reservation)
	if err := checkReservationValid(reservation); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("create reservation params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkMacOrIpInUsed(tx, subnet.GetID(), reservation.HwAddress, reservation.IpAddress, false); err != nil {
			return err
		}

		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnetIfCanCreateDynamicPool(subnet); err != nil {
			return err
		}

		if checkIPsBelongsToSubnet(subnet.Ipnet, reservation.IpAddress) == false {
			return fmt.Errorf("reservation ipaddress %s not belongs to subnet %s", reservation.IpAddress, subnet.Ipnet)
		}

		if pdpool, conflict, err := checkIPConflictWithSubnetPDPool(tx, subnet.GetID(), reservation.IpAddress); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("reservation ipaddress %s conflicts with pdpool %s in subnet %s",
				reservation.IpAddress, pdpool, subnet.GetID())
		}

		conflictPool, conflict, err := checkPoolConflictWithSubnetPool(tx, subnet.GetID(),
			&resource.Pool{BeginAddress: reservation.IpAddress, EndAddress: reservation.IpAddress})
		if err != nil {
			return err
		}

		if conflict == false {
			if _, err := tx.Update(TableSubnet, map[string]interface{}{
				"capacity": subnet.Capacity + 1,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(TablePool, map[string]interface{}{
				"capacity": conflictPool.Capacity - 1,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s", poolToString(conflictPool), err.Error())
			}
		}

		reservation.Capacity = 1
		reservation.Subnet = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return err
		}

		return sendCreateReservationCmdToDDIAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("create reservation with mac %s failed: %s",
			reservation.HwAddress, err.Error()))
	}

	return reservation, nil
}

func sendCreateReservationCmdToDDIAgent(subnetID uint32, reservation *resource.Reservation) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.CreateReservation4
	if reservation.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.CreateReservation4Request{
			Header:        &pb.DDIRequestHead{Method: "Create", Resource: reservation.GetType()},
			SubnetId:      subnetID,
			HwAddress:     reservation.HwAddress,
			IpAddress:     reservation.IpAddress,
			DomainServers: reservation.DomainServers,
			Routers:       reservation.Routers,
		})
	} else {
		cmd = kafkaconsumer.CreateReservation6
		req, err = proto.Marshal(&pb.CreateReservation6Request{
			Header:      &pb.DDIRequestHead{Method: "Create", Resource: reservation.GetType()},
			SubnetId:    subnetID,
			HwAddress:   reservation.HwAddress,
			IpAddresses: []string{reservation.IpAddress},
			DnsServers:  reservation.DomainServers,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal create reservation request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func reservationToString(reservation *resource.Reservation) string {
	return reservation.HwAddress + "/" + reservation.IpAddress
}

func checkReservationValid(reservation *resource.Reservation) error {
	if _, err := net.ParseMAC(reservation.HwAddress); err != nil {
		return fmt.Errorf("reservation hwaddress %s is invalid", reservation.HwAddress)
	}

	reservation.Version = resource.Version4
	if ip := net.ParseIP(reservation.IpAddress); ip == nil {
		return fmt.Errorf("reservation ipaddress %s is invalid", reservation.IpAddress)
	} else if ip.To4() == nil {
		reservation.Version = resource.Version6
	}

	if err := checkIPsValid(reservation.DomainServers, reservation.Version); err != nil {
		return fmt.Errorf("reservation domain servers invalid: %s", err.Error())
	}

	if err := checkIPsValid(reservation.Routers, reservation.Version); err != nil {
		return fmt.Errorf("reservation routers invalid: %s", err.Error())
	}

	return nil
}

func (s *ReservationHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var reservations resource.Reservations
	if err := db.GetResources(map[string]interface{}{"subnet": subnetID}, &reservations); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list reservations with subnet %s from db failed: %s",
			subnetID, err.Error()))
	}

	for _, reservation := range reservations {
		if err := setReservationLeasesUsedRatio(reservation); err != nil {
			log.Warnf("get reservation %s with subnet %s leases used ratio failed: %s",
				reservationToString(reservation), subnetID, err.Error())
		}
	}

	sort.Sort(reservations)
	return reservations, nil
}

func (s *ReservationHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservationID := ctx.Resource.(*resource.Reservation).GetID()
	var reservations []*resource.Reservation
	reservationInterface, err := restdb.GetResourceWithID(db.GetDB(), reservationID, &reservations)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get reservation %s with subnetID %s from db failed: %s",
			reservationID, subnetID, err.Error()))
	}

	reservation := reservationInterface.(*resource.Reservation)
	if err := setReservationLeasesUsedRatio(reservation); err != nil {
		log.Warnf("get reservation %s with subnet %s leases used ratio failed: %s",
			reservationToString(reservation), subnetID, err.Error())
	}
	return reservation, nil
}

func setReservationLeasesUsedRatio(reservation *resource.Reservation) error {
	if reservation.Capacity == 0 {
		return nil
	}

	var resp *pb.GetLeasesCountResponse
	var err error
	if reservation.Version == resource.Version4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetReservation4LeasesCount(context.TODO(),
			&pb.GetReservation4LeasesCountRequest{
				SubnetId:  subnetIDStrToUint32(reservation.Subnet),
				HwAddress: reservation.HwAddress,
			})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetReservation6LeasesCount(context.TODO(),
			&pb.GetReservation6LeasesCountRequest{
				SubnetId:  subnetIDStrToUint32(reservation.Subnet),
				HwAddress: reservation.HwAddress,
			})
	}

	if err != nil {
		return err
	}

	if resp.Succeed {
		reservation.UsedRatio = fmt.Sprintf("%.4f", float64(resp.GetLeasesCount())/float64(reservation.Capacity))
	}

	return nil
}

func (s *ReservationHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservation := ctx.Resource.(*resource.Reservation)
	if err := checkReservationValid(reservation); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("update reservation params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setReservationFromDB(tx, reservation); err != nil {
			return err
		}

		if _, err := tx.Update(TableReservation, map[string]interface{}{
			"domainServers": reservation.DomainServers,
			"routers":       reservation.Routers,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendUpdateReservationCmdToDDIAgent(subnetID, reservation)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update reservation %s with subnet %s failed: %s",
			reservation.GetID(), subnetID, err.Error()))
	}

	return reservation, nil
}

func setReservationFromDB(tx restdb.Transaction, reservation *resource.Reservation) error {
	var reservations []*resource.Reservation
	reservationInterface, err := getResourceWithIDTx(tx, reservation.GetID(), &reservations)
	if err != nil {
		return err
	}

	r := reservationInterface.(*resource.Reservation)
	reservation.Version = r.Version
	reservation.HwAddress = r.HwAddress
	reservation.IpAddress = r.IpAddress
	reservation.Capacity = r.Capacity
	return nil
}

func sendUpdateReservationCmdToDDIAgent(subnetID string, reservation *resource.Reservation) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.UpdateReservation4
	if reservation.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.UpdateReservation4Request{
			Header:        &pb.DDIRequestHead{Method: "Update", Resource: reservation.GetType()},
			SubnetId:      subnetIDStrToUint32(subnetID),
			HwAddress:     reservation.HwAddress,
			DomainServers: reservation.DomainServers,
			Routers:       reservation.Routers,
		})
	} else {
		cmd = kafkaconsumer.UpdateReservation6
		req, err = proto.Marshal(&pb.UpdateReservation6Request{
			Header:     &pb.DDIRequestHead{Method: "Update", Resource: reservation.GetType()},
			SubnetId:   subnetIDStrToUint32(subnetID),
			HwAddress:  reservation.HwAddress,
			DnsServers: reservation.DomainServers,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal update reservation request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func (s *ReservationHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	reservation := ctx.Resource.(*resource.Reservation)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservationFromDB(tx, reservation); err != nil {
			return err
		}

		conflictPool, conflict, err := checkPoolConflictWithSubnetPool(tx, subnet.GetID(),
			&resource.Pool{BeginAddress: reservation.IpAddress, EndAddress: reservation.IpAddress})
		if err != nil {
			return err
		} else if conflict {
			if _, err := tx.Update(TablePool, map[string]interface{}{
				"capacity": conflictPool.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s", conflictPool.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(TableSubnet, map[string]interface{}{
				"capacity": subnet.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
			}
		}

		if _, err := tx.Delete(TableReservation, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservationCmdToDDIAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete reservation %s with subnet %s failed: %s",
			reservationToString(reservation), subnet.GetID(), err.Error()))
	}

	return nil
}

func sendDeleteReservationCmdToDDIAgent(subnetID uint32, reservation *resource.Reservation) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.DeleteReservation4
	if reservation.Version == resource.Version4 {
		req, err = proto.Marshal(&pb.DeleteReservation4Request{
			Header:    &pb.DDIRequestHead{Method: "Delete", Resource: reservation.GetType()},
			SubnetId:  subnetID,
			HwAddress: reservation.HwAddress,
		})
	} else {
		cmd = kafkaconsumer.DeleteReservation6
		req, err = proto.Marshal(&pb.DeleteReservation6Request{
			Header:    &pb.DDIRequestHead{Method: "Delete", Resource: reservation.GetType()},
			SubnetId:  subnetID,
			HwAddress: reservation.HwAddress,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal delete reservation request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}
