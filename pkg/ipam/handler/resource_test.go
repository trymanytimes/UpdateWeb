package handler

import (
	"fmt"
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/config"
	"github.com/linkingthing/ddi-controller/pkg/db"
	ipamresource "github.com/linkingthing/ddi-controller/pkg/ipam/resource"
)


func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&ipamresource.Plan{},
		&ipamresource.Layout{},
		&ipamresource.PlanNode{},
	}
}

func TestAddressPlanValidate(t *testing.T) {
	cases := []struct {
		prefix  string
		maskLen int
		isValid bool
	}{
		{"2001:503:ba3e::/48", 64, true},
		{"2001:503:ba3e::/48", 32, false},
		{"2001:503:ba3e::/64", 64, false},
		{"2001:503:ba3e::/65", 64, false},
		{"1.0.0.0/8", 16, true},
		{"1.0.0.0/18", 16, false},
		{"1.0.0.0/8", 25, false},
	}

	for _, c := range cases {
		p := &ipamresource.Plan{
			Prefix:  c.prefix,
			MaskLen: c.maskLen,
		}
		err := p.Validate()
		if c.isValid {
			ut.Assert(t, err == nil, "")
		} else {
			ut.Assert(t, err != nil, "")
		}
	}
}

func TestLayoutValidate(t *testing.T) {
	p := &ipamresource.Plan{
		Prefix:  "1999:1201::/32",
		MaskLen: 64,
	}
	p.ID = "1201"
	p.Description = "plan.1201"
	ut.Assert(t, p.Validate() == nil, "")

	l := &ipamresource.Layout{
		Plan:     p.ID,
		Name:     "layout1",
		AutoFill: true,
		FirstFinished: true,
		PlanNodes: []*ipamresource.PlanNode{
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "30"},
				PId:          "0",
				Name:         "lx",
				Sequence:     0,
				BitWidth:     0,
				Value:        0,
				IPv4:         "",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "31"},
				PId:          "30",
				Name:         "group1",
				Sequence:     1,
				BitWidth:     16,
				Value:        1,
				IPv4:         "10.1.0.0/16,10.10.2.0/24",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "32"},
				PId:          "30",
				Name:         "group2",
				Sequence:     2,
				BitWidth:     16,
				Value:        2,
				IPv4:         "10.2.0.0/16",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "33"},
				PId:          "31",
				Name:         "group11",
				Sequence:     1,
				BitWidth:     16,
				Value:        3,
				IPv4:         "10.1.1.0/24,10.12.0.0/20",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "34"},
				PId:          "31",
				Name:         "group12",
				Sequence:     2,
				BitWidth:     16,
				Value:        4,
				IPv4:         "10.1.2.0/24,10.9.3.0/24",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "35"},
				PId:          "32",
				Name:         "group21",
				Sequence:     1,
				BitWidth:     16,
				Value:        5,
				IPv4:         "10.2.1.0/24,10.12.5.0/24",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "36"},
				PId:          "32",
				Name:         "group22",
				Sequence:     2,
				BitWidth:     16,
				Value:        6,
				IPv4:         "10.2.2.0/24,10.10.0.0/16,10.12.0.0/14",
				Modified:     1,
			},
		},
	}

	l2 := &ipamresource.Layout{
		Plan:     p.ID,
		Name:     "layout1new",
		AutoFill: true,
		FirstFinished: true,
		PlanNodes: []*ipamresource.PlanNode{
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "30"},
				PId:          "0",
				Name:         "lx",
				Sequence:     0,
				BitWidth:     0,
				Value:        0,
				IPv4:         "10.0.0.0/8",
				Modified:     0,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "31"},
				PId:          "30",
				Name:         "group1",
				Sequence:     1,
				BitWidth:     16,
				Value:        1,
				IPv4:         "10.1.0.0/16",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "32"},
				PId:          "30",
				Name:         "group2",
				Sequence:     2,
				BitWidth:     16,
				Value:        2,
				IPv4:         "10.2.0.0/16",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "33"},
				PId:          "31",
				Name:         "group11",
				Sequence:     1,
				BitWidth:     16,
				Value:        3,
				IPv4:         "10.1.4.0/24",
				Modified:     0,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "37"},
				PId:          "31",
				Name:         "group12new",
				Sequence:     2,
				BitWidth:     16,
				Value:        4,
				IPv4:         "10.1.2.0/24",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "38"},
				PId:          "31",
				Name:         "group12new",
				Sequence:     2,
				BitWidth:     16,
				Value:        4,
				IPv4:         "10.3.2.0/24",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "50"},
				PId:          "32",
				Name:         "group21",
				Sequence:     1,
				BitWidth:     16,
				Value:        5,
				IPv4:         "10.2.1.0/24",
				Modified:     1,
			},
			&ipamresource.PlanNode{
				ResourceBase: restresource.ResourceBase{ID: "51"},
				PId:          "32",
				Name:         "group22",
				Sequence:     2,
				BitWidth:     16,
				Value:        6,
				IPv4:         "10.2.3.0/24",
				Modified:     1,
			},
		},
	}

	err := l.Validate(p, false)
	ut.Assert(t, err == nil, "")

	conf, err := config.LoadConfig("../../../etc/ddi-controller.conf")
	ut.Assert(t, err == nil, "load config file failed")

	db.RegisterResources(PersistentResources()...)
	err = db.Init(conf)
	ut.Assert(t, err == nil, "init db failed")

	err = ipamresource.DeletePlanFromDB(p.ID)
	ut.Assert(t, err == nil, "")

	err = p.SavePlanToDB()
	ut.Assert(t, err == nil, "")

	err = l.Delete()
	ut.Assert(t, err == nil, "")

	err = l.SaveLayoutToDB()
	ut.Assert(t, err == nil, "")

	// for layout update
	l2.ID = l.ID
	err = l2.Validate(p, false)
	ut.Assert(t, err == nil, "")

	err = l2.UpdateToDB()
	ut.Assert(t, err == nil, "")

	tree1, err := l2.GetNodeTree()
	ut.Assert(t, err == nil, "")

	// reload case
	l12, err := ipamresource.LoadLayoutFromDB(l.ID)
	ut.Assert(t, err == nil, "")

	tree2, err := l12.GetNodeTree()
	ut.Assert(t, err == nil, "")

	t1Nodes := tree1.GetPlanNodesWithHead()
	t2Nodes := tree2.GetPlanNodesWithHead()
	bEqual := len(t1Nodes) == len(t2Nodes)
	ut.Assert(t, bEqual, "")

	netNodesV6, err := ipamresource.GetNetNodesList(p, l12, "netv6")
	ut.Assert(t, err == nil, "")
	for _, netNode := range netNodesV6 {
		for i := 0; i < len(netNode.NetItems); i++ {
			fmt.Printf("netNodeV6: %d, Name: %v, Prefix:%v, Tags:%v, Level:%v, Usage:%v\n", i,
				netNode.NetItems[i].Name, netNode.NetItems[i].Prefix, netNode.NetItems[i].Tags, netNode.NetItems[i].Level, netNode.NetItems[i].Usage)
		}
	}

	netNodesV4, err := ipamresource.GetNetNodesList(p, l12, "netv4")
	ut.Assert(t, err == nil, "")
	for _, netNode := range netNodesV4 {
		for i := 0; i < len(netNode.NetItems); i++ {
			fmt.Printf("netNodeV4: %d, Name: %v, Prefix:%v, Tags:%v, Level:%v, Usage:%v\n", i,
				netNode.NetItems[i].Name, netNode.NetItems[i].Prefix, netNode.NetItems[i].Tags, netNode.NetItems[i].Level, netNode.NetItems[i].Usage)
		}
	}
}

