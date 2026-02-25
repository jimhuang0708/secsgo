package secs_message

import (
	"fmt"
	"strings"
)

type ListNode struct {
    Node
}

func (node *ListNode) Clone() (ElementType) {
    nodeValues := make([]interface{}, 0, len(node.NodeValues))
    for _ , value := range node.NodeValues {
        if v, ok := value.(ElementType); ok {
            nodeValues = append(nodeValues, v.Clone())
        } else {
            panic("input argument contains invalid type for ListNode")
        }
     }
     return &ListNode{Node{ NodeValues : nodeValues, NodeType : node.NodeType}}
}


func (node *ListNode) Values() interface{} {
    out := make([]ElementType, len(node.NodeValues))
    for i, v := range node.NodeValues {
        out[i] = v.(ElementType)
    }
    return out
}

func (node *ListNode) Type() string {
    return node.NodeType
}

func (node *ListNode) Code() byte {
    return  0o00
}

func CreateListNode(values ...interface{}) ElementType {
    if  len(values) > MAX_BYTE_SIZE {
        panic("List too long")
    }
    nodeValues := make([]interface{}, 0, len(values))
    for _ , value := range values {
        if v, ok := value.(ElementType); ok {
            nodeValues = append(nodeValues, v)
        } else {
            panic("input argument contains invalid type for ListNode")
        }
    }
    node := &ListNode{Node{ NodeValues : nodeValues, NodeType : "L" }}
    return node
}

func (node *ListNode) Size() int {
    return len(node.NodeValues)

}
func (node *ListNode) DataLength() int {
    return len(node.NodeValues)
}

func (node *ListNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return []byte{}
    }
    for _, item := range node.NodeValues {
        // Call EncodeBytes() of child node recursively
        childResult := item.(ElementType).EncodeBytes()
        if len(childResult) == 0 {
            return []byte{}
        }
        result = append(result, childResult...)
    }
    return result
}

func (node *ListNode) Get(indices ...int) (ElementType, error) {
    itemNode := ElementType(node)
    if len(indices) == 0 {
        return node, nil
    }

    for _, index := range indices {
        if itemNode.Type() != "L" {
            return nil, fmt.Errorf("not list")
        }
        listNode := itemNode.(*ListNode)
        if index < 0 || index >= len(listNode.NodeValues) {
            return nil, fmt.Errorf("index out of bounds error, size : %d", len(listNode.NodeValues))
        }
	itemNode = listNode.NodeValues[index].(ElementType)
    }
    return itemNode, nil
}

func (node *ListNode) ToSml() string {
    return node.stringIndented(0)
}

func (node *ListNode) stringIndented(level int) string {
    indentStr := strings.Repeat("  ", level)
    if node.Size() == 0 {
        return fmt.Sprintf("%v<L[0]>", indentStr)
    }
    var ( sizeDetermined bool  = true
          sb strings.Builder )
    for _, val := range node.NodeValues {
        if v, ok := val.(*ListNode); ok {
            fmt.Fprintln(&sb, v.stringIndented(level+1))
        } else {
            fmt.Fprintf(&sb, "%v  %v\n", indentStr, val.(ElementType).ToSml())
        }
    }
    sizeStr := ""
    if sizeDetermined {
        sizeStr = fmt.Sprintf("[%d]", node.Size())
    }
    return fmt.Sprintf("%v<L%v\n%v%v>", indentStr, sizeStr, sb.String(), indentStr)
}

