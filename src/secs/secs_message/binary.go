package secs_message

import (
	"fmt"
	"strconv"
	"strings"
)

type BinaryNode struct {
    Node
}

func (node *BinaryNode) Clone() (ElementType) {
    nodeValues := make([]interface{},  len(node.NodeValues))
    copy(nodeValues,node.NodeValues)
    return &BinaryNode{ Node{ NodeValues : nodeValues , NodeType :node.NodeType}}
}

func (node *BinaryNode) Values() []any {
    out := make([]any, len(node.NodeValues))
    for i, v := range node.NodeValues {
        out[i] = v
    }
    return out
}

func (node *BinaryNode) Type() string {
    return node.NodeType
}

func (node *BinaryNode) Code() byte {
    return 0o10
}

func CreateBinaryNode(values ...byte) ElementType {
    if len(values) > MAX_BYTE_SIZE {
        panic("datalength too long\n")
    }
    nodeValues :=  make([]interface{}, 0, len(values))
    for _ , value := range values {
        nodeValues = append(nodeValues,value)
    }
    node := &BinaryNode{ Node{ NodeValues :nodeValues,  NodeType : "B"}}
    return node
}

func (node *BinaryNode) Size() int {
    return len(node.NodeValues)
}

func (node *BinaryNode) DataLength() int {
    return len(node.NodeValues)
}

func (node *BinaryNode) EncodeBytes() []byte {
    result, err := buildHeader( node.Code() , node.DataLength())
    if err != nil {
        return []byte{}
    }
    for _, value := range node.NodeValues {
        result = append(result, value.(byte))
    }
    return result
}

func (node *BinaryNode) ToSml() string {
    if node.Size() == 0 {
        return "<B[0]>"
    }
    values := make([]string, 0, node.Size())
    for _, value := range node.NodeValues {
        str := "0b" + strconv.FormatInt(int64(value.(byte)), 2)
        values = append(values, str)
    }
    return fmt.Sprintf("<B[%d] %v>", node.Size(), strings.Join(values, " "))
}

