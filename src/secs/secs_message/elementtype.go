package secs_message

import (
    "fmt"
    "encoding/json"
//    "reflect"
)

const MAX_BYTE_SIZE = 1<<24 - 1

type Node struct {
    NodeType   string       `json:"type"`              // "U4", "A", "L", "B", "F4", ...
    NodeValues []interface{}    `json:"values,omitempty"`  // numeric type , bool ,byte
}

type Node_Wire struct {
    NodeType   string       `json:"type"`              // "U4", "A", "L", "B", "F4", ...
    NodeValues json.RawMessage `json:"values"`
}


type ElementType interface {
    Code() byte
    Size() int
    DataLength() int
    EncodeBytes() []byte
    Values() []any
    Type() string
    ToSml() string
    Clone()(ElementType)
}

type emptyElementType struct{}

func (node emptyElementType) Clone() (ElementType) {
    return emptyElementType{}
}

func (node emptyElementType) Code() (byte) {
    return 0
}

func (node emptyElementType) Values() []any {
    return nil
}

func (node emptyElementType) Type() string {
    return "empty"
}

func CreateEmptyElementType() ElementType {
    return emptyElementType{}
}

func (node emptyElementType) Size() int {
    return 0
}

func (node emptyElementType) ByteSize() int {
    return 0
}


func (node emptyElementType) DataLength() int {
    return 0
}

func (node emptyElementType) EncodeBytes() []byte {
    return []byte{}
}

func (node emptyElementType) ToSml() string {
    return ""
}

func buildHeader(code byte, n int) ([]byte, error) {
    if n > MAX_BYTE_SIZE {
        return nil, fmt.Errorf("datalength too long")
    }
    raw := []byte{byte(n >> 16), byte(n >> 8), byte(n)}
    for len(raw) > 1 && raw[0] == 0 {
        raw = raw[1:]
    }
    header := append([]byte{(code << 2) | byte(len(raw))}, raw...)
    return header, nil
}

func (n *Node) EncodeSecs() (ElementType) {
    out := make([]interface{}, len(n.NodeValues))
    for i, v := range n.NodeValues {
        out[i] = v
    }
    switch n.NodeType {
    case "L":
        for k , v := range out {
            if _ , ok := v.(ElementType); !ok {
                out[k] = v.(*Node).EncodeSecs()
            }
        }
        return &ListNode{ Node : Node{ NodeType : n.NodeType , NodeValues : out } }
    case "I1" , "I2" , "I4" , "I8":
        return &IntNode{ Node : Node{ NodeType : n.NodeType , NodeValues : out } }
    case "U1" , "U2" , "U4" , "U8":
        return &UintNode{ Node : Node{ NodeType : n.NodeType , NodeValues : out } }
    case "A":
        return &ASCIINode{ Node : Node{ NodeType : n.NodeType , NodeValues : out } }
    case "B":
        return &BinaryNode{ Node : Node{ NodeType : n.NodeType , NodeValues : out } }
    case "BOOLEAN":
        return &BooleanNode{ Node : Node{ NodeType : n.NodeType , NodeValues : out } }

    }
    return nil
}

func (n *Node) Clone() (Node) {
    out := make([]interface{}, len(n.NodeValues))
    for i, v := range n.NodeValues {
        out[i] = v
    }
    return Node{ NodeType : n.NodeType , NodeValues : out }
}

func (n *Node) UnmarshalJSON(data []byte) error {
    var w Node_Wire
    if err := json.Unmarshal(data, &w); err != nil {
        return err
    }
    n.NodeType = w.NodeType
    switch n.NodeType {
    case "L":
        var arr []*Node
        if err := json.Unmarshal(w.NodeValues, &arr); err != nil {
            return fmt.Errorf("L values must be array of NodeValue: %w", err)
        }
        n.NodeValues = make([]any, 0, len(arr))
        for i := range arr {
            n.NodeValues = append(n.NodeValues, arr[i])
        }
        return nil

    case "A":
            var bs []string
            if err := json.Unmarshal(w.NodeValues, &bs); err != nil {
                return fmt.Errorf("Ascii NodeValues must be array of ascii: %w", err)
            }
            n.NodeValues = make([]any, 0, len(bs))
            for _, b := range bs {
                n.NodeValues = append(n.NodeValues, b[0])
            }
            return nil

            return nil

    case "BOOLEAN":
            var bs []bool
            if err := json.Unmarshal(w.NodeValues, &bs); err != nil {
                return fmt.Errorf("Boolean NodeValues must be array of bool: %w", err)
            }
            n.NodeValues = make([]any, 0, len(bs))
            for _, b := range bs {
                n.NodeValues = append(n.NodeValues, b)
            }
            return nil

    case "B":
            var bs []byte
            if err := json.Unmarshal(w.NodeValues, &bs); err != nil {
                return fmt.Errorf("Binary NodeValues must be array of binary: %w", err)
            }
            n.NodeValues = make([]any, 0, len(bs))
            for _, b := range bs {
                n.NodeValues = append(n.NodeValues, b)
            }
            return nil
     case "I1" , "I2" , "I4" , "I8" :
            var bs []int64
            if err := json.Unmarshal(w.NodeValues, &bs); err != nil {
                return fmt.Errorf("Boolean NodeValues must be array of bool: %w", err)
            }
            n.NodeValues = make([]any, 0, len(bs))
            for _, b := range bs {
                n.NodeValues = append(n.NodeValues, b)
            }

            return nil
     case "U1" , "U2" , "U4" , "U8" :
            var bs []uint64
            if err := json.Unmarshal(w.NodeValues, &bs); err != nil {
                return fmt.Errorf("Boolean NodeValues must be array of bool: %w", err)
            }
            n.NodeValues = make([]any, 0, len(bs))

            for _, b := range bs {
                n.NodeValues = append(n.NodeValues, b)
            }
            return nil
    }
    return nil;
}
