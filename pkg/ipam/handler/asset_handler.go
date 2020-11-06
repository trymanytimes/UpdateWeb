package handler

import (
	"fmt"
	"net"

	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"
	goresterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/auth/authentification"
	"github.com/linkingthing/ddi-controller/pkg/db"
	dhcpresource "github.com/linkingthing/ddi-controller/pkg/dhcp/resource"
	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

const (
	FilterNameMac        = "mac"
	FilterNameIp         = "ip"
	FilterNameName       = "name"
	FilterNameSwitchName = "switch_name"
)

var (
	TableAsset = restdb.ResourceDBType(&ipamresource.Asset{})
)

type AssetHandler struct{}

func NewAssetHandler() *AssetHandler {
	return &AssetHandler{}
}

func (h *AssetHandler) Create(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	asset := ctx.Resource.(*ipamresource.Asset)
	if err := asset.Validate(); err != nil {
		return nil, goresterr.NewAPIError(goresterr.InvalidBodyContent, err.Error())
	}

	if !authentification.PrefixFilter(ctx, append(asset.Ipv4s, asset.Ipv6s...)...) {
		return nil, goresterr.NewAPIError(goresterr.PermissionDenied, fmt.Sprintf("the asset is not allow for creating"))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := checkDuplicateIp(tx, asset)
		if err != nil {
			return err
		}

		_, err = tx.Insert(asset)
		return err
	}); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("create asset failed: %s", err.Error()))
	}

	return asset, nil
}

func checkDuplicateIp(tx restdb.Transaction, asset *ipamresource.Asset) error {
	var assets []*ipamresource.Asset
	err := tx.Fill(nil, &assets)
	if err != nil {
		return err
	}

	if mac, ip, exists := getDuplicateIp(assets, asset); exists {
		return fmt.Errorf("asset ip %s is duplicate, it exists in asset with mac %s", ip, mac)
	}

	return nil
}

func getDuplicateIp(assets []*ipamresource.Asset, asset *ipamresource.Asset) (string, string, bool) {
	if mac, ip, exists := getDuplicateIpWithIps(assets, asset.GetID(), asset.Ipv4s, true); exists {
		return mac, ip, exists
	}

	return getDuplicateIpWithIps(assets, asset.GetID(), asset.Ipv6s, false)
}

func getDuplicateIpWithIps(assets []*ipamresource.Asset, assetId string, ips []string, isV4 bool) (string, string, bool) {
	for _, ip := range ips {
		for _, asset_ := range assets {
			if asset_.GetID() == assetId {
				continue
			}

			ips_ := asset_.Ipv4s
			if isV4 == false {
				ips_ = asset_.Ipv6s
			}
			if slice.SliceIndex(ips_, ip) != -1 {
				return asset_.Mac, ip, true
			}
		}
	}

	return "", "", false
}

func (h *AssetHandler) List(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	var assets []*ipamresource.Asset
	err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(util.GenStrConditionsFromFilters(ctx.GetFilters(), FilterNameMac, FilterNameName, FilterNameSwitchName), &assets)
	})

	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	}

	if ip, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNameIp, ctx.GetFilters()); ok {
		assets = getAssetsByIP(assets, ip)
	}

	var visible []*ipamresource.Asset
	for _, asset := range assets {
		if authentification.PrefixFilter(ctx, append(asset.Ipv4s, asset.Ipv6s...)...) {
			visible = append(visible, asset)
		}
	}
	return visible, nil
}

func getAssetsByIP(assets []*ipamresource.Asset, ipstr string) []*ipamresource.Asset {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil
	}
	isv6 := ip.To4() == nil

	var retAssets []*ipamresource.Asset
	for _, asset := range assets {
		ips := asset.Ipv4s
		if isv6 {
			ips = asset.Ipv6s
		}
		if slice.SliceIndex(ips, ipstr) != -1 {
			retAssets = append(retAssets, asset)
			break
		}
	}

	return retAssets
}

func (h *AssetHandler) Get(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	var assets []*ipamresource.Asset
	asset, err := restdb.GetResourceWithID(db.GetDB(), ctx.Resource.GetID(), &assets)
	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return asset.(resource.Resource), nil
	}
}

func (h *AssetHandler) Delete(ctx *resource.Context) *goresterr.APIError {
	err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(TableAsset, map[string]interface{}{"id": ctx.Resource.GetID()})
		return err
	})
	if err != nil {
		return goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return nil
	}
}

func (h *AssetHandler) Update(ctx *resource.Context) (resource.Resource, *goresterr.APIError) {
	newAsset := ctx.Resource.(*ipamresource.Asset)
	if err := newAsset.Validate(); err != nil {
		return nil, goresterr.NewAPIError(goresterr.InvalidBodyContent, err.Error())
	}

	if !authentification.PrefixFilter(ctx, append(newAsset.Ipv4s, newAsset.Ipv6s...)...) {
		return nil, goresterr.NewAPIError(goresterr.PermissionDenied, fmt.Sprintf("the asset is not allow for updating"))
	}

	err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkDuplicateIp(tx, newAsset); err != nil {
			return err
		}

		newData, _ := restdb.ResourceToMap(newAsset)
		modified, err := tx.Update(
			TableAsset,
			newData,
			map[string]interface{}{"id": newAsset.GetID()},
		)
		if err != nil {
			return err
		} else if modified == 0 {
			return fmt.Errorf("asset is unknown")
		} else {
			return nil
		}
	})

	if err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, err.Error())
	} else {
		return newAsset, nil
	}
}

func (h *AssetHandler) Action(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	switch ctx.Resource.GetAction().Name {
	case ipamresource.ActionNameRegister:
		return h.register(ctx)
	default:
		return nil, goresterr.NewAPIError(goresterr.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *AssetHandler) register(ctx *resource.Context) (interface{}, *goresterr.APIError) {
	assetRegister, ok := ctx.Resource.GetAction().Input.(*ipamresource.AssetRegister)
	if ok == false {
		return nil, goresterr.NewAPIError(goresterr.InvalidFormat, fmt.Sprintf("parse action register input invalid"))
	}

	if err := assetRegister.Validate(); err != nil {
		return nil, goresterr.NewAPIError(goresterr.InvalidFormat, fmt.Sprintf("parse action register input invalid: %s", err.Error()))
	}

	assetID := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var assets []*ipamresource.Asset
		if err := tx.Fill(map[string]interface{}{restdb.IDField: assetID}, &assets); err != nil {
			return err
		} else if len(assets) != 1 {
			return fmt.Errorf("no found asset %s", assetID)
		}

		var subnets []*dhcpresource.Subnet
		if err := tx.Fill(map[string]interface{}{restdb.IDField: assetRegister.SubnetId}, &subnets); err != nil {
			return err
		} else if len(subnets) != 1 {
			return fmt.Errorf("no found subnet %s", assetRegister.SubnetId)
		}

		_, ipnet, _ := net.ParseCIDR(subnets[0].Ipnet)
		if ipnet.Contains(net.ParseIP(assetRegister.Ip)) == false {
			return fmt.Errorf("ip %s not belongs to subnet %s", assetRegister.Ip, subnets[0].Ipnet)
		}

		if err := checkDuplicateIp(tx, assetRegister.ToAsset(assetID)); err != nil {
			return err
		}

		updateField := map[string]interface{}{
			"vlan_id":       assetRegister.VlanId,
			"computer_room": assetRegister.ComputerRoom,
			"computer_rack": assetRegister.ComputerRack,
			"switch_name":   assetRegister.SwitchName,
			"switch_port":   assetRegister.SwitchPort,
		}
		if assetRegister.Isv4 {
			updateField["ipv4s"] = replaceOrAppendIpToIps(ipnet, assets[0].Ipv4s, assetRegister.Ip)
		} else {
			updateField["ipv6s"] = replaceOrAppendIpToIps(ipnet, assets[0].Ipv6s, assetRegister.Ip)
		}

		_, err := tx.Update(TableAsset, updateField, map[string]interface{}{restdb.IDField: assetID})
		return err
	}); err != nil {
		return nil, goresterr.NewAPIError(goresterr.ServerError, fmt.Sprintf("register asset %s failed: %s", assetID, err.Error()))
	}

	return nil, nil
}

func replaceOrAppendIpToIps(ipnet *net.IPNet, ips []string, ip string) []string {
	var retIps []string
	for _, ip_ := range ips {
		if ipnet.Contains(net.ParseIP(ip_)) == false {
			retIps = append(retIps, ip_)
		}
	}

	return append(retIps, ip)
}
