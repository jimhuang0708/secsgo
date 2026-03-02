package secs

import (
    "time"
    "secs/logger"
    "errors"
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
    ctx := &UIEvtCtx{ Datatype : "S10F1" , Data : text }
    hm.oChan <- Evt{ cmd : "uievent" ,ctx : ctx  }
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
    if( len(v) == 1 && v[0].(byte) == 0){
        if( v[0].(byte) == 0) {//accept
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
    /////
    out := make([]byte, len(textNodce.Values()))
    for i, v := range textNodce.Values() {
        out[i] = v.(byte)
    }
    text := string(out)
    ////
    hm.TellUI(text)
    hm.log.Printf("Get message from Equipment : \n %s\n",text);

    replyMsg :=  sm.CreateDataMessage( 10,2, false, sm.CreateBinaryNode( byte(ACKC10_DISPLAY) ) , -1 , msg.SystemBytes() ,msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    hm.oChan <- act
}

func (hm *HOSTMODULE) sendS1F13CB(err error,s *SendCtx,r * RecvCtx)(int){
    if(err != nil ){
        if(errors.Is(err,ErrTimeout)){
            hm.log.Printf("HOSTMODULE Timeout %v\n",s);
        } else {
            hm.log.Printf("HOSTMODULE Unknown Error %v\n",err);
        }
    } else {
        hm.log.Printf("HOSTMODULE get ack %v\n",r);
    }
    return 0
}

func (hm *HOSTMODULE)sendS1F13(){
    msg := sm.CreateDataMessage( 1, 13, true, sm.CreateListNode(), -1, 0 , "ALL" )
    ctx := &SendCtx{ msg : msg , cb : hm.sendS1F13CB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    hm.log.Printf("HOST sendS1F13()\n")
    hm.oChan <- act
    return
}

func (hm *HOSTMODULE)sendS1F14(msg *sm.DataMessage){
    replyMsg := sm.CreateDataMessage( 1, 14, false, sm.CreateListNode ( sm.CreateBinaryNode( byte(COMMACK_OK) ) ,  sm.CreateListNode() ), -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    hm.oChan <- act
    return
}

func (hm *HOSTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 1 ){
        if(msg.FunctionCode() == 1){
            var node sm.ElementType
            node = sm.CreateListNode( )
            //allow attempt online
            replyMsg := sm.CreateDataMessage( 1, 2, false, node , msg.SessionID() , msg.SystemBytes(),msg.SourceHost())
            //reject attempt online
            //replyMsg := sm.CreateDataMessage( 1, 0, false, node , msg.SessionID() , msg.SystemBytes(),msg.SourceHost())
            ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
            act := Evt{ cmd : "send" , ctx : ctx }
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
            replyMsg := sm.CreateDataMessage( 5, 2, false, sm.CreateBinaryNode(  byte(ACKC5_OK) ) , msg.SessionID() , msg.SystemBytes(),msg.SourceHost())
            ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
            act := Evt{ cmd : "send" , ctx : ctx }
            hm.oChan <- act
        }
    }

    if(msg.StreamCode() == 6){
        if(msg.FunctionCode() == 1){
            replyMsg := sm.CreateDataMessage( 6, 2, false, sm.CreateBinaryNode(  byte(ACKC6_OK) ) , msg.SessionID() , msg.SystemBytes(),msg.SourceHost())
            ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
            act := Evt{ cmd : "send" , ctx : ctx }
            hm.oChan <- act
        }
        if(msg.FunctionCode() == 11){
            replyMsg := sm.CreateDataMessage( 6, 12, false, sm.CreateBinaryNode( byte(ACKC6_OK) ) , msg.SessionID() , msg.SystemBytes(),msg.SourceHost())
            ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
            act := Evt{ cmd : "send" , ctx : ctx }
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
    if(evt.cmd == "recv"){
        msg := evt.ctx.(*RecvCtx).msg.(*sm.DataMessage)
        hm.processMsg(msg)
    }
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
                if(evt.ctx != nil){
                    hm.log.Printf("Host Get : %s\n",evt.ctx.(*RecvCtx).msg.(sm.HSMSMessage).ToSml());
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
