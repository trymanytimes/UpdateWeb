package handler

import (
	"fmt"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/alarm/resource"
	"github.com/linkingthing/ddi-controller/pkg/db"
)

const (
	DefaultCpuUsedRatio     uint64 = 80
	DefaultMemoryUsedRatio  uint64 = 80
	DefaultStorageUsedRatio uint64 = 90
	DefaultQPS              uint64 = 200000
	DefaultLPS              uint64 = 3000
	DefaultSubnetUsedRatio  uint64 = 95
)

type ThresholdHandler struct{}

func NewThresholdHandler() (*ThresholdHandler, error) {
	h := &ThresholdHandler{}
	if err := h.init(); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *ThresholdHandler) init() error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := initThreshold(tx, resource.ThresholdNameCpuUsedRatio, resource.ThresholdLevelCritical,
			resource.ThresholdTypeRatio, DefaultCpuUsedRatio); err != nil {
			return fmt.Errorf("init cpu threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameMemoryUsedRatio, resource.ThresholdLevelCritical,
			resource.ThresholdTypeRatio, DefaultMemoryUsedRatio); err != nil {
			return fmt.Errorf("init memory threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameStorageUsedRatio, resource.ThresholdLevelCritical,
			resource.ThresholdTypeRatio, DefaultStorageUsedRatio); err != nil {
			return fmt.Errorf("init storage threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameHATrigger, resource.ThresholdLevelCritical,
			resource.ThresholdTypeTrigger, 0); err != nil {
			return fmt.Errorf("init ha threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameNodeOffline, resource.ThresholdLevelCritical,
			resource.ThresholdTypeTrigger, 0); err != nil {
			return fmt.Errorf("init node state threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameDNSOffline, resource.ThresholdLevelCritical,
			resource.ThresholdTypeTrigger, 0); err != nil {
			return fmt.Errorf("init dns state threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameDHCPOffline, resource.ThresholdLevelCritical,
			resource.ThresholdTypeTrigger, 0); err != nil {
			return fmt.Errorf("init dhcp state threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameQPS, resource.ThresholdLevelCritical,
			resource.ThresholdTypeValue, DefaultQPS); err != nil {
			return fmt.Errorf("init dhcp state threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameLPS, resource.ThresholdLevelCritical,
			resource.ThresholdTypeValue, DefaultLPS); err != nil {
			return fmt.Errorf("init dhcp state threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameSubnetUsedRatio, resource.ThresholdLevelMajor,
			resource.ThresholdTypeRatio, DefaultSubnetUsedRatio); err != nil {
			return fmt.Errorf("init dhcp state threshold failed: %s", err.Error())
		}
		if err := initThreshold(tx, resource.ThresholdNameIPConflict, resource.ThresholdLevelMajor,
			resource.ThresholdTypeTrigger, 0); err != nil {
			return fmt.Errorf("init ip conflict threshold failed: %s", err.Error())
		}
		return nil
	})
}

func initThreshold(tx restdb.Transaction, name resource.ThresholdName, level resource.ThresholdLevel, typ resource.ThresholdType, value uint64) error {
	sendMail := true
	if level != resource.ThresholdLevelCritical {
		sendMail = false
	}

	id := strings.ToLower(string(name))
	if exists, err := tx.Exists(resource.TableThreshold, map[string]interface{}{restdb.IDField: id}); err != nil {
		return err
	} else if exists == false {
		threshold := &resource.Threshold{
			Name:          name,
			Level:         level,
			ThresholdType: typ,
			Value:         value,
			SendMail:      sendMail,
		}
		threshold.SetID(id)
		if _, err := tx.Insert(threshold); err != nil {
			return err
		}
	}

	return nil
}

func (h *ThresholdHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	threshold := ctx.Resource.(*resource.Threshold)
	threshold.SetID(strings.ToLower(string(threshold.Name)))
	level, err := getThresholdLevelByName(threshold.Name)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("add threshold %s faield: %s", threshold.Name, err.Error()))
	}

	threshold.Level = level
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(threshold)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("add threshold %s faield: %s", threshold.Name, err.Error()))
	}

	return ctx.Resource, nil
}

func getThresholdLevelByName(name resource.ThresholdName) (resource.ThresholdLevel, error) {
	switch name {
	case resource.ThresholdNameCpuUsedRatio, resource.ThresholdNameMemoryUsedRatio, resource.ThresholdNameQPS, resource.ThresholdNameLPS,
		resource.ThresholdNameHATrigger, resource.ThresholdNameNodeOffline, resource.ThresholdNameDNSOffline, resource.ThresholdNameDHCPOffline:
		return resource.ThresholdLevelCritical, nil
	case resource.ThresholdNameSubnetUsedRatio, resource.ThresholdNameStorageUsedRatio:
		return resource.ThresholdLevelMajor, nil
	default:
		return resource.ThresholdLevel(""), fmt.Errorf("unsupported threshold %s", name)
	}
}

func (h *ThresholdHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var thresholds []*resource.Threshold
	if err := db.GetResources(nil, &thresholds); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list threshold faield: %s", err.Error()))
	}

	return thresholds, nil
}

func (h *ThresholdHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	thresholdID := ctx.Resource.(*resource.Threshold).GetID()
	var thresholds []*resource.Threshold
	threshold, err := restdb.GetResourceWithID(db.GetDB(), thresholdID, &thresholds)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get threshold %s faield: %s", thresholdID, err.Error()))
	}

	return threshold.(restresource.Resource), nil
}

func (h *ThresholdHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	threshold := ctx.Resource.(*resource.Threshold)
	if threshold.ThresholdType != resource.ThresholdTypeTrigger && threshold.Value == 0 {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update threshold %s faield with zero value", threshold.GetID()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TableThreshold, map[string]interface{}{
			"value":     threshold.Value,
			"send_mail": threshold.SendMail,
		}, map[string]interface{}{restdb.IDField: threshold.GetID()})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update threshold %s faield: %s", threshold.GetID(), err.Error()))
	}

	return threshold, nil
}

func (h *ThresholdHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	thresholdID := ctx.Resource.(*resource.Threshold).GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(resource.TableThreshold, map[string]interface{}{restdb.IDField: thresholdID})
		return err
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete threshold %s failed: %s", thresholdID, err.Error()))
	}

	return nil
}
