package secs_message

import (
	"fmt"
	"strconv"
	"strings"
)

type BinaryNode struct {
    values    []byte
    symbol string
}

func (node *BinaryNode) Clone() (ElementType) {
    nodeValues := make([]byte,  len(node.values))
    copy(nodeValues,node.values)
    return &BinaryNode{ nodeValues , node.symbol}
}

func (node *BinaryNode) Values() interface{} {
    return node.values
}

func (node *BinaryNode) Type() string {
    return node.symbol
}

func (node *BinaryNode) Code() byte {
    return 0o10
}

func CreateBinaryNode(values ...interface{}) ElementType {
    if len(values) > MAX_BYTE_SIZE {
        panic("datalength too long\n")
    }
    var nodeValues []byte =  make([]byte, 0, len(values))
    for _ , value := range values {
        if v, ok := value.(byte); ok {
            nodeValues = append(nodeValues, byte(v))
        } else {
            panic("Conver to byte failed")
        }
    }
    node := &BinaryNode{nodeValues,  "B"}
    return node
}

func (node *BinaryNode) Size() int {
    return len(node.values)
}

func (node *BinaryNode) DataLength() int {
    return len(node.values)
}

func (node *BinaryNode) EncodeBytes() []byte {
    result, err := buildHeader( node.Code() , node.DataLength())
    if err != nil {
        return []byte{}
    }
    for _, value := range node.values {
        result = append(result, byte(value))
    }
    return result
}

func (node *BinaryNode) ToSml() string {
    if node.Size() == 0 {
        return "<B[0]>"
    }
    values := make([]string, 0, node.Size())
    for _, value := range node.values {
        str := "0b" + strconv.FormatInt(int64(value), 2)
        values = append(values, str)
    }
    return fmt.Sprintf("<B[%d] %v>", node.Size(), strings.Join(values, " "))
}

