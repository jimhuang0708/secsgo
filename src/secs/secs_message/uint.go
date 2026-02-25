package secs_message

import (
    "fmt"
    "reflect"
    "strconv"
    "strings"
    "encoding/binary"
)

type UintNode struct {
    Node
}

func(node *UintNode) Clone() (ElementType) {
    nodeValues := make([]interface{},  len(node.NodeValues))
    copy(nodeValues,node.NodeValues)
    return &UintNode{ Node : Node{ NodeType : node.NodeType , NodeValues : nodeValues }  }

}


func (node *UintNode) Values() interface{} {
    out := make([]uint64, len(node.NodeValues))
    for i, v := range node.NodeValues {
        out[i] = v.(uint64)
    }
    return out
}

func (node *UintNode) Type() string {
    return node.NodeType
}

func (node *UintNode) Code() byte {
    if(node.NodeType == "U1"){
        return 0o51
    } else if(node.NodeType == "U2"){
        return 0o52
    } else if(node.NodeType == "U4"){
        return 0o54
    } else if(node.NodeType == "U8"){
        return 0o50
    } else {
        log.Printf("Error unknown uint symbol %s\n",node.NodeType);
        return 0
    }
}

func CreateUintNode(byteSize int, values ...interface{}) ElementType {
    if byteSize*len(values) > MAX_BYTE_SIZE {
        panic("uint datalength too long")
    }

    nodeValues := make([]interface{}, 0, len(values))

    for _, v := range values {
        uv, ok := convertToUint64(v)
        if !ok {
            panic("input argument contains invalid type for UintNode")
        }
        nodeValues = append(nodeValues, uv)
    }

    node := &UintNode{ Node : Node{ NodeType : fmt.Sprintf("U%d", byteSize) , NodeValues : nodeValues } }
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
    return len(node.NodeValues)
}

func (node *UintNode) ByteSize() int {
    if(node.NodeType == "U1"){
        return 1
    } else if(node.NodeType == "U2"){
        return 2
    } else if(node.NodeType == "U4"){
        return 4
    } else if(node.NodeType == "U8"){
        return 8
    } else {
        log.Printf("Error unknown uint symbol %s\n",node.NodeType);
        return 0
    }
}


func (node *UintNode) DataLength() int {
    return node.ByteSize() * node.Size()
}

func (node *UintNode) EncodeBytes() []byte {
    header, err := buildHeader(node.Code(), node.DataLength())
    if err != nil {
        return nil
    }
    result := make([]byte, 0, len(header)+len(node.NodeValues)*node.ByteSize())
    result = append(result, header...)
    var tmp [8]byte
    for _, v := range node.NodeValues {
        binary.BigEndian.PutUint64(tmp[:], uint64(v.(uint64)))
        result = append(result, tmp[8 - node.ByteSize():8]...)
    }
    return result
}

func (node *UintNode) ToSml() string {
    if node.Size() == 0 {
        return fmt.Sprintf("<%s[0]>", node.NodeType)
    }
    values := make([]string, 0, node.Size())
    for _, v := range node.NodeValues {
        values = append(values, strconv.FormatUint(v.(uint64), 10))
    }
    return fmt.Sprintf("<%s[%d] %v>", node.NodeType , node.Size(), strings.Join(values, " "))
}

