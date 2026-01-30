package secs

import (
    "net"
    "time"
//    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

type SessionAttacher  interface {
    AttachSession(conn net.Conn, mode string)
}

type BaseContext struct {
    attacher  SessionAttacher
    iChan chan Evt
    oChan chan Evt
    run bool
    dispatchMap [255][255]MSGMODULE
    deviceID int
    log *logger.Logger
}


func (bc *BaseContext)buildError(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , bc.deviceID , 0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    bc.oChan <- act
    return
}

func (bc *BaseContext)dispatchHSMSDataMsg(evt Evt)(bool){
    msg := evt.msg.(*sm.DataMessage)
    // all sessionId shoule be same as equipment's DEVICE ID
    if(msg.SessionID() != bc.deviceID){
        bc.buildError(msg,1)
        //TODO :  should send separate req
        bc.log.Printf("Incorrect session id : %d != %d | %s",msg.SessionID(),bc.deviceID,msg.ToSml())
        return true
    }
    s := msg.StreamCode()
    f := msg.FunctionCode()
    if(bc.dispatchMap[s][f] != nil){
        bc.dispatchMap[s][f].PutEvt(evt)
        return true
    }
    return false
}

func (bc *BaseContext)sendUnknownError(msg *sm.DataMessage){
    s := msg.StreamCode()
    for idx := 0 ; idx < 255 ; idx++ {
        if(bc.dispatchMap[s][idx] != nil){
            bc.buildError(msg,5)//unknown function
            return
        }
    }
    bc.buildError(msg,3)//unknown stream
}


func (bc *BaseContext)ConnectActive(addr string,quit <-chan struct{}){
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        bc.log.Printf("Error dialing : %v\n", err)
        return
    }
    defer conn.Close()
    bc.attacher.AttachSession(conn,"ACTIVE")
    <-quit
    bc.log.Printf("Exit ConnectActive\n");
    return
}

func (bc *BaseContext)ConnectPassive(addr string,quit <-chan struct{}) {
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        bc.log.Printf("Error net.Listen: %v\n", err)
        return
    }

    defer ln.Close()

    go func() {
        <-quit
        bc.log.Printf("ConnectPassive quit\n")
        ln.Close()
    }()

    for {
        conn, err := ln.Accept()
        if err != nil {
            bc.log.Printf("Exit ConnectPassive: %v\n", err)
            return
        }
        bc.attacher.AttachSession(conn, "PASSIVE")
    }
}

func (bc *BaseContext)Connect(mode string,addr string,quit <-chan struct{}) {
    if(mode == "ACTIVE"){
        bc.ConnectActive(addr,quit)
    }
    if(mode == "PASSIVE"){
        bc.ConnectPassive(addr,quit)
    }
    bc.log.Printf("Error : mode %s not support\n",mode);
}
