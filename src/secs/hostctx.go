package secs
import (
    "encoding/json"
    "log/slog"
    "net"
    "time"

    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)


type HostContext struct {
    BaseContext
    log *logger.Logger
    hsms_ss      *  HSMS_SS
    dstModule * DSTMODULE
    hostModule * HOSTMODULE //for host
    UICmdChan *chan string
    UIEvtChan *chan string
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
    DataItem data.NodeValue `json:"dataitem"`
}

func NewHostContext(deviceID int, l *slog.Logger) *HostContext {
    baseLog := logger.New(l).With("context", "host", "deviceID", deviceID)
    hc := &HostContext{
                         BaseContext: BaseContext{
                             oChan : make(chan Evt,10 ) ,
                             iChan : make(chan Evt,10),
                             run : false,
                             deviceID : deviceID,
                             log: baseLog,
                         },
                         log: baseLog,
                         hostModule : NewHOSTMODULE(deviceID, baseLog.With("module", "HOSTMODULE")) ,
                         dstModule : NewDSTMODULE(deviceID, baseLog.With("module", "DSTMODULE")),
                         UICmdChan : nil,
                         UIEvtChan : nil,
                         hsms_ss : nil,
                     }
    data.SetLogger(baseLog.With("module", "DATA"));
    go hc.stateRun()
    return hc
}

func (hc *HostContext)AttachSession(conn net.Conn,mode string){
    ts := NewTransport(conn, hc.log.With("component", "transport", "session_mode", mode))
    hc.hsms_ss = NewHSMS_SS(mode,ts, hc.log.With("component", "hsms_ss", "session_mode", mode))
}


func (hc *HostContext)sendSXFY(stream int , function int , node sm.ElementType) {
    msg := sm.CreateDataMessage( stream, function , true , node , hc.deviceID , 0 , "ALL" )
    act := Evt{ cmd : "send" , msg : msg ,ts : time.Now().Unix() }
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
    node ,_ := c.DataItem.EncodeSecs();
    if(node == nil){
        node = sm.CreateEmptyElementType()
    }
    hc.sendSXFY(stream,function,node)
}

func (hc *HostContext)processUIEvt(uievt string){
    if( hc.UIEvtChan != nil ){
        select {
            case *hc.UIEvtChan <- uievt: // not full
            default:
                // full → pop oldest
                <-*hc.UIEvtChan
                *hc.UIEvtChan <- uievt
        }
    }
}

func (hc *HostContext)processEvt(evt Evt){
    if(evt.cmd == "uievent"){
        hc.processUIEvt(evt.msg.(string))
    } else if(evt.cmd == "disconnect"){
        uievt := &UIEvt{ EvtType : "disconnect" , Source : "Transport" , Data : nil }
        jsonData, _ := json.Marshal(uievt)
        hc.processUIEvt(string(jsonData))
        hc.StateStop()
    } else {
        hc.hostModule.iChan <- evt
    }
}

func (hc *HostContext)StateStop(){
    hc.run = false
    return
}

func (hc *HostContext)stateRun(){
    hc.run = true
    for hc.run {
        var hsms_oChan <-chan Evt
        if hc.hsms_ss != nil {
            hsms_oChan = hc.hsms_ss.oChan
        } else {
            hsms_oChan = nil
        }
        select {
            case o := <-hsms_oChan:
                hc.log.Printf("get from hsms_ss.oChan %v",o);
                hc.processEvt(o)
            case o := <-hc.hostModule.oChan:
                hc.log.Printf("get from hc.hostModule.oChan %v",o);
                if(o.cmd == "uievent"){
                    hc.processUIEvt(o.msg.(string))
                } else {
                    hc.hsms_ss.iChan <- o
                }
            case o := <-hc.dstModule.oChan:
                hc.log.Printf("get from hc.dstModule.oChan %v",o);
                if(o.cmd == "uievent"){
                    hc.processUIEvt(o.msg.(string))
                } else {
                    hc.hsms_ss.iChan <- o
                }

            case o := <- *hc.UICmdChan:
                hc.doUICommand(o)
            default:
                time.Sleep(100 * time.Millisecond)
        }
    }
    hc.hostModule.stateStop()
    hc.log.Printf("Exit HostContext")
}

////////////////////API
func (hc *HostContext)GetRun() bool{
    return hc.run
}

func (hc *HostContext)AttachUICmdChan(cmdChan *chan string){
    hc.UICmdChan = cmdChan
}

func (hc *HostContext)AttachUIEvtChan(uiChan *chan string){
    hc.UIEvtChan = uiChan
}

func (hc *HostContext)ReadEq(dsName string) {
    hc.dstModule.sendS13F3(1 , dsName  , 0)
    time.Sleep(200 * time.Millisecond)
    hc.dstModule.sebdS13F5(1, 4096)
    time.Sleep(200 * time.Millisecond)
    hc.dstModule.sendS13F7(1)
    time.Sleep(200 * time.Millisecond)
}

func (hc *HostContext)WriteEq(dsName string) {
}
