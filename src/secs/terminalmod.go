package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "sync"
    "encoding/json"
)

type TERMINALMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    deviceID int
}

func NewTERMINALMODULE(deviceID int) *TERMINALMODULE {
    o := TERMINALMODULE{
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

func (tm * TERMINALMODULE) PutEvt(e Evt) {
    tm.iChan <- e
}

func (tm * TERMINALMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , tm.deviceID ,0,msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    tm.oChan <- act
    return
}

func (tm * TERMINALMODULE)sendS10F1(text string){
    tidNode := sm.CreateBinaryNode( byte(0) ) 
    txtNode := sm.CreateASCIINode(text)
    rootNode :=  sm.CreateListNode(tidNode,txtNode)
    msg := sm.CreateDataMessage(10, 1, true,
                                  rootNode,
                                  tm.deviceID,0 , "ALL")
    act := Evt{ cmd : "send" , msg : msg,ts : time.Now().Unix() }
    tm.oChan <- act
    return
}

func (tm * TERMINALMODULE)handleS10F2(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "B" || item.Size() != 1 ||err != nil){
        fmt.Printf("Error S10F2 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }
    v := item.Values().([]uint8)[0]
    fmt.Printf("S10F2 ack code : %v\n",v);

}

func (tm * TERMINALMODULE)handleS10F3(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 ||err != nil){
        fmt.Printf("Error S10F3 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }
    tidNode , err := item.(*sm.ListNode).Get(0) //TID node ,don't care
    if( tidNode.Type() != "B" || tidNode.Size() != 1 ||err != nil){
        fmt.Printf("Error S10F3 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }
    textNodce , err := item.(*sm.ListNode).Get(1)
    if( textNodce.Type() != "A" || textNodce.Size() > 120 || textNodce.Size() == 0  || err != nil){
        fmt.Printf("Error S10F3 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }

    text := textNodce.Values().(string)
    tm.TellUI(text)
    fmt.Printf("Get message from host : \n %s\n",text);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 10,4, false,
                                     sm.CreateBinaryNode( byte(0) )   ,
                                     tm.deviceID , msg.SystemBytes() ,msg.SourceHost()),ts : time.Now().Unix()}
    tm.oChan <- act
}

func (tm * TERMINALMODULE)TellUI(text string){
    uievt := &UIEvt{ EvtType : "S10F3" , Source : "TERMINALMODULE" , Data : text }
    jsonData, _ := json.Marshal(uievt)
    tm.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
}


func (tm * TERMINALMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 10){
        if(msg.FunctionCode() == 2){
            tm.handleS10F2(msg)
        }

        if(msg.FunctionCode() == 3){
            tm.handleS10F3(msg)
        }

    }


    return true
}

func (tm * TERMINALMODULE)processEvt(evt Evt){
    msg := evt.msg.(*sm.DataMessage)
    tm.processMsg(msg)
}

func (tm * TERMINALMODULE)moduleStop(){
    tm.run = "stop"
    tm.iChan <- Evt{ cmd : "quit"}
    tm.wg.Wait()
}

func (tm * TERMINALMODULE)stateRun(){
    defer tm.wg.Done()
    tm.run = "run"

    for tm.run == "run" {
        select {
            case evt := <-tm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                tm.processEvt(evt)
        }
    }
    tm.run = "stop"
    fmt.Printf("Exit TERMINALMODULE \n");
    return
}
