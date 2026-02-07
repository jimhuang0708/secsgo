package secs

import (
    "time"
    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

type EQCONSTMODULE struct{
    BaseModule
}

func CreateEQCONSTMODULE( log *logger.Logger) *EQCONSTMODULE {
    o := EQCONSTMODULE{ BaseModule : CreateBaseModule(log) }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (em * EQCONSTMODULE) PutEvt(e Evt) {
    em.iChan <- e
}

func (em * EQCONSTMODULE)trigEvt(e uint32,dvCtx map[uint32]interface{}){
    p := make(map[string]interface{})
    p["evtid"] = e
    p["dvctx"] = dvCtx
    em.oChan <- Evt{ cmd : "TRIG_EVENT" , msg : p ,ts : time.Now().Unix()  }
    return
}


/*
   Note : ECV formatcode should be 10, 11, 20, 21, 3(), 4(),5() 
   it can not be list
*/
func (em * EQCONSTMODULE)handleS2F13(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        em.log.Printf("Error S2F13 format\n")
        em.sendS9FX(msg, 7)
        return ;
    }
    ecLst := make([]uint32, 0 )
    for k := 0; k < item.Size() ; k++ {
        ecNode , err := item.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "U4" || ecNode.Size() != 1 || err != nil){
            em.log.Printf("error S2F13 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecID := uint32(ecNode.Values().([]uint64)[0])
        ecLst = append(ecLst,ecID)
    }
    rootNode := data.GetEC(ecLst)
    em.log.Printf("rootNode : %v \n",rootNode);
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 14, false,  rootNode  ,
                  -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    em.oChan <- act

}

func (em * EQCONSTMODULE)handleS2F15(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() < 1 || err != nil){
        em.log.Printf("Error S2F15 format\n")
        em.sendS9FX(msg, 7)
        return ;
    }
    ecs := make(map[uint32]sm.ElementType )

    for k := 0; k < item.Size() ; k++ {
        ecNode , err := item.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "L" || ecNode.Size() != 2  || err != nil ){
            em.log.Printf("error S2F15 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecIDNode , err := ecNode.(*sm.ListNode).Get(0)
        if(ecIDNode.Type() != "U4" || ecIDNode.Size() != 1  || err != nil ){
            em.log.Printf("error S2F15 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecID := uint32(ecIDNode.Values().([]uint64)[0])
        ecValueNode , err := ecNode.(*sm.ListNode).Get(1)
        ecs[ecID] = ecValueNode
    }
    ret := data.SetEC(ecs)
    em.log.Printf("ret : %v \n",ret);
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 16, false,  sm.CreateBinaryNode( byte( ret) )  ,
                  -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    em.oChan <- act

}
func (em * EQCONSTMODULE)handleS2F29(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        em.log.Printf("Error S2F29 format\n")
        em.sendS9FX(msg, 7)
        return ;
    }
    ecLst := make([]uint32, 0 )
    for k := 0; k < item.Size() ; k++ {
        ecNode , err := item.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "U4" || ecNode.Size() != 1 || err != nil){
            em.log.Printf("error S2F29 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecID := uint32(ecNode.Values().([]uint64)[0])
        ecLst = append(ecLst,ecID)
    }
    rootNode := data.GetECName(ecLst)
    em.log.Printf("rootNode : %v \n",rootNode);
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2, 30, false, rootNode  ,
                  -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    em.oChan <- act

}

func (em * EQCONSTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 2){
        if(msg.FunctionCode() == 13){
            em.handleS2F13(msg)
        }
        if(msg.FunctionCode() == 15){
            em.handleS2F15(msg)
        }
        if(msg.FunctionCode() == 29){
            em.handleS2F29(msg)
        }
    }
    return true
}

func (em * EQCONSTMODULE)processEvt(evt Evt){
    msg := evt.msg.(*sm.DataMessage)
    em.processMsg(msg)
}


func (em * EQCONSTMODULE)stateRun(){
    defer func(){
        em.log.Printf("Exit EQCONSTMODULE \n");
        em.wg.Done()
    }()

    for {
        select {
            case evt := <-em.iChan:
                em.processEvt(evt)
            case cmd :=<-em.ctrlChan:
                if(cmd == "quit"){
                    return
                }

        }
    }
    return
}
