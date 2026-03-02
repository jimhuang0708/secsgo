package secs

import (
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
    ctx := &UIEvtCtx{ Datatype : "S2F41" , Data : text}
    rcm.oChan <- Evt{ cmd : "uievent" ,ctx : ctx  }
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
    ///
    out := make([]byte, len(rcmdNode.Values()))
    for i, v := range rcmdNode.Values() {
        out[i] = v.(byte)
    }
    rcmd := string(out)
    ///
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
        ///
        out := make([]byte, len(cpnameNode.Values()))
        for i, v := range cpnameNode.Values() {
            out[i] = v.(byte)
        }
        cpname := string(out)
        ///
        /* cpval is scalar , not array */
        if( cpvalNode.Type() == "A" ){
            ////
            out := make([]byte, len(cpvalNode.Values()))
            for i, v := range cpvalNode.Values() {
                out[i] = v.(byte)
            }
            cpval := string(out)
            /////
            remotecmdstr = remotecmdstr + cpname + " : " + cpval + " , "
        } else if( cpvalNode.Type() == "B" ){
            cpval := cpvalNode.Values()
            s := ""
            for j := 0 ; j < len(cpval) ; j++ {
                s = s + strconv.FormatUint(uint64(cpval[j].(byte)), 10)
            }
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "BOOLEAN"){
            cpval := cpvalNode.Values()
            s := strconv.FormatBool(cpval[0].(bool))
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "U1" || cpvalNode.Type() == "U2" || cpvalNode.Type() == "U4" || cpvalNode.Type() == "U8"){
            cpval := cpvalNode.Values()
            s := strconv.FormatUint(uint64(cpval[0].(uint64)), 10)
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "I1" || cpvalNode.Type() == "I2" || cpvalNode.Type() == "I4" || cpvalNode.Type() == "I8"){
            cpval := cpvalNode.Values()
            s := strconv.FormatUint(uint64(cpval[0].(int64)), 10)
            remotecmdstr = remotecmdstr + cpname + " : " + s + " , "
        } else if( cpvalNode.Type() == "F4" || cpvalNode.Type() == "F8"){
            cpval := cpvalNode.Values()
            s := strconv.FormatFloat(cpval[0].(float64), 'f', 2, 64)
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
