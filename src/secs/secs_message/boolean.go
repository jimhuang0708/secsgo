package secs_message

import (
	"fmt"
	"strings"
)

type BooleanNode struct {
    values []bool
    symbol string
}

func (node *BooleanNode) Clone() (ElementType) {
    nodeValues := make([]bool,  len(node.values))
    copy(nodeValues,node.values)
    return &BooleanNode{nodeValues, node.symbol }
}

func (node *BooleanNode) Values() interface{} {
    return node.values
}

func (node *BooleanNode) Type() string {
    return node.symbol
}

func (node *BooleanNode) Code() byte {
    return 0o11
}

func CreateBooleanNode(values ...interface{}) ElementType {
    if len(values) > MAX_BYTE_SIZE {
        panic("boolean datalength too long\n")
    }

    var (
        nodeValues    []bool         = make([]bool, 0, len(values))
    )

    for _ , value := range values {
        if v, ok := value.(bool); ok {
            nodeValues = append(nodeValues, v)
	} else {
	    panic("Convert to bool failed")
	}
    }
    node := &BooleanNode{nodeValues,  "BOOLEAN"}
    return node
}

func (node *BooleanNode) Size() int {
    return len(node.values)
}

func (node *BooleanNode) DataLength() int {
    return len(node.values)
}

func (node *BooleanNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return []byte{}
    }

    for _, value := range node.values {
        if value {
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
    for _, value := range node.values {
        if value {
            values = append(values, "T")
        } else {
            values = append(values, "F")
        }
    }
    return fmt.Sprintf("<BOOLEAN[%d] %v>", node.Size(), strings.Join(values, " "))
}

