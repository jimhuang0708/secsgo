package secs
import (
    "encoding/json"
    //"log/slog"
    "net"
    "time"
    "secs/data"
    "secs/logger"
    "errors"
    sm "secs/secs_message"
)


type HostContext struct {
    BaseContext
    log *logger.Logger
    hsms_ss      *  HSMS_SS
    dstModule * DSTMODULE
    hostModule * HOSTMODULE //for host
}

type secsObj struct {
    MsgType string `json:"msgtype"`
    SML string `json:"sml"`
    TimeStamp string `json:"timestamp"`
}

type UIEvt struct { //use for notify ui something happen
    EvtType string `json:"evttype"`
    Source string `json:"source"`
    Data any `json:"data,omitempty"`
}


type UICmd struct {
    Stream int `json:"stream"`
    Function int `json:"function"`
    DataItem sm.Node `json:"dataitem"`
}

func CreateHostContext(deviceID int,mode string,addr string, hostLog *logger.Logger ) *HostContext {
    baseLog := hostLog.With("Context", "HostCtx", "deviceID", deviceID,"Session_mode", mode)
    hc := &HostContext {
              BaseContext: CreateBaseContext(deviceID,baseLog),
              log: baseLog,
              hostModule : CreateHOSTMODULE( baseLog.With("Module", "HOSTMODULE")) ,
              dstModule : CreateDSTMODULE( baseLog.With("Module", "DSTMODULE")),
              hsms_ss : nil,
          }
    hc.BaseContext.attacher = hc
    hc.BaseContext.msgsender = hc
    data.SetLogger(baseLog.With("module", "DATA"));
    go hc.stateRun(mode,addr)
    return hc
}

func (hc *HostContext)regProcessModule(){
    /*clean route path */
    for s := 0 ; s < 255 ; s++ {
        for f := 0 ; f < 255 ; f++ {
            hc.dispatchMap[s][f] = hc.hostModule
        }
    }
    /* stream 13 */
    hc.dispatchMap[13][1] = hc.dstModule
    hc.dispatchMap[13][2] = hc.dstModule
    hc.dispatchMap[13][3] = hc.dstModule
    hc.dispatchMap[13][4] = hc.dstModule
    hc.dispatchMap[13][5] = hc.dstModule
    hc.dispatchMap[13][6] = hc.dstModule
    hc.dispatchMap[13][7] = hc.dstModule
    hc.dispatchMap[13][8] = hc.dstModule
}

func (hc *HostContext)AttachSession(conn net.Conn,mode string){
    ts := CreateTransport(conn, hc.log.With("Component", "transport"))
    hc.hsms_ss = CreateHSMS_SS(mode,ts,hc.deviceID, hc.log.With("Component", "hsms_ss"))
    hc.ctrlChan <- "update_hsms_ss"
}

func (hc *HostContext)SendMsg(e Evt){
    if(hc.hsms_ss != nil){
        hc.hsms_ss.iChan <- e
    } else {
        hc.log.Printf("Error : SendMsg but hsms_ss not exitst\n");
    }
}


func (hc *HostContext) GenericCB(err error,s *SendCtx,r * RecvCtx)(int){
    if(err != nil ){
        if(errors.Is(err,ErrTimeout)){
            hc.log.Printf("HostContext Timeout %v\n",s);
        } else {
            hc.log.Printf("HostContext Unknown Error %v\n",err);
        }
    } else {
        hc.log.Printf("HostContext get ack %v\n",r);
    }
    return 0
}


func (hc *HostContext)sendSXFY(stream int , function int , node sm.ElementType) {
    msg := sm.CreateDataMessage( stream, function , true , node , hc.deviceID , 0 , "ALL" )
    ctx := &SendCtx{ msg : msg , cb : hc.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    hc.hsms_ss.iChan <- act
    return
}

func (hc *HostContext)doUICommand(s string) {
    raw := []byte(s)
    var c UICmd
    json.Unmarshal( raw,&c)
    hc.log.Printf("doUICommand %s | %v",raw,c);
    //
    stream := c.Stream
    function := c.Function
    node := c.DataItem.EncodeSecs();
    if(node == nil){
        node = sm.CreateEmptyElementType()
    }
    hc.sendSXFY(stream,function,node)
}

func (hc *HostContext)processUIEvt(uievt string){
    if( hc.UIEvtChan != nil ){
        select {
            case hc.UIEvtChan <- uievt: // not full
            default:
                // full → pop oldest
                <-hc.UIEvtChan
                hc.UIEvtChan <- uievt
        }
    }
}


func (hc *HostContext)processEvt(evt Evt){
    if(evt.cmd == "uievent"){
        hc.processUIEvt(evt.ctx.(string))
    } else if(evt.cmd == "HSMS_SS_EXIT"){
        uievt := &UIEvt{ EvtType : "disconnect" , Source : "Transport" , Data : nil }
        jsonData, _ := json.Marshal(uievt)
        hc.processUIEvt(string(jsonData))
        hc.ctrlChan <- "quit" //interal goroutine , don't use hc.Stop()
    } else if( evt.cmd == "recv" ) {
        //hc.hostModule.iChan <- evt
        hc.regProcessModule();
        if(hc.dispatchHSMSDataMsg(evt)){
            return
        }
        hc.sendUnknownError(evt.ctx.(*RecvCtx).msg.(*sm.DataMessage))
    }  else {
        if(evt.cmd == "READERROR" || evt.cmd == "T8_TIMEOUT" || evt.cmd == "WRITEERROR"){
            hc.log.Printf("Error | Event : %s",evt.cmd)
            return
        }
    }
}

func (hc *HostContext)stateRun(mode string,addr string){
    quit := make(chan struct{})
    defer func(){
        close(quit)
        hc.wg.Wait()
        if(hc.hsms_ss != nil){
            hc.hsms_ss.Stop()
        }
        hc.dstModule.moduleStop()
        hc.hostModule.moduleStop()
        hc.log.Printf("Exit HostContext")
    }()
    hc.wg.Add(1)
    go hc.Connect(mode,addr,quit)
    var hsms_oChan <-chan Evt
    for {
        select {
            case o := <-hsms_oChan:
                hc.log.Printf("get from hsms_ss.oChan %v",o);
                hc.processEvt(o)
            case o := <-hc.hostModule.oChan:
                hc.log.Printf("get from hc.hostModule.oChan %v",o);
                if(o.cmd == "uievent"){
                    hc.processUIEvt(o.ctx.(string))
                } else {
                    hc.hsms_ss.iChan <- o
                }
            case o := <-hc.dstModule.oChan:
                hc.log.Printf("get from hc.dstModule.oChan %v",o);
                if(o.cmd == "uievent"){
                    hc.processUIEvt(o.ctx.(string))
                } else {
                    hc.hsms_ss.iChan <- o
                }

            case o := <- hc.UICmdChan:
                hc.doUICommand(o)
            case cmd :=<-hc.ctrlChan:
                if(cmd == "quit"){
                    return
                }
                if(cmd == "update_hsms_ss"){
                    hsms_oChan = hc.hsms_ss.oChan
                    break
                }
        }
    }
}

////////////////////API

func (hc *HostContext)ReadEq(dsName string) {
    fn := func() {
        hc.dstModule.sendS13F3(1 , dsName  , 0)
    }
    hc.dstModule.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}

func (hc *HostContext)WriteEq(dsName string) {
    fn := func() {
        hc.dstModule.sendS13F1( dsName )
    }
    hc.dstModule.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}

func (hc *HostContext)GetUIEvt()(string,bool){
    select {
        case s := <- hc.UIEvtChan :
            return s , true
        default:
            return "" , false
    }
    return "" , false
}

func (hc *HostContext)PutUICmd(data string){
    hc.UICmdChan <- data
}
