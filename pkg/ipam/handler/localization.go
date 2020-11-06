package handler

import (
	"strconv"
	"time"

	"github.com/linkingthing/ddi-controller/pkg/ipam/resource"
	"github.com/linkingthing/ddi-controller/pkg/util"
)

var Localization = map[string]string{
	string(resource.IPStateActive):   "活跃地址",
	string(resource.IPStateInactive): "不活跃地址",
	string(resource.IPStateConflict): "冲突地址",
	string(resource.IPStateZombie):   "僵尸地址",

	string(resource.IPTypeAssigned):    "已分配地址",
	string(resource.IPTypeUnassigned):  "未分配地址",
	string(resource.IPTypeReservation): "固定地址",
	string(resource.IPTypeStatic):      "静态地址",
	string(resource.IPTypeUnmanagered): "未使用地址",
}

func localizationNicToStrSlice(nic *resource.NetworkInterface) []string {
	slice := []string{nic.Ip, nic.Mac, Localization[string(nic.IpType)], Localization[string(nic.IpState)]}
	if nic.ValidLifetime != 0 {
		slice = append(slice, strconv.Itoa(int(nic.ValidLifetime)))
	}

	if expire := time.Time(nic.Expire); expire.IsZero() == false {
		slice = append(slice, expire.Format(util.TimeFormat))
	}

	return slice
}
