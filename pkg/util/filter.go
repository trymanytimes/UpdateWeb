package util

import (
	"fmt"
	"net"
	"strings"

	restresource "github.com/zdnscloud/gorest/resource"
)

func GenStrConditionsFromFilters(filters []restresource.Filter, filterNames ...string) map[string]interface{} {
	if len(filters) == 0 {
		return nil
	}

	conditions := make(map[string]interface{})
	for _, filterName := range filterNames {
		if value, ok := GetFilterValueWithEqModifierFromFilters(filterName, filters); ok {
			conditions[filterName] = value
		}
	}

	return conditions
}

func GetFilterValueWithEqModifierFromFilters(filterName string, filters []restresource.Filter) (string, bool) {
	for _, filter := range filters {
		if filter.Name == filterName && filter.Modifier == restresource.Eq && len(filter.Values) == 1 {
			return filter.Values[0], true
		}
	}

	return "", false
}

func GetFilterValueWithEqModifierFromFilter(filter restresource.Filter) (string, bool) {
	if filter.Modifier == restresource.Eq && len(filter.Values) == 1 {
		return filter.Values[0], true
	}

	return "", false
}

func PrefixContainsSubnetOrIP(prefix, subnetOrIp string) (bool, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return false, fmt.Errorf("parser prefix:%s failed:%s", prefix, err.Error())
	}

	if strings.Contains(subnetOrIp, "/") {
		if ip, subIpNet, err := net.ParseCIDR(subnetOrIp); err != nil {
			return false, fmt.Errorf("parser subnet:%s failed:%s", subnetOrIp, err.Error())
		} else {
			subSize, _ := subIpNet.Mask.Size()
			size, _ := ipNet.Mask.Size()
			return ipNet.Contains(ip) && (subSize == 0 || size <= subSize), nil
		}
	} else {
		if ip := net.ParseIP(subnetOrIp); ip == nil {
			return false, fmt.Errorf("bad ip:%s", subnetOrIp)
		} else {
			return ipNet.Contains(ip), nil
		}
	}
}
