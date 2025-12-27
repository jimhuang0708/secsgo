package secs_message
import (
    "fmt"
    "encoding/binary"
)
const (
    TypeDataMessage = 0
    TypeSelectReq   = 1
    TypeSelectRsp   = 2
    TypeDeselectReq = 3
    TypeDeselectRsp = 4
    TypeLinktestReq = 5
    TypeLinktestRsp = 6
    TypeRejectReq   = 7
    TypeSeparateReq = 9

    DataMessageStr = "data message"
    SelectReqStr   = "select.req"
    SelectRspStr   = "select.rsp"
    DeselectReqStr = "deselect.req"
    DeselectRspStr = "deselect.rsp"
    LinktestReqStr = "linktest.req"
    LinktestRspStr = "linktest.rsp"
    RejectReqStr   = "reject.req"
    SeparateReqStr = "separate.req"
)

type HSMSMessage interface {
    EncodeBytes() []byte
    MsgType() int32
    ToSml() string
    SystemBytes() uint32
}

type ControlMessage struct {
    header []byte
}

func CloneHSMSControlMessage(header []byte) HSMSMessage {
    headerCopy := make([]byte, 10)
    for i, b := range header {
    	if i > 10 {
            break
	}
	headerCopy[i] = b
    }
    return &ControlMessage{header: headerCopy}
}

func CreateControlMessageReq( stype byte , systemBytes uint32) HSMSMessage {
    header := make([]byte, 10)
    header[0] = 0xFF
    header[1] = 0xFF
    header[5] = stype
    binary.BigEndian.PutUint32(header[6:], systemBytes)
    return &ControlMessage{header}
}

func CreateControlMessageRejectData(sessionID uint16, sType byte, systemBytes uint32) HSMSMessage {
    header := make([]byte, 10)
    header[0] = byte(sessionID >> 8)
    header[1] = byte(sessionID)
    header[2] = sType
    header[3] = 4 //reason code : got datamessage when not selected
    header[5] = TypeRejectReq
    binary.BigEndian.PutUint32(header[6:], systemBytes)
    return &ControlMessage{header}
}

func CreateControlMessageRsp(req HSMSMessage,parameter ...byte) HSMSMessage {
    header := make([]byte, 10)
    msg, _ := req.(*ControlMessage)
    header[0] = msg.header[0]
    header[1] = msg.header[1]
    if(msg.MsgType() == TypeSelectReq){
        header[3] = parameter[0]
        header[5] = TypeSelectRsp
    } else if(msg.MsgType() == TypeDeselectReq){
        header[3] = parameter[0]
        header[5] = TypeDeselectRsp
    } else if(msg.MsgType() == TypeLinktestReq){
        header[5] =TypeLinktestRsp
    } else {
        fmt.Printf("Error CreateControlMessageRsp");
        return nil
    }
    header[6] = msg.header[6]
    header[7] = msg.header[7]
    header[8] = msg.header[8]
    header[9] = msg.header[9]
    return &ControlMessage{header}
}

func (msg *ControlMessage) ToSml() string {
    msgtype := msg.MsgType()
    switch(msgtype){
        case TypeSelectReq:
            return SelectReqStr
        case TypeSelectRsp:
            return SelectRspStr
        case TypeDeselectReq:
            return DeselectReqStr
        case TypeDeselectRsp:
            return DeselectRspStr
        case TypeLinktestReq:
            return LinktestReqStr
        case TypeLinktestRsp:
            return LinktestRspStr
        case TypeRejectReq:
            return RejectReqStr
        case TypeSeparateReq:
            return SeparateReqStr
        default:
            return "unknown control type"
    }
}

func (msg *ControlMessage) MsgType() int32 {
    if msg.header[4] != 0 {
        return -1 //not HSMS
    }
    if (msg.header[5] < 10){
        return int32(msg.header[5])
    }
    return -1
}

func (msg *ControlMessage) EncodeBytes() []byte {
    result := make([]byte, 14) // 4 bytes length + 10 bytes header
    result[0] = 0
    result[1] = 0
    result[2] = 0
    result[3] = 10
    copy(result[4:], msg.header)
    return result
}

func (msg *ControlMessage) SystemBytes() uint32 {
    return binary.BigEndian.Uint32(msg.header[6:10])
}
