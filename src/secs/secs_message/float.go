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
    byteSize  int
    values    []float64
    symbol string
}

func(node *FloatNode) Clone() (ElementType){
    nodeValues := make([]float64,  len(node.values))
    copy(nodeValues,node.values)
    return &FloatNode{node.byteSize, nodeValues, node.symbol}
}

func (node *FloatNode) Values() interface{} {
    return node.values
}

func (node *FloatNode) Type() string {
    return node.symbol
}

func (node *FloatNode) Code() byte {
    if(node.symbol == "F4"){
        return 0o44
    } else if(node.symbol == "F8"){
        return 0o40
    } else {
        fmt.Printf("Error unknown float symbol %s\n",node.symbol);
        return 0 //error
    }
}

func CreateFloatNode(byteSize int, values ...interface{}) ElementType {
    if byteSize*len(values) > MAX_BYTE_SIZE {
        panic("Float datalength too long")
    }

    nodeValues := make([]float64, 0, len(values))

    for _, v := range values {
        f, err := toFloat64(v)
        if err != nil {
            panic(err)
        }
        nodeValues = append(nodeValues, f)
    }

    node := &FloatNode{
        byteSize: byteSize,
        values:   nodeValues,
        symbol: fmt.Sprintf("F%d", byteSize),
    }

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
    return len(node.values)
}

func (node *FloatNode) DataLength() int {
    return node.byteSize * node.Size()
}

func (node *FloatNode) EncodeBytes() []byte {
    result, err := buildHeader(node.Code(),node.DataLength())
    if err != nil {
        return []byte{}
    }

    buf := make([]byte, 8)

    for _, value := range node.values {
        var bits uint64

        if node.byteSize == 4 {
            bits = uint64(math.Float32bits(float32(value)))
        } else {
            bits = math.Float64bits(value)
        }
        binary.BigEndian.PutUint64(buf, bits)
        result = append(result, buf[8 - node.byteSize:]...)
    }

    return result
}

func (node *FloatNode) ToSml() string {
    if node.Size() == 0 {
        return fmt.Sprintf("<%s[0]>", node.symbol)
    }
    values := make([]string, 0, node.Size())
    for _, v := range node.values {
        values = append(values, strconv.FormatFloat(v, 'g', -1, node.byteSize*8))
    }
    return fmt.Sprintf("<%s[%d] %v>", node.symbol, node.Size(), strings.Join(values, " "))
}

