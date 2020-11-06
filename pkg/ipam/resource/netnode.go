package resource

import (
	"encoding/binary"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/zdnscloud/gorest/resource"
)

const (
	NetType = "nettype"
	NetType_V4 = "netv4"
	NetType_V6 = "netv6"

	NetItem_Root_Level = "0A"

	IpV4_Max_Len = 32
)

type NetNode struct {
	resource.ResourceBase `json:",inline"`

	NetItems     []*NetItem `json:"netitems"`
}

type NetItem struct {
	IpNet  *net.IPNet `json:"-"`
	Name   string     `json:"name"`
	Prefix string     `json:"prefix"`
	Tags   string     `json:"tags"`
	Level  string     `json:"level"`
	Usage  string     `json:"usage"`
	child  []*NetItem `json:"-"`
}


func getNetItem(nodeTree *NodeTree, pItem *NetItem, maxMaskLen int) ([]*NetItem, error) {
	if nodeTree == nil || pItem == nil {
		return nil, nil
	}

	var netItems []*NetItem
	netItem := NetItem{
		IpNet:  nil,
		Name:   nodeTree.PlanNode.Name,
		Prefix: nodeTree.PlanNode.Prefix,
		Tags: nodeTree.PlanNode.Name,
		Level:  "",
		Usage:  "",
	}
	if !nodeTree.PlanNode.isRootNode() {
		if nodeTree.PlanNode.Prefix != "" {
			_, treeIpNet, err := net.ParseCIDR(nodeTree.PlanNode.Prefix)
			if err != nil {
				return nil, err
			}

			treeMaskLen, _ := treeIpNet.Mask.Size()
			validlen := maxMaskLen - treeMaskLen
			if validlen > 0 {
				if len(nodeTree.ChildTrees) > 0 {
					var count int64
					var calculated = true
					for i := 0; i < len(nodeTree.ChildTrees); i++ {
						if nodeTree.ChildTrees[i].PlanNode.Prefix == "" {
							calculated = false
							break
						}
						_, subIpNet, err := net.ParseCIDR(nodeTree.ChildTrees[i].PlanNode.Prefix)
						if err != nil {
							return nil, err
						}
						subwidth, _ := subIpNet.Mask.Size()
						sublen := maxMaskLen - subwidth
						count += 1 << sublen
					}
					if calculated {
						usage := count * 100 / (1 << validlen)
						netItem.Usage = strconv.FormatInt(int64(usage), 10) + "%"
					}
				} else {
					netItem.Usage = "0%"
				}
			}
		}

		if pItem.Level == "" {
			netItem.Level = strconv.Itoa(nodeTree.PlanNode.Sequence)
		} else {
			netItem.Level = pItem.Level + "." + strconv.Itoa(nodeTree.PlanNode.Sequence)
		}

		//update tags
		netItem.Tags = pItem.Tags + "," + nodeTree.PlanNode.Name
		netItems = append(netItems, &netItem)
	}

	for _, tree := range nodeTree.ChildTrees {
		childItems, err := getNetItem(tree, &netItem, maxMaskLen)
		if err != nil {
			return nil, err
		}

		if len(childItems) > 0 {
			netItems = append(netItems, childItems...)
		}
	}
	return netItems, nil
}

func (headItem *NetItem) getChildNetItem() []*NetItem {
	var netItems []*NetItem
	for i := 0; i < len(headItem.child); i++ {
		netItems = append(netItems, headItem.child[i])
		childItems := headItem.child[i].getChildNetItem()
		if len(childItems) > 0 {
			netItems = append(netItems, childItems...)
		}
	}
	return netItems
}

func (headItem *NetItem) calcV4Usage() {
	if headItem.Level != NetItem_Root_Level {
		w, _ := headItem.IpNet.Mask.Size()
		sublen := IpV4_Max_Len - w

		var count int
		for i := 0; i < len(headItem.child); i++ {
			wi, _ := headItem.child[i].IpNet.Mask.Size()
			count += 1 << (IpV4_Max_Len - wi)
		}

		usage := count * 100 / (1 << sublen)
		headItem.Usage = strconv.Itoa(usage) + "%"
	}
	for i := 0; i < len(headItem.child); i++ {
		headItem.child[i].calcV4Usage()
	}
}

func (headItem *NetItem) updateSeg(pLevel string, tag string) {
	level := ""
	if pLevel != "" {
		headItem.Level = pLevel
		level = pLevel + "."
	}
	if tag != "" {
		headItem.Tags = tag + "," + headItem.Name
	} else {
		headItem.Tags = headItem.Name
	}

	for i := 0; i < len(headItem.child); i++ {
		headItem.child[i].updateSeg(level + strconv.Itoa(i+1), headItem.Tags)
	}
}


func (headItem *NetItem) removeChildNetItemV4(netItem *NetItem) bool {
	len := len(headItem.child)
	for i := 0; i < len; i++ {
		if headItem.child[i].Prefix == netItem.Prefix {
			copy(headItem.child[i:], headItem.child[i+1:])
			headItem.child[len - 1] = nil
			headItem.child = headItem.child[:len - 1]
			return true
		}
	}
	return false
}

func (headItem *NetItem) insertNetItemV4(netItem *NetItem) bool {
	if netItem == nil {
		return false
	}
	// calculate the new item mask length first
	newItemMaskLen, _ := netItem.IpNet.Mask.Size()

	// insert into child tree
	for i := 0; i < len(headItem.child); i++ {
		iItemMaskLen, _ := headItem.child[i].IpNet.Mask.Size()
		// There is a bug of IpNet.Contains()
		// It will return true for checking 10.12.0.0/20 Contains(10.12.0.0), even the latter is 10.12.0.0/14.
		// so, add mask checking
		if headItem.child[i].IpNet.Contains(netItem.IpNet.IP) && iItemMaskLen < newItemMaskLen {
			return headItem.child[i].insertNetItemV4(netItem)
		}
	}

	// insert as the new child and set it as the father of all previous children
	needInsert := false
	count := len(headItem.child)
	for i := 0; i < count; {
		iItemMaskLen, _ := headItem.child[i].IpNet.Mask.Size()
		if netItem.IpNet.Contains(headItem.child[i].IpNet.IP) && newItemMaskLen < iItemMaskLen {
			netItem.insertNetItemV4(headItem.child[i])
			needInsert = true
			if headItem.removeChildNetItemV4(headItem.child[i]) {
				count -= 1
			} else {
				i++
			}
		} else {
			i++
		}
	}
	if needInsert {
		headItem.insertNetItemV4(netItem)
		return true
	}

	// append as a new child
	isAppended := false
	for i := 0; i < len(headItem.child); i++ {
		newIPVal := binary.BigEndian.Uint32(netItem.IpNet.IP)
		curIPVal := binary.BigEndian.Uint32(headItem.child[i].IpNet.IP)
		if newIPVal < curIPVal {
			headItem.child = append(headItem.child[:i], append([]*NetItem{netItem}, headItem.child[i:]...)...)
			isAppended = true
			break
		}
	}
	if !isAppended {
		headItem.child = append(headItem.child, netItem)
	}
	return true
}

func getNetItemV4(nodeTree *NodeTree) ([]*NetItem, error) {
	planNodes := nodeTree.GetPlanNodesWithoutHead()
	if len(planNodes) == 0 {
		return nil, nil
	}

	headItem := NetItem{
		IpNet:  nil,
		Name:   nodeTree.PlanNode.Name,
		Prefix: "",
		Tags:   nodeTree.PlanNode.Name,
		Level:  NetItem_Root_Level,
		Usage:  "",
	}

	unSortedItems := make(map[string]*NetItem)
	// split the plan nodes to items
	for i := 0; i < len(planNodes); i++ {
		if planNodes[i].IPv4 == "" {
			continue
		}
		ipv4PrefixList := strings.Split(planNodes[i].IPv4, ",")
		for _, ipv4Prefix := range ipv4PrefixList {
			if ipv4Prefix == "" {
				continue
			}
			_, ipNet, err := net.ParseCIDR(ipv4Prefix)
			if err != nil {
				return nil, err
			}

			prefixStr := ipNet.String()
			if _, bExists := unSortedItems[prefixStr]; bExists {
				return nil, errors.New("duplicated plan node prefix: " + prefixStr)
			}
			netItem := NetItem{
				IpNet:  ipNet,
				Name:   planNodes[i].Name,
				Prefix: prefixStr,
				Tags:  "",
				Level:  "",
				Usage:  "",
			}
			unSortedItems[prefixStr] = &netItem
		}
	}

	// insert items
	for  key, _ := range unSortedItems {
		headItem.insertNetItemV4(unSortedItems[key])
	}

	// update segments: level and tags
	headItem.updateSeg("", "")

	// calculate the usage ratio
	headItem.calcV4Usage()

	return headItem.getChildNetItem(), nil
}

func GetNetNodesList(p *Plan, l *Layout, netType string) ([]*NetNode, error) {
	if netType != NetType_V4 && netType != NetType_V6 {
		return nil, errors.New("net type unknown!")
	}

	var netNodes []*NetNode
	var netNode NetNode
	nodeTree, err := l.GetNodeTree()
	if err != nil {
		return nil, err
	}
	if nodeTree == nil {
		return nil, nil
	}

	if netType == NetType_V6 {
		_, ipPrefix, err := net.ParseCIDR(p.Prefix)
		if err != nil {
			return nil, err
		}
		preItem := NetItem{
			IpNet: ipPrefix,
			Name: "",
			Prefix: "",
			Tags: "",
			Level: "",
			Usage: "",
		}
		netNode.NetItems, err = getNetItem(nodeTree, &preItem, p.MaskLen)
		if err != nil {
			return nil, err
		}
	} else {
		netNode.NetItems, err = getNetItemV4(nodeTree)
		if err != nil {
			return nil, err
		}
	}
	netNodes = append(netNodes, &netNode)
	return netNodes, nil
}

func (s NetNode) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Layout{}}
}



