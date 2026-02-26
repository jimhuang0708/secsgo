package secs_message

import (
	"fmt"
	"strings"
)

type BooleanNode struct {
    Node
}

func (node *BooleanNode) Clone() (ElementType) {
    nodeValues := make([]interface{},  len(node.NodeValues))
    copy(nodeValues,node.NodeValues)
    return &BooleanNode{Node{ NodeValues : nodeValues, NodeType : node.NodeType} }
}

func (node *BooleanNode) Values() []any {
    out := make([]any, len(node.NodeValues))
    for i, v := range node.NodeValues {
        out[i] = v
    }
    return out
}

func (node *BooleanNode) Type() string {
    return node.NodeType
}

func (node *BooleanNode) Code() byte {
    return 0o11
}

func CreateBooleanNode(values ...interface{}) ElementType {
    if len(values) > MAX_BYTE_SIZE {
        panic("boolean datalength too long\n")
    }

    nodeValues := make([]interface{}, 0, len(values))
    for _ , value := range values {
        if v, ok := value.(bool); ok {
            nodeValues = append(nodeValues, v)
	} else {
	    panic("Convert to bool failed")
	}
    }
    node := &BooleanNode{Node{ NodeValues : nodeValues, NodeType : "BOOLEAN"}}
    return node
}

func (node *BooleanNode) Size() int {
    return len(node.NodeValues)
}

func (node *BooleanNode) DataLength() int {
    return len(node.NodeValues)
}

func (node *BooleanNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return []byte{}
    }

    for _, value := range node.NodeValues {
        if value.(bool) {
            result = append(result, 1)
        } else {
            result = append(result, 0)
        }
    }
    return result
}

func (node *BooleanNode) ToSml() string {
    if node.Size() == 0 {
        return "<BOOLEAN[0]>"
    }
    values := make([]string, 0, node.Size())
    for _, value := range node.NodeValues {
        if value.(bool) {
            values = append(values, "T")
        } else {
            values = append(values, "F")
        }
    }
    return fmt.Sprintf("<BOOLEAN[%d] %v>", node.Size(), strings.Join(values, " "))
}

