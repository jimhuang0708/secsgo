package secs

import (
    //"time"
    "secs/data"
    "secs/logger"
    "errors"
    sm "secs/secs_message"
)

type EAC byte
const(
    EAC_OK EAC = iota
    EAC_NOT_EXIST
    EAC_BUSY
    EAC_OUT_OF_RANGE
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
    p := &TrigerEvtCtx{ evtid : e , dvctx : dvCtx  }
    em.oChan <- Evt{ cmd : "TRIG_EVENT" , ctx : p  }
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
        ecID := uint32(ecNode.Values()[0].(uint64))
        ecLst = append(ecLst,ecID)
    }
    rootNode := data.GetEC(ecLst)
    em.log.Printf("rootNode : %v \n",rootNode);
    replyMsg :=  sm.CreateDataMessage( 2, 14, false,  rootNode , -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil  , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    em.oChan <- act

}


func (em * EQCONSTMODULE)SetECS(node sm.ElementType,trig bool)(error,byte){
    dvContextMap := make(map[uint32]interface{})
    ecs := make(map[uint32]sm.ElementType )
    evtIdLst := data.GetEvtByName( "EQ_CONST_CHANGED")
    for k := 0; k < node.Size() ; k++ {
        ecNode , err := node.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "L" || ecNode.Size() != 2  || err != nil ){
            return errors.New("wrong type") , 0;
        }
        ecIDNode , err := ecNode.(*sm.ListNode).Get(0)
        if(ecIDNode.Type() != "U4" || ecIDNode.Size() != 1  || err != nil ){
            return errors.New("wrong type") , 0 ;
        }
        ecID := uint32(ecIDNode.Values()[0].(uint64))
        ecValueNode , err := ecNode.(*sm.ListNode).Get(1)
        ecs[ecID] = ecValueNode

        dvContext := make(map[uint32]interface{})
        vidList := data.GetDvByName("ECID_CHANGED","EC_VALUE_CHANGED","PREVIOUS_EC_VALUE")
        dvContext[ vidList[0] ] = sm.CreateUintNode(4,ecID)
        dvContext[ vidList[1] ] = ecValueNode.Clone()
        ecIDLst := make([]uint32, 1 )
        ecIDLst[0] = ecID
        oldNodeLst := data.GetEC(ecIDLst)
        oldNode , _ := oldNodeLst.(*sm.ListNode).Get(0)
        dvContext[ vidList[2] ] = oldNode.Clone()
        //em.trigEvt(evtIdLst[0],dvContext)
        dvContextMap[ecID] = dvContext
    }
    ret := data.SetEC(ecs)
    if(trig && ret == 0){
        for _ , v := range dvContextMap {
             em.trigEvt(evtIdLst[0],v.(map[uint32]interface{}))
        }
    }
    return nil , byte(ret)
}

func (em * EQCONSTMODULE)handleS2F15(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() < 1 || err != nil){
        em.log.Printf("Error S2F15 format\n")
        em.sendS9FX(msg, 7)
        return ;
    }
    err , ret := em.SetECS(item,false) //ret is EAC code
    if(err != nil){
        em.log.Printf("Error S2F15 format\n")
        em.sendS9FX(msg, 7)
        return
    }
    em.log.Printf("ret : %v \n",ret);
    replyMsg := sm.CreateDataMessage( 2, 16, false,  sm.CreateBinaryNode( byte( ret) ) , -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil  , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    em.oChan <- act
}

func (em * EQCONSTMODULE)operatorSetECS(node sm.ElementType){
    err , ret := em.SetECS(node,true)
    em.log.Printf("operatorSetECS | ret : %d , error : %v \n",ret,err);
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
        ecID := uint32(ecNode.Values()[0].(uint64))
        ecLst = append(ecLst,ecID)
    }
    rootNode := data.GetECName(ecLst)
    em.log.Printf("rootNode : %v \n",rootNode);
    replyMsg := sm.CreateDataMessage(2, 30, false, rootNode , -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil  , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
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
    if(evt.cmd == "executefn"){
        fn := evt.ctx.(func())
        fn()
        return
    }
    if(evt.cmd == "recv"){
        msg := evt.ctx.(*RecvCtx).msg.(*sm.DataMessage)
        em.processMsg(msg)
    }
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
