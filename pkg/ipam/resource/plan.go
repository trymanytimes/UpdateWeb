package resource

import (
	"errors"
	"fmt"
	"net"

	ipamdb "github.com/linkingthing/ddi-controller/pkg/db"
	"github.com/zdnscloud/gorest/db"

	"github.com/zdnscloud/gorest/resource"
)

type Plan struct {
	resource.ResourceBase `json:",inline"`

	Prefix      string `json:"prefix" db:"uk"`
	MaskLen     int    `json:"maskLen"`
	Description string `json:"description"`
}

func (plan *Plan) SavePlanToDB() error {
	if err := plan.Validate(); err != nil {
		return err
	}
	return db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		_, err := tx.Insert(plan)
		return err
	})
}

func (p *Plan) Validate() error {
	_, ipnet, err := net.ParseCIDR(p.Prefix)
	if err != nil {
		return err
	}

	ones, total := ipnet.Mask.Size()
	if ones >= p.MaskLen {
		return fmt.Errorf("no bit is left to plan subnet")
	}

	if total == 128 {
		if p.MaskLen > 64 {
			return fmt.Errorf("ipv6 subnet mask should smaller than 64")
		}

		if ones > 64 {
			return fmt.Errorf("ipv6 prefix mask should smaller than 64")
		}
	} else {
		if p.MaskLen > 24 {
			return fmt.Errorf("ipv4 subnet mask should smaller than 24")
		}

		if ones > 24 {
			return fmt.Errorf("ipv4 prefix mask should smaller than 24")
		}
	}

	validPrefix := ipnet.String()
	if p.Prefix != validPrefix {
		return fmt.Errorf("invalid prefix! suggested: %s", validPrefix)
	}
	return nil
}

func DeletePlanFromDB(planId string) error {
	return db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		_, err := tx.Delete(db.ResourceDBType(&Plan{}), map[string]interface{}{"id": planId})
		return err
	})
}

func (plan *Plan) UpdatePlanToDB() error {
	var plans []*Plan
	plan_, err := db.GetResourceWithID(ipamdb.GetDB(), plan.ID, &plans)
	if err != nil {
		return err
	}

	if plan_ == nil {
		return errors.New("plan not found!")
	}

	if plan_.(*Plan).Description == plan.Description {
		return nil
	}

	return db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		_, err := tx.Update(db.ResourceDBType(&Plan{}), map[string]interface{}{"description": plan.Description}, map[string]interface{}{"id": plan.ID})
		return err
	})
}
