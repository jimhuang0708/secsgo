package secs_message

import (
    "encoding/binary"
    "math"
)

const (
    formatCodeList    = 0o00
    formatCodeBinary  = 0o10
    formatCodeBoolean = 0o11
    formatCodeASCII   = 0o20
    formatCodeI8      = 0o30
    formatCodeI1      = 0o31
    formatCodeI2      = 0o32
    formatCodeI4      = 0o34
    formatCodeF8      = 0o40
    formatCodeF4      = 0o44
    formatCodeU8      = 0o50
    formatCodeU1      = 0o51
    formatCodeU2      = 0o52
    formatCodeU4      = 0o54
)

func Decode(input []byte) (msg HSMSMessage, ok bool) {
    defer func() {
        if r := recover(); r != nil {
            ok = false
        }
    }()

    p := &decoder{input: input}
    if ok := p.decodeMessageLength(); !ok {
        return p.msg, false
    }
    if ok := p.decodeMessage(); !ok {
        return p.msg, false
    }
    return p.msg, true
}

type decoder struct {
    input     []byte
    pos       int
    msgLength int
    msg       HSMSMessage
}

func (p *decoder) decodeMessageLength() (ok bool) {
    if len(p.input) < 14 {
        return false
    }
    lengthBytes := p.input[0:4]
    p.pos += 4
    p.msgLength = int(binary.BigEndian.Uint32(lengthBytes))
    return len(p.input[p.pos:]) == p.msgLength
}

func (p *decoder) decodeMessage() (ok bool) {
    headerBytes := p.input[p.pos : p.pos+10]
    p.pos += 10

    if headerBytes[4] != 0 {
        return false
    }

    switch headerBytes[5] {
    case TypeDataMessage:
        stream := int(headerBytes[2] & 0b01111111)
        function := int(headerBytes[3])
        waitBit := (headerBytes[2] >> 7) == 1
        sessionID := int(binary.BigEndian.Uint16(headerBytes[:2]))
        systemBytes := binary.BigEndian.Uint32(headerBytes[6:10])

        dataItem, ok := p.decodeMessageText()
        if !ok {
            return false
        }

        p.msg = CreateDataMessage(stream, function, waitBit, dataItem, sessionID, systemBytes, "")
        return true

    case TypeSelectReq, TypeSelectRsp, TypeDeselectReq, TypeDeselectRsp,
        TypeLinktestReq, TypeLinktestRsp, TypeRejectReq, TypeSeparateReq:

        p.msg = CloneHSMSControlMessage(headerBytes)
        return true
    }

    return false
}

//
// ==========================
//   FORMAT DISPATCH TABLE
// ==========================
//

var formatHandlers map[byte]func(*decoder, int) (ElementType, bool)

func init() {
    formatHandlers = map[byte]func(*decoder, int) (ElementType, bool){
        formatCodeList:    (*decoder).decodeList,
        formatCodeASCII:   (*decoder).decodeASCII,
        formatCodeBinary:  (*decoder).decodeBinary,
        formatCodeBoolean: (*decoder).decodeBoolean,

        formatCodeF4: func(p *decoder, l int) (ElementType, bool) { return p.decodeFloat(4, l) },
        formatCodeF8: func(p *decoder, l int) (ElementType, bool) { return p.decodeFloat(8, l) },

        formatCodeI1: func(p *decoder, l int) (ElementType, bool) { return p.decodeInt(1, l) },
        formatCodeI2: func(p *decoder, l int) (ElementType, bool) { return p.decodeInt(2, l) },
        formatCodeI4: func(p *decoder, l int) (ElementType, bool) { return p.decodeInt(4, l) },
        formatCodeI8: func(p *decoder, l int) (ElementType, bool) { return p.decodeInt(8, l) },

        formatCodeU1: func(p *decoder, l int) (ElementType, bool) { return p.decodeUint(1, l) },
        formatCodeU2: func(p *decoder, l int) (ElementType, bool) { return p.decodeUint(2, l) },
        formatCodeU4: func(p *decoder, l int) (ElementType, bool) { return p.decodeUint(4, l) },
        formatCodeU8: func(p *decoder, l int) (ElementType, bool) { return p.decodeUint(8, l) },
    }
}

func (p *decoder) decodeMessageText() (ElementType, bool) {
    if p.msgLength == 10 {
        return CreateEmptyElementType(), true
    }

    header := p.input[p.pos]
    formatCode := header >> 2
    lenBytes := int(header & 0x03)
    if lenBytes == 0 {
        return CreateEmptyElementType(), false
    }
    p.pos++

    // --- decode item length ---
    length := 0
    for i := 0; i < lenBytes; i++ {
        length = (length << 8) | int(p.input[p.pos+i])
    }
    p.pos += lenBytes

    // --- dispatch handler ---
    handler, ok := formatHandlers[formatCode]
    if !ok {
        return CreateEmptyElementType(), false
    }
    return handler(p, length)
}

//
// ==========================
//   FORMAT HANDLER FUNCTIONS
// ==========================
//

func (p *decoder) decodeList(length int) (ElementType, bool) {
    values := make([]interface{}, length)
    for i := 0; i < length; i++ {
        v, ok := p.decodeMessageText()
        if !ok {
            return CreateEmptyElementType(), false
        }
        values[i] = v
    }
    return CreateListNode(values...), true
}

func (p *decoder) decodeASCII(length int) (ElementType, bool) {
    s := string(p.input[p.pos : p.pos+length])
    p.pos += length
    return CreateASCIINode(s), true
}

func (p *decoder) decodeBinary(length int) (ElementType, bool) {
    buf := p.input[p.pos : p.pos+length]
    p.pos += length
    values := make([]interface{}, length)
    for i, v := range buf {
        values[i] = v
    }
    return CreateBinaryNode(values...), true
}

func (p *decoder) decodeBoolean(length int) (ElementType, bool) {
    buf := p.input[p.pos : p.pos+length]
    p.pos += length
    values := make([]interface{}, length)
    for i, v := range buf {
        values[i] = (v != 0)
    }
    return CreateBooleanNode(values...), true
}

//
// ========== FLOAT ==========
//

func (p *decoder) decodeFloat(byteSize int, length int) (ElementType, bool) {
    if length%byteSize != 0 {
        return CreateEmptyElementType(), false
    }

    cnt := length / byteSize
    values := make([]interface{}, cnt)

    for i := 0; i < cnt; i++ {
        start := p.pos + i*byteSize
        end := start + byteSize
        if byteSize == 4 {
            bits := binary.BigEndian.Uint32(p.input[start:end])
            values[i] = math.Float32frombits(bits)
        } else {
            bits := binary.BigEndian.Uint64(p.input[start:end])
            values[i] = math.Float64frombits(bits)
        }
    }

    p.pos += length
    return CreateFloatNode(byteSize, values...), true
}

//
// ========== SIGNED INT ==========
//

func (p *decoder) readInt(byteSize int, offset int) interface{} {
    switch byteSize {
    case 1:
        return int8(p.input[offset])
    case 2:
        return int16(binary.BigEndian.Uint16(p.input[offset:]))
    case 4:
        return int32(binary.BigEndian.Uint32(p.input[offset:]))
    case 8:
        return int64(binary.BigEndian.Uint64(p.input[offset:]))
    default:
        return nil
    }
}

func (p *decoder) decodeInt(byteSize, length int) (ElementType, bool) {
    if length%byteSize != 0 {
        return CreateEmptyElementType(), false
    }

    count := length / byteSize
    values := make([]interface{}, 0, count)

    for i := 0; i < count; i++ {
        values = append(values, p.readInt(byteSize, p.pos+i*byteSize))
    }

    p.pos += length
    return CreateIntNode(byteSize, values...), true
}

//
// ========== UNSIGNED INT ==========
//

func (p *decoder) readUint(byteSize int, offset int) interface{} {
    switch byteSize {
    case 1:
        return uint8(p.input[offset])
    case 2:
        return binary.BigEndian.Uint16(p.input[offset:])
    case 4:
        return binary.BigEndian.Uint32(p.input[offset:])
    case 8:
        return binary.BigEndian.Uint64(p.input[offset:])
    default:
        return nil
    }
}

func (p *decoder) decodeUint(byteSize, length int) (ElementType, bool) {
    if length%byteSize != 0 {
        return CreateEmptyElementType(), false
    }

    count := length / byteSize
    values := make([]interface{}, 0, count)

    for i := 0; i < count; i++ {
        values = append(values, p.readUint(byteSize, p.pos+i*byteSize))
    }

    p.pos += length
    return CreateUintNode(byteSize, values...), true
}
