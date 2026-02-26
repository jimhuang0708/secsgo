package secs

import (
    //"sync"
    "time"
    "secs/data"
    "secs/logger"
    "errors"
    sm "secs/secs_message"
)

type ACKC5 byte
const (
    ACKC5_OK ACKC5 = iota
    ACKC5_ERR

)
type ALARMMODULE struct{
    BaseModule
}


func CreateALARMMODULE( log *logger.Logger) *ALARMMODULE {
    o := ALARMMODULE{ BaseModule : CreateBaseModule(log) }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (am * ALARMMODULE) GenericCB(err error,s *SendCtx,r * RecvCtx)(int){
    if(err != nil ){
        if(errors.Is(err,ErrTimeout)){
            am.log.Printf("ALARMMODULE Timeout %v\n",s);
        } else {
             am.log.Printf("ALARMMODULE Unknown Error %v\n",err);
        }
    } else {
        am.log.Printf("ALARMMODULE get ack %v\n",r);
    }
    return 0
}

func (am * ALARMMODULE) PutEvt(e Evt) {
    am.iChan <- e
}

func (am * ALARMMODULE)sendS5F1(id uint64){
    alids := make([]uint64,1)
    alids[0] = id
    rootNode := data.GetAlarmsLst(alids)
    node , _ := rootNode.(*sm.ListNode).Get(0)
    ctx := &SendCtx{ msg : sm.CreateDataMessage( 5, 1, true,  node , -1 , 0 , "ALL") , cb : am.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    am.log.Printf("send report\n")
    am.oChan <- act
}

func (am * ALARMMODULE)trigEvt(e uint32){
    dvCtx := make(map[uint32]interface{}) //TODO : change strcture later
    p := &TrigerEvtCtx{ evtid : e , dvctx : dvCtx  }
    am.oChan <- Evt{ cmd : "TRIG_EVENT" , ctx : p  }
    return
}

func (am * ALARMMODULE)setAlarm(id uint64,v int){
    evt , ok := data.SetAlarm(id,v)
    if(ok){
        am.sendS5F1(id)
        am.trigEvt(evt);
    }
}

func (am * ALARMMODULE)handleS5F2(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "B" || item.Size() != 1 || err != nil){
        am.log.Printf("Error S5F23 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
}

func (am * ALARMMODULE)handleS5F3(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 || err != nil){
        am.log.Printf("Error S5F3 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
    aledNode , err := item.(*sm.ListNode).Get(0);
    if(aledNode.Type() != "B" || aledNode.Size() != 1 || err != nil){
        am.log.Printf("Error S5F3 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
    alidNode , err := item.(*sm.ListNode).Get(1);
    if(alidNode.Type() != "U4" || err != nil){
        am.log.Printf("Error S5F3 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
    aled := aledNode.Values()[0].(uint8)
    alid := uint64(0xFFFFFFFFFFFFFFFF)
    if(alidNode.Size() > 0){
        alid = alidNode.Values()[0].(uint64)
    }
    ret := ACKC5(data.SetAlarmEnable(alid,aled))
    ctx := &SendCtx{ msg : sm.CreateDataMessage( 5, 4, false, sm.CreateBinaryNode( byte(ret) ) , -1 , msg.SystemBytes() , msg.SourceHost()) , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    am.oChan <- act
}

func (am * ALARMMODULE)handleS5F5(msg *sm.DataMessage){
    alidNode , err := msg.Get()
    if( alidNode.Type() != "U4" || err != nil){
        am.log.Printf("Error S5F5 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
    //
    alids := alidNode.Values()
    out := make([]uint64, len(alids))
    for i, v := range alids {
        out[i] = v.(uint64)
    }
    //
    rootNode := data.GetAlarmsLst(out)
    ctx := &SendCtx{ msg :  sm.CreateDataMessage( 5, 6, false, rootNode , -1 , msg.SystemBytes() , msg.SourceHost()) , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    am.oChan <- act
}



func (am * ALARMMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 5){
        if(msg.FunctionCode() == 2){
            am.handleS5F2(msg)
        }
        if(msg.FunctionCode() == 3){
            am.handleS5F3(msg)
        }
        if(msg.FunctionCode() == 5){
            am.handleS5F5(msg)
        }
    }
    return true
}


func (am * ALARMMODULE)processEvt(evt Evt){
    if(evt.cmd == "executefn"){
        fn := evt.ctx.(func())
        fn()
        return
    }
    if(evt.cmd == "recv"){
        am.processMsg(evt.ctx.(*RecvCtx).msg.(*sm.DataMessage))
    }
}


func (am * ALARMMODULE)stateRun(){
    defer func() {
        am.wg.Done()
        am.log.Printf("Exit ALARMMODULE \n");
    }()
    for {
        select {
            case evt := <-am.iChan:
                am.processEvt(evt)
            case cmd :=<-am.ctrlChan:
                if(cmd == "quit"){
                    return
                }
        }
    }
    return
}
