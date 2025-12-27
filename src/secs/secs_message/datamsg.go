package secs_message

import (
    "fmt"
    "bytes"
    "encoding/binary"
	//"unicode"
)

type DataMessage struct {
    sourceHost  string // come from where.
    stream      int
    function    int
    waitBit     bool
    dataItem    ElementType
    sessionID   int
    systemBytes uint32
}


func CreateDataMessage(stream int, function int, waitBit bool, dataItem ElementType, sessionID int, systemBytes uint32,sourceHost string) *DataMessage {
    message := &DataMessage{
        stream:      stream,
        function:    function,
        waitBit:     waitBit,
        dataItem:    dataItem,
        sessionID:   sessionID,
	systemBytes: systemBytes,
        sourceHost : sourceHost,
    }
    return message
}

func (node *DataMessage) StreamCode() int {
    return node.stream
}

func (node *DataMessage) FunctionCode() int {
    return node.function
}

func (node *DataMessage) WaitBit() bool {
    return node.waitBit;
}

func (node *DataMessage) SetWaitBit(waitBit bool) *DataMessage {
    message := &DataMessage{
        stream:      node.stream,
        function:    node.function,
        waitBit:     waitBit,
        dataItem:    node.dataItem,
        sessionID:   node.sessionID,
        systemBytes: node.systemBytes,
        sourceHost : node.sourceHost,
    }
    return message
}

func (node *DataMessage) SetSourceHost(sourceHost string) *DataMessage {
    message := &DataMessage{
        stream:      node.stream,
        function:    node.function,
        waitBit:     node.waitBit,
        dataItem:    node.dataItem,
        sessionID:   node.sessionID,
        systemBytes: node.systemBytes,
        sourceHost : sourceHost,
    }
    return message
}



func (node *DataMessage) SessionID() int {
    return node.sessionID
}

func (node *DataMessage) SystemBytes() uint32 {
    return node.systemBytes
}

func (node *DataMessage) SourceHost() string {
    return node.sourceHost
}

func (node *DataMessage) SetSystemBytes( systemBytes uint32) *DataMessage {
    message := &DataMessage{
 	stream:      node.stream,
	function:    node.function,
	waitBit:     node.waitBit,
	dataItem:    node.dataItem,
	sessionID:   node.sessionID,
	systemBytes: systemBytes,
        sourceHost : node.sourceHost,
    }
    return message
}

func (node *DataMessage) Header() string {
    header := fmt.Sprintf("S%dF%d", node.stream, node.function)
    if (node.waitBit){
        header += " W"
    }
    return header
}

func (node *DataMessage) Get() (ElementType, error) {
    return node.dataItem , nil
}

func (node *DataMessage) MsgType() int32 {
    return 0
}

func (node *DataMessage) EncodeBytes() []byte {
    if node.sessionID == -1 {
        return []byte{}
    }
    itemBytes := node.dataItem.EncodeBytes()
    const headerSize = 10
    msgLength := uint32(len(itemBytes) + headerSize)
    buf := &bytes.Buffer{}
    buf.Grow(int(msgLength) + 4)
    _ = binary.Write(buf, binary.BigEndian, msgLength)
    _ = binary.Write(buf, binary.BigEndian, uint16(node.sessionID))
    header := node.StreamCode()
    if node.WaitBit() {
        header |= 0x80
    }
    buf.WriteByte(byte(header))
    buf.WriteByte(byte(node.FunctionCode()))
    _ = binary.Write(buf, binary.BigEndian, uint16(0))

    systembytesBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(systembytesBuf, node.systemBytes)

    buf.Write(systembytesBuf[:4])
    buf.Write(itemBytes)
    return buf.Bytes()
}

func (node *DataMessage) ToSml() string {
    if _, ok := node.dataItem.(emptyElementType); ok {
        return fmt.Sprintf("%s\n.", node.Header())
    }
    return fmt.Sprintf("%s\n%s\n.", node.Header(), node.dataItem.ToSml())
}

