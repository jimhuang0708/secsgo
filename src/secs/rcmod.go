package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "sync"
    "encoding/json"
)

/*
remote commnad module
*/

type RCMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    deviceID int
}

func NewRCMODULE(deviceID int) *RCMODULE {
    o := RCMODULE{
                         run : "stop",
                         iChan : make(chan Evt,10),
                         oChan : make(chan Evt,10 ) ,
                         wg : new(sync.WaitGroup),
                         deviceID : deviceID,
                  }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (rcm * RCMODULE) PutEvt(e Evt) {
    rcm.iChan <- e
}

func (rcm * RCMODULE)TellUI(text string){
    uievt := &UIEvt{ EvtType : "S2F41" , Source : "RCMODULE" , Data : text }
    jsonData, _ := json.Marshal(uievt)
    rcm.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
}


func (rcm * RCMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , rcm.deviceID ,0,msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    rcm.oChan <- act
    return
}

func (rcm * RCMODULE)handleS2F41(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 || err != nil){
        fmt.Printf("Error S2F41 format\n")
        rcm.sendS9FX(msg, 7)
        return ;
    }
    rcmdNode , err := item.(*sm.ListNode).Get(0)
    if( rcmdNode.Type() != "A" || err != nil ){
        fmt.Printf("Error S2F41 format\n")
        rcm.sendS9FX(msg, 7)
        return ;
    }
    parametersNode , err := item.(*sm.ListNode).Get(1)
    if( parametersNode.Type() != "L" || err != nil ){
        fmt.Printf("Error S2F41 format\n")
        rcm.sendS9FX(msg, 7)
        return ;
    }
    rcmd :=  rcmdNode.Values().(string)
    fmt.Printf("Get Remote command: %s\n",rcmd)
    remotecmdstr := rcmd + "( "
    for i := 0 ; i < parametersNode.Size() ; i++ {
        pNode , err := parametersNode.(*sm.ListNode).Get(i)
        if(pNode.Type() != "L" || err != nil){
            fmt.Printf("Error S2F41 format\n")
            rcm.sendS9FX(msg, 7)
            return ;
        }
        cpnameNode , err := pNode.(*sm.ListNode).Get(0)
        if(cpnameNode.Type() != "A" || err != nil){
            fmt.Printf("Error S2F41 format\n")
            rcm.sendS9FX(msg, 7)
            return ;
        }
        cpvalNode , err := pNode.(*sm.ListNode).Get(1)
        if(err != nil){
            fmt.Printf("Error S2F41 format\n")
            rcm.sendS9FX(msg, 7)
            return ;
        }
        cpname := cpnameNode.Values().(string)
        cpval := cpvalNode.Values().(string)
        remotecmdstr = remotecmdstr + cpname + " : " + cpval + " , "
        fmt.Printf("cpname : %s , cpval %s\n",cpname,cpval);
    }
    remotecmdstr = remotecmdstr + " )"
    rcm.TellUI(remotecmdstr)

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2,42, false,
                                     sm.CreateListNode(sm.CreateBinaryNode( byte(0) )  , sm.CreateListNode()) ,
                                     rcm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    rcm.oChan <- act
}

func (rcm * RCMODULE)handleS2F49(msg *sm.DataMessage){
}


func (rcm * RCMODULE)processMsg(msg *sm.DataMessage)(bool){

    if(msg.StreamCode() == 2){
        if(msg.FunctionCode() == 41){
            rcm.handleS2F41(msg)
        }
        if(msg.FunctionCode() == 49){
            rcm.handleS2F49(msg)
        }

    }

    return true
}

func (rcm * RCMODULE)processEvt(evt Evt){
    msg := evt.msg.(*sm.DataMessage)
    rcm.processMsg(msg)
}

func (rcm * RCMODULE)moduleStop(){
    rcm.run = "stop"
    rcm.iChan <- Evt{ cmd : "quit"}
    rcm.wg.Wait()
}

func (rcm * RCMODULE)stateRun(){
    defer rcm.wg.Done()
    rcm.run = "run"

    for rcm.run == "run" {
        select {
            case evt := <-rcm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                rcm.processEvt(evt)
        }
    }
    rcm.run = "stop"
    fmt.Printf("Exit RCMODULE \n");
    return
}
