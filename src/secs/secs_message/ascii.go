package secs_message

import (
	"fmt"
	"strings"
)

type ASCIINode struct {
    Node
}

func (node *ASCIINode) Clone() (ElementType) {
    nodeValues := make([]interface{},  len(node.NodeValues))
    copy(nodeValues,node.NodeValues)
    return  &ASCIINode{ Node{NodeValues : node.NodeValues, NodeType: node.NodeType}}
}

func (node *ASCIINode) Values() []any {
    out := make([]any, len(node.NodeValues))
    for i, v := range node.NodeValues {
        out[i] = v
    }
    return out
}

func (node *ASCIINode) Type() string {
    return node.NodeType
}

func (node *ASCIINode) Code() byte {
    return 0o20
}


func CreateASCIINode(str string) ElementType {
    if  len(str) > MAX_BYTE_SIZE {
        panic("string length too long")
    }
    result := make([]interface{}, 0, len(str))
    for i := 0; i < len(str); i++ {
	result = append(result, interface{}(str[i]))
    }
    node := &ASCIINode{ Node{NodeValues : result , NodeType: "A"}}
    return node
}

func (node *ASCIINode) Size() int {
    return len(node.NodeValues)
}

func (node *ASCIINode) DataLength() int {
    return len(node.NodeValues)
}

func (node *ASCIINode) EncodeBytes() []byte {

    result, err := buildHeader( node.Code() , node.DataLength())
    if err != nil {
        return []byte{}
    }

    for _, ch := range node.NodeValues {
        result = append(result, ch.(byte))
    }

    return result
}

func (node *ASCIINode) ToSml() string {
    if len(node.NodeValues) == 0 {
        return "<A[0]>"
    }
    var sb strings.Builder
    inPrintable := false
    for _, ch := range node.NodeValues {
        if isPrintableASCII(rune(ch.(byte))) {
            if !inPrintable {
                inPrintable = true
                sb.WriteString(` "`)
            }
            sb.WriteRune(rune(ch.(byte)))
            continue
        }
        // non-printable
        if inPrintable {
            inPrintable = false
            sb.WriteString(`"`)
        }
        fmt.Fprintf(&sb, " 0x%02X", ch)
    }
    if inPrintable {
        sb.WriteString(`"`)
    }
    return "<A" + sb.String() + ">"
}

func isPrintableASCII(ch rune) bool {
	return ch >= 32 && ch != 127
}
