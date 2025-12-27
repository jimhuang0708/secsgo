package secs
import (
    "fmt"
    "time"
    "encoding/json"
    "net"
    sm "secs/secs_message"
    "secs/data"
)


type HostContext struct {
    BaseContext
    hsms_ss      *  HSMS_SS
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

func NewHostContext(deviceID int) *HostContext {
    hc := &HostContext{
                         BaseContext: BaseContext{
                             oChan : make(chan Evt,10 ) ,
                             iChan : make(chan Evt,10),
                             run : false,
                             deviceID : deviceID,
                         },
                         hostModule : NewHOSTMODULE(deviceID) ,
                         UICmdChan : nil,
                         UIEvtChan : nil,
                         hsms_ss : nil,
                     }
    go hc.stateRun()
    return hc
}

func (hc *HostContext)AttachSession(conn net.Conn,mode string){
    ts := NewTransport(conn);
    hc.hsms_ss = NewHSMS_SS(mode,ts);
}


func (hc *HostContext)sendSXFY(stream int , function int , node sm.ElementType) {
    msg := sm.CreateDataMessage( stream, function , true , node , hc.deviceID , 0 , "ALL" )
    act := Evt{ cmd : "send" , msg : msg ,ts : time.Now().Unix() }
    hc.hsms_ss.iChan <- act
    return
}

func (hc *HostContext)doUICommand(s string) {
    raw := []byte(s)
    if len(raw) > 4 {
        raw = raw[4:]
    }
    var c UICmd
    json.Unmarshal( raw,&c)
    fmt.Printf("=============>%s  | %v \n",raw,c);
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
        var hsms_oChan <-chan Evt   // 用實際型別
        if hc.hsms_ss != nil {
            hsms_oChan = hc.hsms_ss.oChan
        } else {
            hsms_oChan = nil // 明確 disable
        }
        select {
            case o := <-hsms_oChan:
                fmt.Printf("get from hsms_ss.oChan %v\n",o);
                hc.processEvt(o)
            case o := <-hc.hostModule.oChan:
                fmt.Printf("get from hc.hostModule.oChan %v\n",o);
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
    fmt.Printf("Exit HostContext\n")
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

