package secs_message

import (
    "fmt"
    "math"
    "strconv"
    "strings"
    "encoding/binary"
    "reflect"
)

type FloatNode struct {
        Node
}

func(node *FloatNode) Clone() (ElementType){
    nodeValues := make([]any,  len(node.NodeValues))
    copy(nodeValues,node.NodeValues)
    return &FloatNode{ Node : Node{ NodeType : node.NodeType , NodeValues : nodeValues }  }
}

func (node *FloatNode) Values() []any {
    out := make([]any, len(node.NodeValues))
    for i, v := range node.NodeValues {
        out[i] = v
    }
    return out
}

func (node *FloatNode) Type() string {
    return node.NodeType
}

func (node *FloatNode) ByteSize() int {
    if(node.NodeType == "F4"){
        return 4
    }else if(node.NodeType == "F8"){
        return 8
    }else{
        log.Printf("Error unknown Float Type %s\n",node.NodeType);
        return 0
    }
}


func (node *FloatNode) Code() byte {
    if(node.NodeType == "F4"){
        return 0o44
    } else if(node.NodeType == "F8"){
        return 0o40
    } else {
        log.Printf("Error unknown float symbol %s\n",node.NodeType);
        return 0 //error
    }
}

func CreateFloatNode(byteSize int, values ...interface{}) ElementType {
    if byteSize*len(values) > MAX_BYTE_SIZE {
        panic("Float datalength too long")
    }

    nodeValues := make([]any, 0, len(values))

    for _, v := range values {
        f, err := toFloat64(v)
        if err != nil {
            panic(err)
        }
        nodeValues = append(nodeValues, f)
    }
    node := &FloatNode{ Node : Node{NodeType : fmt.Sprintf("F%d", byteSize) , NodeValues : nodeValues } }
    return node
}

func toFloat64(v interface{}) (float64, error) {
    switch x := v.(type) {
    case int, int8, int16, int32, int64:
        return float64(reflect.ValueOf(x).Int()), nil
    case uint, uint8, uint16, uint32, uint64:
        return float64(reflect.ValueOf(x).Uint()), nil
    case float32:
        return float64(x), nil
    case float64:
        return x, nil
    default:
        return 0, fmt.Errorf("Convert to float failed  | %T", v)
    }
}


func (node *FloatNode) Size() int {
    return len(node.NodeValues)
}

func (node *FloatNode) DataLength() int {
    return node.ByteSize() * node.Size()
}


func (node *FloatNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(),node.DataLength())
    if err != nil {
        return []byte{}
    }

    buf := make([]byte, 8)

    for _, value := range node.NodeValues {
        var bits uint64

        if node.ByteSize() == 4 {
            bits = uint64(math.Float32bits(value.(float32)))
        } else {
            bits = math.Float64bits(value.(float64))
        }
        binary.BigEndian.PutUint64(buf, bits)
        result = append(result, buf[8 - node.ByteSize():]...)
    }

    return result
}

func (node *FloatNode) ToSml() string {
    if node.Size() == 0 {
        return fmt.Sprintf("<%s[0]>", node.NodeType)
    }
    values := make([]string, 0, node.Size())
    for _, v := range node.NodeValues {
        values = append(values, strconv.FormatFloat(v.(float64), 'g', -1, node.ByteSize()*8))
    }
    return fmt.Sprintf("<%s[%d] %v>", node.NodeType, node.Size(), strings.Join(values, " "))
}
