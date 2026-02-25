//HSMS-SS (High-Speed SECS Message Service Single Selected Mode）
package secs

import (
    "time"
    "secs/logger"
    "errors"
    sm "secs/secs_message"
)

var (
    ErrTimeout        = errors.New("timeout")
)

type HSMS_SS struct{
    BaseComponent
    ts * Transport
    connectState string
    sysByte    uint32
    deviceID int
    waitQueue map[uint32]*SendCtx
    timer_T7 *time.Timer
}

func CreateHSMS_SS(mode string,ts * Transport, deviceID int, log *logger.Logger) *HSMS_SS {
    o := HSMS_SS{ BaseComponent : CreateBaseComponent(log),
                  connectState : "NOTSELECTED",
                  sysByte    : 0,
                  waitQueue : make(map[uint32]*SendCtx),
                  deviceID : deviceID,
                  ts : ts }
    o.wg.Add(1)
    go o.stateRun(mode)
    return &o
}


func (ss *HSMS_SS)incSysByte(){
    ss.sysByte = ss.sysByte + 1
    if ss.sysByte == 0xFFFFFFFF {
        ss.sysByte = 0
    }
    return
}

func (ss *HSMS_SS)GenericControlCB(err error,s *SendCtx,r *RecvCtx)(int){
    if(err != nil){
        ss.log.Printf("GenericControlCB : T6 timeout!\n");
        ss.detachTransport();
    }
    return 0
}

func (ss *HSMS_SS)sendLinkTestReq(){
    ss.log.Printf("sendLinkTestReq()\n");
    msg := sm.CreateControlMessageReq(sm.TypeLinktestReq,ss.sysByte)
    ctx := &SendCtx{ msg : msg , cb : ss.GenericControlCB , timeout : time.Now().Unix() + (T6/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    ss.waitQueue[ss.sysByte] = ctx
    ss.incSysByte()
    ss.ts.iChan <- act
}

func (ss *HSMS_SS)sendSelectReq(){
    msg := sm.CreateControlMessageReq(sm.TypeSelectReq, ss.sysByte)
    ctx := &SendCtx{ msg : msg , cb : ss.GenericControlCB , timeout : time.Now().Unix() + (T6/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    ss.waitQueue[ss.sysByte] = ctx
    ss.incSysByte()
    ss.ts.iChan <- act
    return
}

func (ss *HSMS_SS)sendRejectReq(msg sm.HSMSMessage){
    rawbytes := msg.EncodeBytes()
    systembytes := msg.SystemBytes();
    sessionid := uint16((rawbytes[0]<<8) | rawbytes[1])
    /* 4 is in not select */
    replyMsg := sm.CreateControlMessageRejectData( sessionid , rawbytes[5] ,systembytes)
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    ss.ts.iChan <- act
}



func (ss *HSMS_SS)sendSelectRsp(msg sm.HSMSMessage){
    replyMsg := sm.CreateControlMessageRsp(msg,0)
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    ss.ts.iChan <- act
    return
}

func (ss *HSMS_SS)sendLinkTestRsp(msg sm.HSMSMessage){
    replyMsg := sm.CreateControlMessageRsp(msg)
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    ss.ts.iChan <- act
    return
}

func (ss *HSMS_SS)processEvt(evt Evt){
    if(evt.cmd == "uievent"){
        ss.oChan <- evt
        return
    }

    if(evt.cmd == "SOCKET_CLOSE"){
        ss.detachTransport();
        return
    }

    if(evt.cmd == "recv"){
        ss.processMsg(evt.ctx.(*RecvCtx).msg.(sm.HSMSMessage))
        return
    }

}

func (ss *HSMS_SS)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , ss.deviceID , ss.sysByte , msg.SourceHost() )
    ctx := &SendCtx{ msg : errmsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    ss.incSysByte()
    ss.iChan <- act
    return
}


func (ss *HSMS_SS)processMsg(msg sm.HSMSMessage){
    v , ok := ss.waitQueue[ msg.SystemBytes() ]
    if ok {
        v.cb(nil,v,&RecvCtx{msg : msg})
        delete(ss.waitQueue, msg.SystemBytes() )
    }
    if(msg.MsgType() == sm.TypeSeparateReq){
        ss.detachTransport();
        return
    }
    if(ss.connectState == "NOTSELECTED"){
        if(msg.MsgType() == sm.TypeSelectReq){
            ss.sendSelectRsp(msg)
            ss.connectState = "SELECTED"
            ss.stopT7()
            ss.oChan <- Evt{ cmd : "NOTIFY_SELECTED" , ctx : nil  }
        } else if(msg.MsgType() == sm.TypeSelectRsp){
            if( msg.EncodeBytes()[4 + 3] == 0 ){
                ss.connectState = "SELECTED"
                ss.oChan <- Evt{ cmd : "NOTIFY_SELECTED" , ctx : nil  }
            } else {
                ss.log.Printf("Select rejected & quit\n");
            }
        } else {
            if(msg.MsgType() == sm.TypeDataMessage){
                ss.log.Printf("Got data message when hsms-ss not selected\n");
                ss.sendRejectReq(msg)
            } else {
                ss.log.Printf("checkSelect() failed ignore : %v\n",msg);
            }
        }
        return
    } else {
        if(msg.MsgType() == sm.TypeLinktestReq){
            ss.sendLinkTestRsp(msg)
            return
        }
        if(msg.MsgType() == sm.TypeLinktestRsp){
            return
        }

        if(msg.MsgType() == sm.TypeSelectReq || msg.MsgType() == sm.TypeSelectRsp){
             ss.log.Printf("Aready selected ignore : %v\n",msg);
             return
        }

        if(msg.SessionID() != ss.deviceID){
            ss.log.Printf(" Handle  sssionID remote : %d | local : %d mismatch | send S9F1 back\n",msg.SessionID(),ss.deviceID)
            ss.sendS9FX(msg.(*sm.DataMessage),1)
            return
        }
        ctx := &RecvCtx{msg : msg}
        ss.oChan <- Evt{ cmd : "recv", ctx : ctx }
        //msg.(*sm.DataMessage).Clone() //TODO : notify this to ui
        return
    }
}


func (ss *HSMS_SS)stopT7() {
    ss.log.Print("STOP T7\n");
    if !ss.timer_T7.Stop() {
        select {
            case <-ss.timer_T7.C:
            default:
        }
    }
}

func (ss *HSMS_SS )handleInput( evt Evt ){
    // determine it is primary message, and append systembytes
    // and put in waitQ
    if(evt.cmd == "send"){
        ctx := evt.ctx.(*SendCtx)
        if(evt.ctx.(*SendCtx).msg.(*sm.DataMessage).WaitBit()){
            ctx.msg = evt.ctx.(*SendCtx).msg.(*sm.DataMessage).SetSystemBytes( ss.sysByte ).SetSessionID(ss.deviceID)
            ss.waitQueue[ss.sysByte] = evt.ctx.(*SendCtx)
            ss.incSysByte()
        } else {
            ctx.msg = evt.ctx.(*SendCtx).msg.(*sm.DataMessage).SetSessionID(ss.deviceID)
        }
        evt.ctx = ctx
        ss.ts.iChan <- evt
    }
}

func (ss *HSMS_SS )detachTransport(){
    ss.log.Printf("Notify Event HSMS_SS_EXIT\n");
    ss.oChan <-Evt{ cmd : "HSMS_SS_EXIT" , ctx : nil }
}

func (ss *HSMS_SS )stateRun(mode string){
    defer func(){
        if(ss.ts != nil){
            ss.ts.Stop()
        }
        ss.log.Printf("Exit HSMS_SS \n");
        ss.wg.Done()
    }()
    //passive check if recv select.req
    ss.timer_T7 = time.NewTimer(T7 * time.Millisecond)
    if( mode == "ACTIVE"){
        ss.sendSelectReq()
        ss.stopT7()//active mode disable T7
    }
    lnktest_ticker := time.NewTicker(60*time.Second)
    waitAct_ticker := time.NewTicker(1*time.Second)
    for {
        select {
            case evt := <-ss.ts.oChan:
                ss.processEvt(evt)
            case evt := <-ss.iChan:
                ss.handleInput( evt  )
            case <-lnktest_ticker.C:
                ss.sendLinkTestReq()
            case <-ss.timer_T7.C:
                ss.log.Printf("T7 comes,Check if selected \n")
                if(ss.connectState != "SELECTED"){
                    ss.log.Printf("NOT Selected Error T7_TIMEOUT -> EXIT\n")
                    ss.oChan <-Evt{ cmd : "disconnect" , ctx : nil }
                    ss.ctrlChan <- "quit"
                    return
                } else {
                    ss.log.Printf("yes , selected \n")
                }
            case <-waitAct_ticker.C:
                for k , v := range ss.waitQueue {
                    if( time.Now().Unix() > v.timeout ){
                        if(v.msg.MsgType() == sm.TypeDataMessage){
                            ss.sendS9FX( v.msg.(*sm.DataMessage),9)
                        }
                        v.cb(ErrTimeout,v,nil)
                        delete(ss.waitQueue,k)
                    }
                }

            case cmd :=<-ss.ctrlChan:
                if(cmd == "quit"){
                    return
                }

        }
    }
    return
}
