package secs_message

import (
	"fmt"
	"strings"
)

type ASCIINode struct {
    value  string
    symbol string
}

func (node *ASCIINode) Clone() (ElementType) {
    return  &ASCIINode{value: node.value, symbol: node.symbol}
}

func (node *ASCIINode) Values() interface{} {
    return node.value
}

func (node *ASCIINode) Type() string {
    return node.symbol
}

func (node *ASCIINode) Code() byte {
    return 0o20
}


func CreateASCIINode(str string) ElementType {
    if  len(str) > MAX_BYTE_SIZE {
        panic("string length too long")
    }
    node := &ASCIINode{value: str, symbol: "A"}
    return node
}

func (node *ASCIINode) Size() int {
    return len(node.value)
}

func (node *ASCIINode) DataLength() int {
    return len(node.value)
}

func (node *ASCIINode) EncodeBytes() []byte {

    result, err := buildHeader( node.Code() , node.DataLength())
    if err != nil {
        return []byte{}
    }

    for _, ch := range node.value {
        result = append(result, byte(ch))
    }

    return result
}

func (node *ASCIINode) ToSml() string {
    if node.value == "" {
        return "<A[0]>"
    }
    var sb strings.Builder
    inPrintable := false
    for _, ch := range node.value {
        if isPrintableASCII(ch) {
            if !inPrintable {
                inPrintable = true
                sb.WriteString(` "`)
            }
            sb.WriteRune(ch)
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
