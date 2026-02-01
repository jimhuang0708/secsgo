package secs

import (
    "encoding/json"
    "time"
    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

type TERMINALMODULE struct{
    BaseModule
}

func CreateTERMINALMODULE( log *logger.Logger) *TERMINALMODULE {
    o := TERMINALMODULE{ BaseModule : CreateBaseModule(log) }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (tm * TERMINALMODULE) PutEvt(e Evt) {
    tm.iChan <- e
}

func (tm * TERMINALMODULE)trigEvt(e uint32,dvCtx map[uint32]interface{}){
    p := make(map[string]interface{})
    p["evtid"] = e
    p["dvctx"] = dvCtx
    tm.oChan <- Evt{ cmd : "TRIG_EVENT" , msg : p ,ts : time.Now().Unix()  }
    return
}

func (tm * TERMINALMODULE)sendS10F1(text string){
    tidNode := sm.CreateBinaryNode( byte(0) ) 
    txtNode := sm.CreateASCIINode(text)
    rootNode :=  sm.CreateListNode(tidNode,txtNode)
    msg := sm.CreateDataMessage(10, 1, true,
                                  rootNode,
                                  -1,0 , "ALL")
    act := Evt{ cmd : "send" , msg : msg,ts : time.Now().Unix() }
    tm.oChan <- act
    return
}

func (tm * TERMINALMODULE)handleS10F2(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "B" || item.Size() != 1 ||err != nil){
        tm.log.Printf("Error S10F2 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }
    v := item.Values().([]uint8)[0]
    tm.log.Printf("S10F2 ack code : %v\n",v);

}

func (tm * TERMINALMODULE)handleS10F3(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 ||err != nil){
        tm.log.Printf("Error S10F3 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }
    tidNode , err := item.(*sm.ListNode).Get(0) //TID node ,don't care
    if( tidNode.Type() != "B" || tidNode.Size() != 1 ||err != nil){
        tm.log.Printf("Error S10F3 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }
    textNodce , err := item.(*sm.ListNode).Get(1)
    if( textNodce.Type() != "A" || textNodce.Size() > 120 || textNodce.Size() == 0  || err != nil){
        tm.log.Printf("Error S10F3 format\n")
        tm.sendS9FX(msg, 7)
        return ;
    }

    text := textNodce.Values().(string)
    tm.TellUI(text)
    tm.log.Printf("Get message from host : \n %s\n",text);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 10,4, false,
                                     sm.CreateBinaryNode( byte(0) )   ,
                                     -1 , msg.SystemBytes() ,msg.SourceHost()),ts : time.Now().Unix()}
    tm.oChan <- act
}

func (tm * TERMINALMODULE)sendRecognizeEvent(){
    evtIdLst := data.GetEvtByName("MSG_RECOGNITION")
    dvContext := make(map[uint32]interface{})
    tm.trigEvt(evtIdLst[0],dvContext)
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
    tm.run = false
    tm.iChan <- Evt{ cmd : "quit"}
    tm.wg.Wait()
}

func (tm * TERMINALMODULE)stateRun(){
    defer tm.wg.Done()
    tm.run = true

    for tm.run == true {
        select {
            case evt := <-tm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                tm.processEvt(evt)
        }
    }
    tm.run = false
    tm.log.Printf("Exit TERMINALMODULE \n");
    return
}
