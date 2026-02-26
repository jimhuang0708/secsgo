package secs_message

import (
    "fmt"
    "reflect"
    "math"
    "strconv"
    "strings"
    "encoding/binary"
)

type IntNode struct {
    Node
}

func (node *IntNode) Clone() (ElementType){
    nodeValues := make([]interface{},  len(node.NodeValues))
    copy(nodeValues,node.NodeValues)
    return &IntNode{ Node : Node{ NodeType : node.NodeType , NodeValues : nodeValues }  }
}

func CreateIntNode(byteSize int, values ...interface{}) ElementType {
    if byteSize*len(values) > MAX_BYTE_SIZE {
        panic("int datalength too long")
    }

    nodeValues := make([]interface{} , 0, len(values))

    for _, v := range values {
        iv, ok := convertToInt64(v)
        if !ok {
            panic("input argument contains invalid type for IntNode")
        }
        nodeValues = append(nodeValues, iv)
    }

    node := &IntNode{ Node : Node{NodeType : fmt.Sprintf("I%d", byteSize) , NodeValues : nodeValues } }
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

func (node *IntNode) Values() []any {
    out := make([]any, len(node.NodeValues))
    for i, v := range node.NodeValues {
        out[i] = v
    }
    return out
}

func (node *IntNode) Type() string {
    return node.NodeType
}

func (node *IntNode) Code() byte{
    if(node.NodeType == "I1"){
        return 0o31
    }else if(node.NodeType == "I2"){
        return 0o32
    }else if(node.NodeType == "I4"){
        return 0o34
    }else if(node.NodeType == "I8"){
        return 0o30
    }else{
        log.Printf("Error unknown Int symbol %s\n",node.NodeType);
        return 0
    }
}

func (node *IntNode) Size() int {
    return len(node.NodeValues)
}

func (node *IntNode) ByteSize() int {
    if(node.NodeType == "I1"){
        return 1
    }else if(node.NodeType == "I2"){
        return 2
    }else if(node.NodeType == "I4"){
        return 4
    }else if(node.NodeType == "I8"){
        return 8
    }else{
        log.Printf("Error unknown Int Type %s\n",node.NodeType);
        return 0
    }
}


func (node *IntNode) DataLength() int {
    return node.ByteSize() * node.Size()
}

func (node *IntNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return []byte{}
    }

    buf := make([]byte, 8)

    for _ , value := range node.NodeValues {
        bits := uint64(value.(int64))
        binary.BigEndian.PutUint64(buf, bits)
        result = append(result, buf[8 - node.ByteSize():]...)
    }

    return result
}

func (node *IntNode) ToSml() string {
    if node.Size() == 0 {
        return fmt.Sprintf("<%s<[0]>", node.Type())
    }
    values := make([]string, 0, node.Size())
    for _, v := range node.NodeValues {
        values = append(values, strconv.FormatInt(v.(int64) , 10))
    }
    return fmt.Sprintf("<%s[%d] %v>", node.Type(), node.Size(), strings.Join(values, " "))
}
