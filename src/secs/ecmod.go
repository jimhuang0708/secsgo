package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "secs/data"
    "sync"
)

type EQCONSTMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    deviceID int
}

func NewEQCONSTMODULE(deviceID int) *EQCONSTMODULE {
    o := EQCONSTMODULE{
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

func (em * EQCONSTMODULE) PutEvt(e Evt) {
    em.iChan <- e
}

func (em * EQCONSTMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , em.deviceID , 0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    em.oChan <- act
    return
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
        fmt.Printf("Error S2F13 format\n")
        em.sendS9FX(msg, 7)
        return ;
    }
    ecLst := make([]uint32, 0 )
    for k := 0; k < item.Size() ; k++ {
        ecNode , err := item.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "U4" || ecNode.Size() != 1 || err != nil){
            fmt.Printf("error S2F13 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecID := uint32(ecNode.Values().([]uint64)[0])
        ecLst = append(ecLst,ecID)
    }
    rootNode := data.GetEC(ecLst)
    fmt.Printf("rootNode : %v \n",rootNode);
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 14, false,  rootNode  ,
                  em.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    em.oChan <- act

}

func (em * EQCONSTMODULE)handleS2F15(msg *sm.DataMessage,notify bool){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() < 1 || err != nil){
        fmt.Printf("Error S2F15 format\n")
        em.sendS9FX(msg, 7)
        return ;
    }
    ecs := make(map[uint32]interface{} )
    evtIdLst := data.GetEvtByName( "EQ_CONST_CHANGED")

    for k := 0; k < item.Size() ; k++ {
        ecNode , err := item.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "L" || ecNode.Size() != 2  || err != nil ){
            fmt.Printf("error S2F15 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecIDNode , err := ecNode.(*sm.ListNode).Get(0)
        if(ecIDNode.Type() != "U4" || ecIDNode.Size() != 1  || err != nil ){
            fmt.Printf("error S2F15 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecID := uint32(ecIDNode.Values().([]uint64)[0])
        ecValueNode , err := ecNode.(*sm.ListNode).Get(1)
        ecs[ecID] = ecValueNode
        if notify {
            ///////////////When operator change EC , equipment should notify host
            dvContext := make(map[uint32]interface{})
            vidList := data.GetDvByName("ECID_CHANGED","EC_VALUE_CHANGED","PREVIOUS_EC_VALUE")
            dvContext[ vidList[0] ] = sm.CreateUintNode(4,ecID)
            dvContext[ vidList[1] ] = ecValueNode.Clone()
            ecIDLst := make([]uint32, 1 )
            ecIDLst[0] = ecID
            oldNodeLst := data.GetEC(ecIDLst)
            oldNode , _ := oldNodeLst.(*sm.ListNode).Get(0)
            dvContext[ vidList[2] ] = oldNode.Clone()
            em.trigEvt(evtIdLst[0],dvContext)
            //////////////
        }
    }
    ret := data.SetEC(ecs)
    fmt.Printf("ret : %v \n",ret);
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 16, false,  sm.CreateBinaryNode( interface{}(byte( ret)))  ,
                  em.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    em.oChan <- act

}
func (em * EQCONSTMODULE)handleS2F29(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        fmt.Printf("Error S2F29 format\n")
        em.sendS9FX(msg, 7)
        return ;
    }
    ecLst := make([]uint32, 0 )
    for k := 0; k < item.Size() ; k++ {
        ecNode , err := item.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "U4" || ecNode.Size() != 1 || err != nil){
            fmt.Printf("error S2F29 format\n");
            em.sendS9FX(msg, 7)
            return;
        }
        ecID := uint32(ecNode.Values().([]uint64)[0])
        ecLst = append(ecLst,ecID)
    }
    rootNode := data.GetECName(ecLst)
    fmt.Printf("rootNode : %v \n",rootNode);
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2, 30, false, rootNode  ,
                  em.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    em.oChan <- act

}

func (em * EQCONSTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 2){
        if(msg.FunctionCode() == 13){
            em.handleS2F13(msg)
        }
        if(msg.FunctionCode() == 15){
            em.handleS2F15(msg,false)
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

func (em * EQCONSTMODULE)moduleStop(){
    em.run = "stop"
    em.iChan <- Evt{ cmd : "quit"}
    em.wg.Wait()
}

func (em * EQCONSTMODULE)stateRun(){
    defer em.wg.Done()
    em.run = "run"

    for em.run == "run" {
        select {
            case evt := <-em.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                em.processEvt(evt)
        }
    }
    em.run = "stop"
    fmt.Printf("Exit EQCONSTMODULE \n");
    return
}
