//HSMS-SS (High-Speed SECS Message Service Single Selected Mode）
package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "sync"
)

type WaitItem struct {
    msg sm.HSMSMessage
    ts  int64
    evtChan chan Evt
    evt Evt
}


type HSMS_SS struct{
    ts * Transport
    iChan chan Evt
    oChan chan Evt
    connectState string
    run        string
    wg         *sync.WaitGroup
    sysByte    uint32
    waitQueue map[uint32]WaitItem
    timer_T7 *time.Timer
}

func NewHSMS_SS(mode string,ts * Transport) *HSMS_SS {
    o := HSMS_SS{
                         connectState : "NOTSELECTED",
                         run : "stop",
                         iChan : make(chan Evt,10),
                         oChan : make(chan Evt,10 ) ,
                         wg    : new(sync.WaitGroup),
                         sysByte    : 0,
                         waitQueue : make(map[uint32]WaitItem),
                         ts : ts,
                     }
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

func (ss *HSMS_SS)sendLinkTestReq(){
    fmt.Printf("sendLinkTestReq()\n");
    act := Evt{ cmd : "send" , msg : sm.CreateControlMessageReq(sm.TypeLinktestReq,ss.sysByte),ts : time.Now().Unix() }
    alarmEvt := Evt{ cmd : "T6_TIMEOUT" , msg : act.msg ,ts : time.Now().Unix() }
    ss.waitQueue[ss.sysByte] = WaitItem { evt : alarmEvt,ts : act.ts + (T6/1000) , evtChan : ss.iChan }
    ss.incSysByte()
    ss.ts.iChan <- act
}

func (ss *HSMS_SS)sendSelectReq(){
    act := Evt{ cmd : "send", msg : sm.CreateControlMessageReq(sm.TypeSelectReq, ss.sysByte),ts : time.Now().Unix() }
    alarmEvt := Evt{ cmd : "T6_TIMEOUT" , msg : act.msg  ,ts : time.Now().Unix() }
    ss.waitQueue[ss.sysByte] = WaitItem { evt : alarmEvt ,ts : act.ts + (T6/1000) , evtChan : ss.iChan}
    ss.incSysByte()
    ss.ts.iChan <- act
    return
}

func (ss *HSMS_SS)sendRejectReq(msg sm.HSMSMessage){
    rawbytes := msg.EncodeBytes()
    systembytes := msg.SystemBytes();
    sessionid := uint16((rawbytes[0]<<8) | rawbytes[1])
    /* 4 is in not select */
    act := Evt{ cmd : "send" , msg : sm.CreateControlMessageRejectData( sessionid , rawbytes[5] ,systembytes),ts : time.Now().Unix() }
    ss.ts.iChan <- act
}



func (ss *HSMS_SS)sendSelectRsp(msg sm.HSMSMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateControlMessageRsp(msg,0) ,ts : time.Now().Unix() }
    ss.ts.iChan <- act
    return
}

func (ss *HSMS_SS)sendLinkTestRsp(msg sm.HSMSMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateControlMessageRsp(msg),ts : time.Now().Unix() }
    ss.ts.iChan <- act
    return
}

func (ss *HSMS_SS)processEvt(evt Evt){
    if(evt.cmd == "uievent"){
        ss.oChan <- evt
        return
    }

    if(evt.cmd == "disconnect"){
        fmt.Printf("Disconnect detachTransport()\n");
        ss.detachTransport();
        return
    }

    if(evt.cmd == "recv"){
        ss.processMsg(evt.msg.(sm.HSMSMessage))
        return
    }

}

func (ss *HSMS_SS)processMsg(msg sm.HSMSMessage){
    _ , ok := ss.waitQueue[ msg.SystemBytes() ]
    if ok {
        delete(ss.waitQueue, msg.SystemBytes() )
    }

    if(msg.MsgType() == sm.TypeSeparateReq){
        ss.detachTransport();
        fmt.Printf("Get separate.req\n");
        return
    }


    if(ss.connectState == "NOTSELECTED"){
        if(msg.MsgType() == sm.TypeSelectReq){
            ss.sendSelectRsp(msg)
            ss.connectState = "SELECTED"
            ss.stopT7()
            ss.oChan <- Evt{ cmd : "NOTIFY_SELECTED" , msg : nil  }
        } else if(msg.MsgType() == sm.TypeSelectRsp){
            if( msg.EncodeBytes()[4 + 3] == 0 ){
                ss.connectState = "SELECTED"
                ss.oChan <- Evt{ cmd : "NOTIFY_SELECTED" , msg : nil  }
            } else {
                fmt.Printf("Select rejected & quit\n");
            }
        } else {
            if(msg.MsgType() == sm.TypeDataMessage){
                fmt.Printf("Got data message when hsms-ss not selected\n");
                ss.sendRejectReq(msg)
            } else {
                fmt.Printf("checkSelect() failed ignore : %v\n",msg);
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
             fmt.Printf("Aready selected ignore : %v\n",msg);
             return
        }

        ss.oChan <- Evt{ cmd : "recv", msg : msg,ts : time.Now().Unix() }
        return
    }
}

func (ss *HSMS_SS )StateStop(){
     ss.run = "stop"
     ss.wg.Wait()
}

func (ss *HSMS_SS)stopT7() {
    fmt.Print("STOP T7\n");
    if !ss.timer_T7.Stop() {
        select {
            case <-ss.timer_T7.C:
            default:
        }
    }
}

func (ss *HSMS_SS )handleInput( evt Evt ){

    if(evt.cmd == "T3_TIMEOUT"){
        fmt.Printf("T3 timeout just log\n");
        return
    }
    if(evt.cmd == "T6_TIMEOUT"){
        fmt.Printf("T6 timeout  detachTransport()\n");
        ss.detachTransport();
        return
    }

    // determine it is primary message, and append systembytes
    // and put in waitQ

    if(evt.msg.(*sm.DataMessage).WaitBit()){
        evt.ts = time.Now().Unix()
        evt.msg = evt.msg.(*sm.DataMessage).SetSystemBytes( ss.sysByte )
        if(evt.waitAlarm == nil){
            alarmEvt := Evt{ cmd : "T3_TIMEOUT" , msg : evt.msg ,ts : time.Now().Unix() }
            ss.waitQueue[ss.sysByte] = WaitItem {  evt : alarmEvt, ts : evt.ts + (T3/1000) , evtChan : ss.iChan }
        } else {
            ss.waitQueue[ss.sysByte] = evt.waitAlarm.(WaitItem)
        }
        ss.incSysByte()
    }
    ss.ts.iChan <- evt
}

func (ss *HSMS_SS )detachTransport(){
    ss.ts.StateStop();
    ss.oChan <-Evt{ cmd : "disconnect" , msg : nil , ts : time.Now().Unix() }
    ss.run = "stop"
    ss.ts = nil
    fmt.Printf("Get separate.req\n");
}

func (ss *HSMS_SS )stateRun(mode string){
    defer ss.wg.Done()
    //passive check if recv select.req
    ss.timer_T7 = time.NewTimer(T7 * time.Millisecond)
    if( mode == "ACTIVE"){
        ss.sendSelectReq()
        ss.stopT7()//active mode disable T7
    }
    lnktest_ticker := time.NewTicker(60*time.Second)
    waitAct_ticker := time.NewTicker(1*time.Second)
    ss.run = "run"
    for ss.run == "run" {
        select {
            case evt := <-ss.ts.oChan:
                ss.processEvt(evt)
            case evt := <-ss.iChan:
                ss.handleInput( evt  )
            case <-lnktest_ticker.C:
                ss.sendLinkTestReq()
            case <-ss.timer_T7.C:
                fmt.Printf("T7 comes,Check if selected \n")
                if(ss.connectState != "SELECTED"){
                    fmt.Printf("NOT Selected Error T7_TIMEOUT -> EXIT\n")
                    ss.oChan <-Evt{ cmd : "disconnect" , msg : nil }
                    ss.run = "stop"
                    return
                } else {
                    fmt.Printf("yes , selected \n")
                }
            case <-waitAct_ticker.C:
                for k , v := range ss.waitQueue {
                    if( time.Now().Unix() > v.ts ){
                        v.evtChan <- v.evt
                        delete(ss.waitQueue,k)
                    }
                }
        }
    }
    ss.run = "stop"
    if(ss.ts != nil){
        ss.ts.StateStop()
    }
    fmt.Printf("Exit HSMS_SS \n");
    return
}

