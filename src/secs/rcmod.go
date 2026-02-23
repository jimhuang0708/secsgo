package secs

import (
    "encoding/json"
    //"time"
    "strconv"
    "secs/logger"
    sm "secs/secs_message"
)

type HCACK byte
const (
    HCACK_OK HCACK = iota
    HCACK_INVALID_CMD
    HCACK_CAN_NOT_DO
    HCACK_PARAMETER_ERROR
    HCACK__ASYNC
    HCACK_REJECT //rejected, already in desired condition
    HCACK_INVALID_OBJECT
)

/*
remote commnad module
*/

type RCMODULE struct{
    BaseModule
}

func CreateRCMODULE( log *logger.Logger) *RCMODULE {
    o := RCMODULE{ BaseModule : CreateBaseModule(log) }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (rcm * RCMODULE) PutEvt(e Evt) {
    rcm.iChan <- e
}

func (rcm * RCMODULE)TellUI(text string){
    uievt := &UIEvt{ EvtType : "S2F41" , Source : "RCMODULE" , Data : text }
    jsonData, _ := json.Marshal(uievt)
    rcm.oChan <- Evt{ cmd : "uievent" ,ctx : string(jsonData)  }
}

func (rcm * RCMODULE)handleS2F41(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 || err != nil){
        rcm.log.Printf("Error S2F41 format\n")
        rcm.sendS9FX(msg, 7)
        return ;
    }
    rcmdNode , err := item.(*sm.ListNode).Get(0)
    if( rcmdNode.Type() != "A" || err != nil ){
        rcm.log.Printf("Error S2F41 format\n")
        rcm.sendS9FX(msg, 7)
        return ;
    }
    parametersNode , err := item.(*sm.ListNode).Get(1)
    if( parametersNode.Type() != "L" || err != nil ){
        rcm.log.Printf("Error S2F41 format\n")
        rcm.sendS9FX(msg, 7)
        return ;
    }
    rcmd :=  rcmdNode.Values().(string)
    rcm.log.Printf("Get Remote command: %s\n",rcmd)
    remotecmdstr := rcmd + "( "
    for i := 0 ; i < parametersNode.Size() ; i++ {
        pNode , err := parametersNode.(*sm.ListNode).Get(i)
        if(pNode.Type() != "L" || err != nil){
            rcm.log.Printf("Error S2F41 format\n")
            rcm.sendS9FX(msg, 7)
            return ;
        }
        cpnameNode , err := pNode.(*sm.ListNode).Get(0)
        if(cpnameNode.Type() != "A" || err != nil){
            rcm.log.Printf("Error S2F41 format\n")
            rcm.sendS9FX(msg, 7)
            return ;
        }
        cpvalNode , err := pNode.(*sm.ListNode).Get(1)
        if(err != nil){
            rcm.log.Printf("Error S2F41 format\n")
            rcm.sendS9FX(msg, 7)
            return ;
        }
        cpname := cpnameNode.Values().(string)
        /* cpval is scalar , not array */
        if( cpvalNode.Type() == "A" ){
            cpval := cpvalNode.Values().(string)
            remotecmdstr = remotecmdstr + cpname + " : " + cpval + " , "
        } else if( cpvalNode.Type() == "B" ){
            cpval := cpvalNode.Values().([]byte)
            s := ""
            for j := 0 ; j < len(cpval) ; j++ {
                s = s + strconv.FormatUint(uint64(cpval[j]), 10)
            }
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "BOOLEAN"){
            cpval := cpvalNode.Values().([]bool)
            s := strconv.FormatBool(cpval[0])
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "U1" || cpvalNode.Type() == "U2" || cpvalNode.Type() == "U4" || cpvalNode.Type() == "U8"){
            cpval := cpvalNode.Values().([]uint64)
            s := strconv.FormatUint(uint64(cpval[0]), 10)
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "I1" || cpvalNode.Type() == "I2" || cpvalNode.Type() == "I4" || cpvalNode.Type() == "I8"){
            cpval := cpvalNode.Values().([]int64)
            s := strconv.FormatUint(uint64(cpval[0]), 10)
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "F4" || cpvalNode.Type() == "F8"){
            cpval := cpvalNode.Values().([]float64)
            s := strconv.FormatFloat(cpval[0], 'f', 2, 64)
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else {
            remotecmdstr = remotecmdstr + cpname + " : notparsenow , "
        }
        //rcm.log.Printf("cpname : %s , cpval %s\n",cpname,cpval);
    }
    remotecmdstr = remotecmdstr + " )"
    rcm.TellUI(remotecmdstr)
    replyMsg := sm.CreateDataMessage(2,42, false, sm.CreateListNode(sm.CreateBinaryNode( byte(HCACK_OK) ) , sm.CreateListNode()) , -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    rcm.oChan <- act
}

func (rcm * RCMODULE)handleS2F49(msg *sm.DataMessage){
}


func (rcm * RCMODULE)processMsg(msg *sm.DataMessage)(bool){

    if(msg.StreamCode() == 2){
        if(msg.FunctionCode() == 41){
            rcm.handleS2F41(msg)
        }
        if(msg.FunctionCode() == 49){
            rcm.handleS2F49(msg)
        }

    }

    return true
}

func (rcm * RCMODULE)processEvt(evt Evt){
    if(evt.cmd == "recv"){
        msg := evt.ctx.(*RecvCtx).msg.(*sm.DataMessage)
        rcm.processMsg(msg)
    }
}

func (rcm * RCMODULE)stateRun(){
    defer func() {
        rcm.log.Printf("Exit RCMODULE \n");
        rcm.wg.Done()
    }()

    for {
        select {
            case evt := <-rcm.iChan:
                rcm.processEvt(evt)
            case cmd := <-rcm.ctrlChan:
                if(cmd == "quit"){
                    return
                }


        }
    }
    return
}
