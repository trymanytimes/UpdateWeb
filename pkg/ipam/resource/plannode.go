package resource

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/ipam/util"
)

const (
	RootParentId = "0"
)

type PlanNode struct {
	resource.ResourceBase       `json:",inline"`

	Layout   string `db:"ownby"`
	PId      string `json:"pid"`
	Name     string `json:"name"`
	Prefix   string `json:"prefix"`
	Sequence int    `json:"sequence"`
	BitWidth int    `json:"bitWidth"`
	Value    int    `json:"value"`
	IPv4     string `json:"ipv4,omitempty"`
	Modified int    `json:"modified"`
}

type NodeTree struct {
	PlanNode   *PlanNode
	ChildTrees []*NodeTree
}

func insert(tx db.Transaction, nodeTree *NodeTree) error  {
	nodeTree.PlanNode.Modified = 0
	if _, err := tx.Insert(nodeTree.PlanNode); err != nil {
		return err
	}

	for _, tree := range nodeTree.ChildTrees {
		if err := insert(tx, tree); err != nil {
			return err
		}
	}
	return nil
}

func deleteTreeDBItem(tx db.Transaction, planNode *PlanNode) error  {
	var subNodeItems []*PlanNode
	if err := tx.Fill(map[string]interface{}{"p_id": planNode.ID}, &subNodeItems); err != nil {
		return err
	}

	for _, tree := range subNodeItems {
		if err := deleteTreeDBItem(tx, tree); err != nil {
			return err
		}
	}
	_, err := tx.Delete(db.ResourceDBType(&PlanNode{}), map[string]interface{}{"id": planNode.ID})
	return err
}

func (nodeTree *NodeTree) UpdateDB(tx db.Transaction) error {
	// step 1: if nodeTree is modified, delete all its children tree item in DB first
	if nodeTree.PlanNode.Modified == 1 {
		if err := deleteTreeDBItem(tx, nodeTree.PlanNode); err != nil {
			return err
		}
		// step 2: insert new trees
		return insert(tx, nodeTree)
	} else {
		// or step 1': update all its children tree
		for _, tree := range nodeTree.ChildTrees {
			if err := tree.UpdateDB(tx); err != nil {
				return err
			}
		}
		return nil
	}
}

func (node *PlanNode) isRootNode() bool {
	return node.PId == RootParentId
}

func (nodeTree *NodeTree) buildChildTree(nodes []*PlanNode) []*PlanNode {
	var leftTrees []*PlanNode
	for i, _ := range nodes {
		if nodes[i].PId == nodeTree.PlanNode.ID {
			nodeTree.ChildTrees = append(nodeTree.ChildTrees, &NodeTree{PlanNode: nodes[i]})
		} else {
			leftTrees = append(leftTrees, nodes[i])
		}
	}
	if len(leftTrees) == 0 {
		return nil
	}

	for i, _ := range nodeTree.ChildTrees {
		leftTrees = nodeTree.ChildTrees[i].buildChildTree(leftTrees)
		if len(leftTrees) == 0 {
			return nil
		}
	}
	return leftTrees
}

func BuildLayoutTree(nodes []*PlanNode) *NodeTree {
	for i, _ := range nodes {
		if nodes[i].isRootNode() {
			headTree := &NodeTree{PlanNode: nodes[i]}
			if len(nodes) > 1 {
				var leftNodes = make([]*PlanNode, len(nodes) - 1)
				copy(leftNodes, nodes[:i])
				copy(leftNodes[i:], nodes[i+1:])
				headTree.buildChildTree(leftNodes)
			}
			return headTree
		}
	}
	return nil
}

func (nodeTree *NodeTree) getPlanNodesBreadthFirst() []*PlanNode {
	var curTrees []*NodeTree
	curTrees = append(curTrees, nodeTree)

	var planNodes []*PlanNode
	var nextTrees []*NodeTree
	for {
		for _, tree := range curTrees {
			planNodes = append(planNodes, tree.PlanNode)
			nextTrees = append(nextTrees, tree.ChildTrees...)
		}
		if len(nextTrees) > 0 {
			curTrees = curTrees[:0]
			curTrees = append(curTrees, nextTrees...)
			nextTrees = nextTrees[:0]
		} else {
			break
		}
	}
	return planNodes
}

func (nodeTree *NodeTree) GetPlanNodesWithHead() []*PlanNode {
	return nodeTree.getPlanNodesBreadthFirst()
}

func (nodeTree *NodeTree) GetPlanNodesWithoutHead() []*PlanNode {
	if nodes := nodeTree.getPlanNodesBreadthFirst(); len(nodes) > 0 {
		return nodes[1:]
	} else {
		return nil
	}
}

func (nodeTree *NodeTree) GetTreeMaxBitWidth() (int, error) {
	if !nodeTree.PlanNode.isRootNode() && nodeTree.PlanNode.BitWidth <= 0 {
		return 0, errors.New("plan node width less than 0: " + strconv.Itoa(nodeTree.PlanNode.BitWidth))
	}

	curWidth := nodeTree.PlanNode.BitWidth
	maxWidth := nodeTree.PlanNode.BitWidth
	for _, tree := range nodeTree.ChildTrees {
		width, err := tree.GetTreeMaxBitWidth()
		if err != nil {
			return 0, err
		}

		if maxWidth < width + curWidth {
			maxWidth = width + curWidth
		}
	}
	return maxWidth, nil
}

func (nodeTree *NodeTree) fillNodeNetPrefix(pIpNet *net.IPNet) error {
	prefixLen, _ := pIpNet.Mask.Size()
	for i, tree := range nodeTree.ChildTrees {
		if i >= (1 << tree.PlanNode.BitWidth) - 1 {
			return errors.New("plan node number exceeds the capacity")
		}

		treePrefix, _ := util.FromIPNet(pIpNet, prefixLen + tree.PlanNode.BitWidth)
		tree.PlanNode.Value = i + 1
		if err := treePrefix.SetSegment(prefixLen, tree.PlanNode.BitWidth, tree.PlanNode.Value); err != nil {
			return err
		}
		treeIpNet := treePrefix.ToIPNet()
		tree.PlanNode.Prefix = treeIpNet.String()
		if err := tree.fillNodeNetPrefix(treeIpNet); err != nil {
			return err
		}
	}
	return nil
}

func (nodeTree *NodeTree) FillNetPrefixValue(ipNet *net.IPNet, maxMaskLen int) error {
	treeBitWidth, err := nodeTree.GetTreeMaxBitWidth()
	if err != nil {
		return err
	}

	maskLen, _ := ipNet.Mask.Size()
	if treeBitWidth + maskLen > maxMaskLen {
		return errors.New("plan node width: " + strconv.Itoa(treeBitWidth) +
			" exceeds total mask length: " + strconv.Itoa(maxMaskLen))
	}
	// head node stores the basic ipv6 prefix defined in plan
	nodeTree.PlanNode.Prefix = ipNet.String()

	gapWidth := maxMaskLen - maskLen - treeBitWidth
	firstPrefix, _ := util.FromIPNet(ipNet, maskLen + gapWidth)
	if gapWidth > 0 {
		if err = firstPrefix.SetSegment(maskLen, gapWidth, 0); err != nil {
			return err
		}
	}
	return nodeTree.fillNodeNetPrefix(firstPrefix.ToIPNet())
}

func (nodeTree *NodeTree) validateTreeBitWidth(expectedWidth int, isStrictValidate bool) bool {
	if len(nodeTree.ChildTrees) > 0 {
		for _, ssTree := range nodeTree.ChildTrees {
			if ret := ssTree.validateTreeBitWidth(expectedWidth - nodeTree.PlanNode.BitWidth, isStrictValidate); !ret {
				return ret
			}
		}
		return true
	} else {
		return (isStrictValidate && nodeTree.PlanNode.BitWidth == expectedWidth) ||
			(!isStrictValidate && nodeTree.PlanNode.BitWidth <= expectedWidth)
	}
}

func (nodeTree *NodeTree) validateIPv4() error {
	planNodes := nodeTree.GetPlanNodesWithoutHead()
	if len(planNodes) == 0 {
		return nil
	}

	allPrefix := make(map[string]struct{})
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
				return err
			}

			prefixStr := ipNet.String()
			if ipv4Prefix != prefixStr {
				return errors.New("invalid ipv4 prefix! suggested: " + prefixStr)
			}

			if _, bExists := allPrefix[prefixStr]; bExists {
				return errors.New("duplicated ipv4 prefix: " + prefixStr)
			}
			allPrefix[prefixStr] = struct{}{}
		}
	}
	return nil
}

func (nodeTree *NodeTree) Validate(expectedWidth int, isStrictValidate bool) error {
	if nodeTree.validateTreeBitWidth(expectedWidth, isStrictValidate) {
		return nodeTree.validateIPv4()
	} else {
		return errors.New("subnet width doesn't match the pre-defined mask length")
	}
}

