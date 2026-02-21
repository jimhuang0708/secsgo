package secs

import (
    "encoding/json"
    "time"
    "secs/logger"
    sm "secs/secs_message"
)

type ACKC6 byte
const(
    ACKC6_OK ACKC6 = iota
    ACKC6_ERROR
)

type ACKC10 byte
const (
    ACKC10_DISPLAY ACKC10 = iota
    ACKC10_NOT_DISPLAY
    ACKC10_TERMINAL_NOT_AVAILABLE
)

type HOSTMODULE struct{
    BaseModule
    timer_S1F13 *time.Timer
    comState string
}

func CreateHOSTMODULE( log *logger.Logger) *HOSTMODULE {
    o := HOSTMODULE{ BaseModule : CreateBaseModule(log),
                     timer_S1F13 : nil }
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
    hm.log.Printf("HOST COMMUNICATE STATE %v\n",msg)
    item , err := msg.Get()
    if(err != nil) {
    }
    node, err := item.(*sm.ListNode).Get(0)
    if(err != nil){
    }
    v := node.Values()
    if( len(v.([]byte)) == 1 && v.([]byte)[0] == 0){
        if( v.([]byte)[0] == 0) {//accept
            hm.log.Printf("HOST Enter COMMUNICATE STATE | Local initiated\n")
            hm.stopS1F13()
            return;
        }
    } else {
        hm.log.Printf("HOST S1F14 invalid format just restartS1F13 timer!\n")
        hm.restartS1F13();
    }
    return
}

func (hm * HOSTMODULE)handleS10F1(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 ||err != nil){
        hm.log.Printf("Error S10F3 format\n")
        hm.sendS9FX(msg, 7)
        return ;
    }
    tidNode , err := item.(*sm.ListNode).Get(0) //TID node ,don't care
    if( tidNode.Type() != "B" || tidNode.Size() != 1 ||err != nil){
        hm.log.Printf("Error S10F3 format\n")
        hm.sendS9FX(msg, 7)
        return ;
    }
    textNodce , err := item.(*sm.ListNode).Get(1)
    if( textNodce.Type() != "A" || textNodce.Size() > 120 || textNodce.Size() == 0  || err != nil){
        hm.log.Printf("Error S10F3 format\n")
        hm.sendS9FX(msg, 7)
        return ;
    }

    text := textNodce.Values().(string)
    hm.TellUI(text)
    hm.log.Printf("Get message from Equipment : \n %s\n",text);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 10,2, false,
                                     sm.CreateBinaryNode( byte(ACKC10_DISPLAY) ) , -1 , msg.SystemBytes() ,msg.SourceHost()),ts : time.Now().Unix()}
    hm.oChan <- act
}



func (hm *HOSTMODULE)sendS1F13_Timeout(){
    hm.log.Printf("HOST S1F13 T3 timeout\n");
    hm.restartS1F13()
    return
}

func (hm *HOSTMODULE)sendS1F13(){
    msg := sm.CreateDataMessage( 1, 13, true, sm.CreateListNode(), -1, 0 , "ALL" )
    act := Evt{ cmd : "send" , msg : msg,ts : time.Now().Unix()}
    hm.log.Printf("HOST sendS1F13()\n")
    hm.oChan <- act
    return
}


func (hm *HOSTMODULE)sendS1F14(msg *sm.DataMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 14, false,
                                   sm.CreateListNode ( sm.CreateBinaryNode( byte(COMMACK_OK) ) ,  sm.CreateListNode() ),
                                   -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    hm.oChan <- act
    return
}

func (hm *HOSTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 1 ){
        if(msg.FunctionCode() == 1){
            var node sm.ElementType
            node = sm.CreateListNode( )
            //allow attempt online
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 2, false,
                                             node , msg.SessionID() , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}

            //reject attempt online
            //act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 0, false,
            //                                 node , msg.SessionID() , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}

            hm.log.Printf("HOST do On-Line Identification\n")
            hm.oChan <- act
        }

        if(msg.FunctionCode() == 13) {
            hm.log.Printf("HOST Enter COMMUNICATE STATE | Remote initiated\n")
            // Write error will quit , so don't worry send failed
            hm.sendS1F14(msg)
            hm.stopS1F13()
            return false
        }
        if(msg.FunctionCode() == 14){
            hm.handleS1F14(msg)
            return false
        }

        if(msg.FunctionCode() == 16){
            return false
        }
        if(msg.FunctionCode() == 18){
            return false
        }


    }

    if(msg.StreamCode() == 5){
        if(msg.FunctionCode() == 1){
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 5, 2, false, sm.CreateBinaryNode(  byte(ACKC5_OK) ) , msg.SessionID() , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}
            hm.oChan <- act
        }
    }

    if(msg.StreamCode() == 6){
        if(msg.FunctionCode() == 1){
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 6, 2, false, sm.CreateBinaryNode(  byte(ACKC6_OK) ) , msg.SessionID() , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}
            hm.oChan <- act
        }
        if(msg.FunctionCode() == 11){
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 6, 12, false, sm.CreateBinaryNode( byte(ACKC6_OK) ) , msg.SessionID() , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}
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

func (hm *HOSTMODULE)stateRun(){
    defer func() {
        hm.log.Printf("Exit HOSTMODULE \n");
        hm.wg.Done()
    }()
    hm.timer_S1F13 = time.NewTimer(S1F13_Duration * time.Millisecond)
    hm.stopS1F13()

    for {
        select {
            case evt := <-hm.iChan:
                if(evt.msg != nil){
                    hm.log.Printf("Host Get : %s\n",evt.msg.(sm.HSMSMessage).ToSml());
                }
                hm.processEvt(evt)
            case <-hm.timer_S1F13.C:
                hm.log.Printf("HOST S1F13 timer fired\n")
                hm.sendS1F13()
            case cmd :=<-hm.ctrlChan:
                if(cmd == "quit"){
                    return
                }

        }
    }
    return
}
