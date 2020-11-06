package resource

import (
	"net"

	"github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"

	ipamdb "github.com/linkingthing/ddi-controller/pkg/db"
)

type Layout struct {
	resource.ResourceBase `json:",inline"`

	Plan          string      `json:"-" db:"ownby"`
	Name          string      `json:"name"`
	AutoFill      bool        `json:"autofill"`
	FirstFinished bool        `json:"firstfinished"`
	PlanNodes     []*PlanNode `json:"nodes" db:"-"`
	treeHead      *NodeTree   `json:"-" db:"-"`
}

func (layout Layout) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Plan{}}
}

func (layout *Layout) Validate(p *Plan, isStrictValidate bool) error {
	_, network, _ := net.ParseCIDR(p.Prefix)
	baseMaskLen, _ := network.Mask.Size()

	treeHead, err := layout.GetNodeTree()
	if err != nil {
		return err
	}

	if treeHead != nil {
		return treeHead.Validate(p.MaskLen - baseMaskLen, isStrictValidate)
	}
	return nil
}

func (layout *Layout) autoFillNetValue(prefix string, maskLen int) error {
	if !layout.AutoFill {
		return nil
	}

	_, ipPrefix, err := net.ParseCIDR(prefix)
	if err != nil {
		return err
	}

	if treeHead, err := layout.GetNodeTree(); err != nil {
		return err
	} else {
		if treeHead != nil {
			return treeHead.FillNetPrefixValue(ipPrefix, maskLen)
		} else {
			return nil
		}
	}
}

func (layout *Layout) SaveLayoutToDB() error {
	var plans []*Plan
	plan, err := db.GetResourceWithID(ipamdb.GetDB(), layout.Plan, &plans)
	if err != nil {
		return err
	}

	if err = layout.Validate(plan.(*Plan), false); err != nil {
		return err
	}

	if err = layout.autoFillNetValue(plan.(*Plan).Prefix, plan.(*Plan).MaskLen); err != nil {
		return err
	}

	return db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		if _, err = tx.Insert(layout); err != nil {
			return err
		}

		for i := 0; i < len(layout.PlanNodes); i++ {
			layout.PlanNodes[i].Modified = 0
			layout.PlanNodes[i].Layout = layout.ID
			if _, err = tx.Insert(layout.PlanNodes[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func (layout *Layout) UpdateToDB() error {
	// todo: check the timestamp to compare the data version

	// set plan nodes layout id
	for i := 0; i < len(layout.PlanNodes); i++ {
		layout.PlanNodes[i].Layout = layout.ID
	}

	var plans []*Plan
	plan, err := db.GetResourceWithID(ipamdb.GetDB(), layout.Plan, &plans)
	if err != nil {
		return err
	}
	if err = layout.Validate(plan.(*Plan), false); err != nil {
		return err
	}

	// auto fill the values
	if err = layout.autoFillNetValue(plan.(*Plan).Prefix, plan.(*Plan).MaskLen); err != nil {
		return err
	}

	return db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		if _, err := tx.Update(db.ResourceDBType(&Layout{}),
			map[string]interface{}{"name": layout.Name, "first_finished": layout.FirstFinished},
			map[string]interface{}{"id": layout.ID}); err != nil {
			return err
		}
		treeHead, err := layout.GetNodeTree()
		if err != nil {
			return err
		}

		if treeHead != nil {
			return treeHead.UpdateDB(tx)
		} else {
			return nil
		}
	})
}

func LoadLayoutFromDB(layoutId string) (*Layout, error) {
	var layouts []*Layout
	if layout_, err := db.GetResourceWithID(ipamdb.GetDB(), layoutId, &layouts); err != nil {
		return nil, err
	} else {
		_, err := layout_.(*Layout).GetNodeTree()
		return layout_.(*Layout), err
	}
}

func (layout *Layout) GetNodeTree() (*NodeTree, error) {
	if layout.treeHead != nil {
		return layout.treeHead, nil
	}

	var nodes []*PlanNode
	if len(layout.PlanNodes) == 0 {
		if layout.ID != "" {
			if err := db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
				return tx.Fill(map[string]interface{}{"layout": layout.ID, "orderby": "sequence"}, &nodes)
			}); err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	} else {
		nodes = layout.PlanNodes
	}

	if len(nodes) > 0 {
		layout.treeHead = BuildLayoutTree(nodes)
		if layout.treeHead != nil {
			layout.PlanNodes = layout.treeHead.GetPlanNodesWithHead()
		}
		return layout.treeHead, nil
	} else {
		return nil, nil
	}
}

func DeleteLayoutFromDB(layoutId string) error {
	return db.WithTx(ipamdb.GetDB(), func(tx db.Transaction) error {
		_, err := tx.Delete(db.ResourceDBType(&Layout{}), map[string]interface{}{"id": layoutId})
		return err
	})
}

func (layout *Layout) Delete() error {
	return DeleteLayoutFromDB(layout.ID)
}