package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "sync"
    "encoding/json"
)

type HOSTMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run      string
    timer_S1F13 *time.Timer
    wg *sync.WaitGroup
    deviceID int
    comState string
}

func NewHOSTMODULE(deviceID int) *HOSTMODULE {
    o := HOSTMODULE{
                             run : "stop",
                             iChan : make(chan Evt,10),
                             oChan : make(chan Evt,10 ) ,
                             timer_S1F13 : nil,
                             wg : new(sync.WaitGroup),
                             deviceID : deviceID,
                         }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (hm *HOSTMODULE) PutEvt(e Evt) {
    hm.iChan <- e
}

func (hm * HOSTMODULE)TellUI(text string){
    uievt := &UIEvt{ EvtType : "S10F1" , Source : "TERMINALMODULE" , Data : text }
    jsonData, _ := json.Marshal(uievt)
    hm.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
}


func (hm *HOSTMODULE)handleS1F14(msg *sm.DataMessage){
    fmt.Printf("HOST COMMUNICATE STATE %v\n",msg)
    item , err := msg.Get()
    if(err != nil) {
    }
    node, err := item.(*sm.ListNode).Get(0)
    if(err != nil){
    }
    v := node.Values()
    if( len(v.([]byte)) == 1 && v.([]byte)[0] == 0){
        if( v.([]byte)[0] == 0) {//accept
            fmt.Printf("HOST Enter COMMUNICATE STATE | Local initiated\n")
            hm.stopS1F13()
            return;
        }
    } else {
        fmt.Printf("HOST S1F14 invalid format just restartS1F13 timer!\n")
        hm.restartS1F13();
    }
    return
}

func (hm * HOSTMODULE)handleS10F1(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 ||err != nil){
        fmt.Printf("Error S10F3 format\n")
        hm.sendS9FX(msg, 7)
        return ;
    }
    tidNode , err := item.(*sm.ListNode).Get(0) //TID node ,don't care
    if( tidNode.Type() != "B" || tidNode.Size() != 1 ||err != nil){
        fmt.Printf("Error S10F3 format\n")
        hm.sendS9FX(msg, 7)
        return ;
    }
    textNodce , err := item.(*sm.ListNode).Get(1)
    if( textNodce.Type() != "A" || textNodce.Size() > 120 || textNodce.Size() == 0  || err != nil){
        fmt.Printf("Error S10F3 format\n")
        hm.sendS9FX(msg, 7)
        return ;
    }

    text := textNodce.Values().(string)
    hm.TellUI(text)
    fmt.Printf("Get message from Equipment : \n %s\n",text);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 10,2, false,
                                     sm.CreateBinaryNode( byte(0) )   ,
                                     hm.deviceID , msg.SystemBytes() ,msg.SourceHost()),ts : time.Now().Unix()}
    hm.oChan <- act
}



func (hm *HOSTMODULE)sendS1F13_Timeout(){
    fmt.Printf("HOST S1F13 T3 timeout\n");
    hm.restartS1F13()
    return
}

func (hm *HOSTMODULE)sendS1F13(){
    msg := sm.CreateDataMessage( 1, 13, true, sm.CreateListNode(), hm.deviceID, 0 , "ALL" )
    act := Evt{ cmd : "send" , msg : msg,ts : time.Now().Unix()}
    fmt.Printf("HOST sendS1F13()\n")
    hm.oChan <- act
    return
}


func (hm *HOSTMODULE)sendS1F14(msg *sm.DataMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 14, false,
                                   sm.CreateListNode ( sm.CreateBinaryNode( interface{}(byte(0))) ,  sm.CreateListNode() ),
                                   hm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    hm.oChan <- act
    return
}

func (hm *HOSTMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , hm.deviceID ,0,msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    hm.oChan <- act
    return
}


func (hm *HOSTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 1 ){
        if(msg.FunctionCode() == 1){
            var node sm.ElementType
            node = sm.CreateListNode( )
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 2, false,
                                             node , msg.SessionID() , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}
            fmt.Printf("HOST do On-Line Identification\n")
            hm.oChan <- act
        }

        if(msg.FunctionCode() == 13) {
            fmt.Printf("HOST Enter COMMUNICATE STATE | Remote initiated\n")
            // Write error will quit , so don't worry send failed
            hm.sendS1F14(msg)
            hm.stopS1F13()
            return false
        }
        if(msg.FunctionCode() == 14){
            hm.handleS1F14(msg)
            return false
        }
    }


    if(msg.StreamCode() == 6){
        if(msg.FunctionCode() == 11){
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 6, 12, false,
                                        sm.CreateBinaryNode( []interface{}{byte(0)}... ) ,
                                        msg.SessionID() , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}
            hm.oChan <- act
        }
    }
    if(msg.StreamCode() == 10){
        if(msg.FunctionCode() == 1){
            hm.handleS10F1(msg)
            return false
        }
    }

    return true
}


func (hm *HOSTMODULE)processEvt(evt Evt){
    if(evt.cmd == "NOTIFY_SELECTED"){
        hm.restartS1F13()
        return
    }
    msg := evt.msg.(*sm.DataMessage)
    hm.processMsg(msg)
}

func (hm *HOSTMODULE)restartS1F13() {
    hm.stopS1F13()
    hm.timer_S1F13.Reset(S1F13_Duration * time.Millisecond)
}

func (hm *HOSTMODULE)stopS1F13() {
    if !hm.timer_S1F13.Stop() {
        select {
            case <-hm.timer_S1F13.C:
            default:
        }
    }
}



func (hm *HOSTMODULE )stateStop(){
     hm.run = "stop"
     hm.iChan <- Evt{ cmd : "quit"}
     hm.wg.Wait()
}


func (hm *HOSTMODULE)stateRun(){
    defer hm.wg.Done()
    hm.run = "run"
    hm.timer_S1F13 = time.NewTimer(S1F13_Duration * time.Millisecond)
    hm.stopS1F13()

    for hm.run == "run" {
        select {
            case evt := <-hm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                if(evt.msg != nil){
                    fmt.Printf("Host Get : %s\n",evt.msg.(sm.HSMSMessage).ToSml());
                }
                hm.processEvt(evt)
            case <-hm.timer_S1F13.C:
                fmt.Printf("HOST S1F13 timer fired\n")
                hm.sendS1F13()
        }
    }
    hm.run = "stop"
    fmt.Printf("Exit HOSTMODULE \n");
    return
}
