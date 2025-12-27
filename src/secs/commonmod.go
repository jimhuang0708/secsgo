package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "secs/data"
    "sync"
)

type COMMONMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    deviceID int
}

func NewCOMMONMODULE(deviceID int) *COMMONMODULE {
    o := COMMONMODULE{
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

func (cm * COMMONMODULE) PutEvt(e Evt) {
    cm.iChan <- e
}

func (cm * COMMONMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , cm.deviceID , 0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    cm.oChan <- act
    return
}

func (cm * COMMONMODULE)handleS1F3(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        fmt.Printf("Error S1F3 format\n")
        cm.sendS9FX(msg, 7)
        return ;
    }
    svidLst := make( []uint32 , 0  )
    for k := 0; k < item.Size() ; k++ {
        svNode , err := item.(*sm.ListNode).Get(k);
        if(svNode.Type() != "U4" || err != nil){
            fmt.Printf("error S1F3 format\n");
            cm.sendS9FX(msg, 7)
            return;
        }
        svID := uint32(svNode.Values().([]uint64)[0]);
        svidLst = append(svidLst , svID)
    }
    rootNode := data.GetSVElementTypeLst(svidLst)
    fmt.Printf("svLst : %v\n",rootNode);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 4, false, rootNode , cm.deviceID , msg.SystemBytes() , msg.SourceHost() ) , ts : time.Now().Unix()  }
    cm.oChan <- act
}

func (cm * COMMONMODULE)handleS1F11(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        fmt.Printf("Error S1F11 format\n")
        cm.sendS9FX(msg, 7)
        return ;
    }
    svidLst := make( []uint32 , 0  )
    for k := 0; k < item.Size() ; k++ {
        svNode , err := item.(*sm.ListNode).Get(k);
        if(svNode.Type() != "U4" || err != nil){
            fmt.Printf("error S1F11 format\n");
            cm.sendS9FX(msg, 7)
            return;
        }
        svID := uint32(svNode.Values().([]uint64)[0]);
        svidLst = append(svidLst , svID)
    }
    rootNode := data.GetSVNameLst(svidLst)
    fmt.Printf("svLst : %v\n",rootNode);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(1, 12, false, rootNode , cm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix() }
    cm.oChan <- act
}

func (cm * COMMONMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 1){
        if(msg.FunctionCode() == 1){
            item , err := msg.Get()
            if(err != nil || item.Type()!= "empty" ){
                fmt.Printf("error S1F1 format\n");
                cm.sendS9FX(msg, 7)
                return true;
            }

            var node sm.ElementType
            node = sm.CreateListNode( sm.CreateASCIINode("HMITaker") ,sm.CreateASCIINode("1.0") )
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 2, false, node , cm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
            fmt.Printf("do On-Line Identification\n")
            cm.oChan <- act
        }

        if(msg.FunctionCode() == 3){
            cm.handleS1F3(msg)
        }
        if(msg.FunctionCode() == 11){
            cm.handleS1F11(msg)
        }
    }

    return true
}

func (cm * COMMONMODULE)processEvt(evt Evt){
    msg := evt.msg.(*sm.DataMessage)
    cm.processMsg(msg)
}

func (cm * COMMONMODULE)moduleStop(){
    cm.run = "stop"
    cm.iChan <- Evt{ cmd : "quit"}
    cm.wg.Wait()
}

func (cm * COMMONMODULE)stateRun(){
    defer cm.wg.Done()
    cm.run = "run"

    for cm.run == "run" {
        select {
            case evt := <-cm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                cm.processEvt(evt)
        }
    }
    cm.run = "stop"
    fmt.Printf("Exit COMMONMODULE \n");
    return
}
