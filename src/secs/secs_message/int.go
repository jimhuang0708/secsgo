package secs_message

import (
    "reflect"
    "fmt"
    "math"
    "strconv"
    "strings"
    "encoding/binary"
)

type IntNode struct {
    byteSize  int
    values    []int64
    symbol string
}

func (node *IntNode) Clone() (ElementType){
    nodeValues := make([]int64,  len(node.values))
    copy(nodeValues,node.values)
    return &IntNode{node.byteSize, nodeValues, node.symbol}
}

func CreateIntNode(byteSize int, values ...interface{}) ElementType {
    if byteSize*len(values) > MAX_BYTE_SIZE {
        panic("int datalength too long")
    }

    nodeValues := make([]int64, 0, len(values))

    for _, v := range values {
        iv, ok := convertToInt64(v)
        if !ok {
            panic("input argument contains invalid type for IntNode")
        }
        nodeValues = append(nodeValues, iv)
    }

    node := &IntNode{
        byteSize: byteSize,
        values:   nodeValues,
        symbol: fmt.Sprintf("I%d", byteSize),
    }
    return node
}


func convertToInt64(v interface{}) (int64, bool) {
    switch value := v.(type) {
    case int:
        return int64(value), true
    case int8, int16, int32, int64:
        return reflect.ValueOf(value).Int(), true
    case uint, uint8, uint16, uint32:
        return int64(reflect.ValueOf(value).Uint()), true
    case uint64:
        if value > math.MaxInt64 {
            panic("value overflow")
        }
        return int64(value), true
    case float32:
        return int64(value), true
    case float64:
        return int64(value), true
    default:
        return 0, false
    }
}

func (node *IntNode) Values() interface{} {
    return node.values
}

func (node *IntNode) Type() string {
    return node.symbol
}

func (node *IntNode) Code() byte{
    if(node.symbol == "I1"){
        return 0o31
    }else if(node.symbol == "I2"){
        return 0o32
    }else if(node.symbol == "I4"){
        return 0o34
    }else if(node.symbol == "I8"){
        return 0o30
    }else{
        fmt.Printf("Error unknown Int symbol %s\n",node.symbol);
        return 0
    }
}

func (node *IntNode) Size() int {
    return len(node.values)
}

func (node *IntNode) DataLength() int {
    return node.byteSize * node.Size()
}

func (node *IntNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return []byte{}
    }

    buf := make([]byte, 8)

    for _, value := range node.values {
        bits := uint64(value)
        binary.BigEndian.PutUint64(buf, bits)
        result = append(result, buf[8 - node.byteSize:]...)
    }

    return result
}

func (node *IntNode) ToSml() string {
    if node.Size() == 0 {
        return fmt.Sprintf("<%s<[0]>", node.symbol)
    }
    values := make([]string, 0, node.Size())
    for _, v := range node.values {
        values = append(values, strconv.FormatInt(v, 10))
    }
    return fmt.Sprintf("<%s[%d] %v>", node.symbol, node.Size(), strings.Join(values, " "))
}


