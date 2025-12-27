package secs_message

import (
    "reflect"
    "fmt"
    "strconv"
    "strings"
    "encoding/binary"
)

type UintNode struct {
    byteSize  int
    values    []uint64
    symbol string
}

func(node *UintNode) Clone() (ElementType) {
    nodeValues := make([]uint64,  len(node.values))
    copy(nodeValues,node.values)
    return &UintNode{node.byteSize, nodeValues, node.symbol }
}


func (node *UintNode) Values() interface{} {
    return node.values
}

func (node *UintNode) Type() string {
    return node.symbol
}

func (node *UintNode) Code() byte {
    if(node.symbol == "U1"){
        return 0o51
    } else if(node.symbol == "U2"){
        return 0o52
    } else if(node.symbol == "U4"){
        return 0o54
    } else if(node.symbol == "U8"){
        return 0o50
    } else {
        fmt.Printf("Error unknown uint symbol %s\n",node.symbol);
        return 0
    }
}

func CreateUintNode(byteSize int, values ...interface{}) ElementType {
    if byteSize*len(values) > MAX_BYTE_SIZE {
        panic("uint datalength too long")
    }

    nodeValues := make([]uint64, 0, len(values))

    for _, v := range values {
        uv, ok := convertToUint64(v)
        if !ok {
            panic("input argument contains invalid type for UintNode")
        }
        nodeValues = append(nodeValues, uv)
    }

    node := &UintNode{
        byteSize: byteSize,
        values:   nodeValues,
        symbol: fmt.Sprintf("U%d", byteSize),
    }
    return node
}

func convertToUint64(v interface{}) (uint64, bool) {
    switch value := v.(type) {
    case int, int8, int16, int32, int64:
        iv := reflect.ValueOf(value).Int()
        if iv < 0 {
            panic("converted to uint64 failed | negative")
        }
        return uint64(iv), true

    case uint, uint8, uint16, uint32, uint64:
        return reflect.ValueOf(value).Uint(), true

    case float32:
        if value < 0 {
            panic("converted to uint64 failed | negative")
        }
        return uint64(value), true

    case float64:
        if value < 0 {
            panic("converted to uint64 failed | negative")
        }
        return uint64(value), true

    default:
        return 0, false
    }
}

func (node *UintNode) Size() int {
    return len(node.values)
}

func (node *UintNode) DataLength() int {
    return node.byteSize * node.Size()
}

func (node *UintNode) EncodeBytes() []byte {
    header, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return nil
    }
    result := make([]byte, 0, len(header)+len(node.values)*node.byteSize)
    result = append(result, header...)
    var tmp [8]byte
    for _, v := range node.values {
        binary.BigEndian.PutUint64(tmp[:], uint64(v))
        result = append(result, tmp[8 - node.byteSize:8]...)
    }
    return result
}

func (node *UintNode) ToSml() string {
    if node.Size() == 0 {
        return fmt.Sprintf("<%s[0]>", node.symbol)
    }
    values := make([]string, 0, node.Size())
    for _, v := range node.values {
        values = append(values, strconv.FormatUint(v, 10))
    }
    return fmt.Sprintf("<%s[%d] %v>", node.symbol , node.Size(), strings.Join(values, " "))
}

