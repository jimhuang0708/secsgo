package secs_message

import (
    "fmt"
)

const MAX_BYTE_SIZE = 1<<24 - 1

type ElementType interface {
    Code() byte
    Size() int
    DataLength() int
    EncodeBytes() []byte
    Values() interface{}
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

func (node emptyElementType) Values() interface{} {
    return ""
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
