package secs_message

import (
	"fmt"
	"strings"
)

type ListNode struct {
    values []ElementType
    symbol string
}

func (node *ListNode) Clone() (ElementType) {
    nodeValues := make([]ElementType, 0, len(node.values))
    for _ , value := range node.values {
        if v, ok := value.(ElementType); ok {
            nodeValues = append(nodeValues, v.Clone())
        } else {
            panic("input argument contains invalid type for ListNode")
        }
     }
     return &ListNode{nodeValues, node.symbol}
}


func (node *ListNode) Values() interface{} {
    return node.values
}

func (node *ListNode) Type() string {
    return node.symbol
}

func (node *ListNode) Code() byte {
    return  0o00
}

func CreateListNode(values ...interface{}) ElementType {
    if  len(values) > MAX_BYTE_SIZE {
        panic("List too long")
    }
    var nodeValues []ElementType = make([]ElementType, 0, len(values))
    for _ , value := range values {
        if v, ok := value.(ElementType); ok {
            nodeValues = append(nodeValues, v)
        } else {
            panic("input argument contains invalid type for ListNode")
        }
    }
    node := &ListNode{nodeValues,  "L"}
    return node
}

func (node *ListNode) Size() int {
    return len(node.values)

}
func (node *ListNode) DataLength() int {
    return len(node.values)
}

func (node *ListNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return []byte{}
    }
    for _, item := range node.values {
        // Call EncodeBytes() of child node recursively
        childResult := item.EncodeBytes()
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
        if index < 0 || index >= len(listNode.values) {
            return nil, fmt.Errorf("index out of bounds error, size : %d", len(listNode.values))
        }
	itemNode = listNode.values[index]
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
    for _, val := range node.values {
        if v, ok := val.(*ListNode); ok {
            fmt.Fprintln(&sb, v.stringIndented(level+1))
        } else {
            fmt.Fprintf(&sb, "%v  %v\n", indentStr, val.ToSml())
        }
    }
    sizeStr := ""
    if sizeDetermined {
        sizeStr = fmt.Sprintf("[%d]", node.Size())
    }
    return fmt.Sprintf("%v<L%v\n%v%v>", indentStr, sizeStr, sb.String(), indentStr)
}

