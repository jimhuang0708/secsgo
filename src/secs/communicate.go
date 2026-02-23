package secs

import (
    "crypto/rand"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "time"
    "secs/logger"
    //"errors"
    sm "secs/secs_message"
)
/* spec
  A connection transaction failure occurs when
  attempting to establish communications and is
  caused by
  — a communication failure( transaction from selected to notconnected)
  — the failure to receive an S1,F14 reply within a reply timeout limit, or
  — receipt of S1,F14 that has been improperly formatted or with COMMACK2 not set to 0.
*/
const S1F13_Duration = 1000
type COMMUNICATESTATE struct{
    BaseModule
    hsms_ss * HSMS_SS
    comfsm *ComFSM
    sessionID string
}

type COMMACK byte
const (
    COMMACK_OK COMMACK = iota
    COMMACK_DENY
)


func RandUint64String() string {
    var b [8]byte
    if _, err := rand.Read(b[:]); err != nil {
	panic(err)
    }
    n := binary.LittleEndian.Uint64(b[:])
    return fmt.Sprintf("%d", n)
}

func CreateCOMMUNICATESTATE(comState int,hsms_ss * HSMS_SS,cs * CTRLSTATE, log *logger.Logger) *COMMUNICATESTATE {
    config := CommConfig {
        SystemDefault:  MajorState(comState),
        CommDelay:     5 * time.Second,
        StrictDiscard: true,
    }

    o := COMMUNICATESTATE{   BaseModule : CreateBaseModule(log),
                             hsms_ss : hsms_ss,
                             sessionID : RandUint64String() }
    o.comfsm = CreateComFSM(config,&o)
    cs.attachSession(&o);
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (cs * COMMUNICATESTATE)TellUI(){ //notify UI comstate changed
    uievt := &UIEvt{ EvtType : "CommunicateChange" , Source : "ComState" , Data : cs.comfsm.major.String() }
    jsonData, _ := json.Marshal(uievt)
    cs.oChan <- Evt{ cmd : "uievent" ,ctx : string(jsonData)  }
}

func (cs *COMMUNICATESTATE)OP_SetComEnabled(enable bool){
    if(enable){
        cs.comfsm.Emit(CommFSMEvent{EvOperatorEnable , nil})
    } else {
        cs.comfsm.Emit(CommFSMEvent{EvOperatorDisable , nil})
    }
}

func (cs *COMMUNICATESTATE)handleS1F14(msg *sm.DataMessage){
    cs.log.Printf("COMMUNICATE STATE %v\n",msg)
    item , err := msg.Get()
    if(err != nil || item.Type() != "L" || item.Size() != 2 ) {
        cs.log.Printf("Error S1F14 format\n")
        cs.sendS9FX(msg,7)
        return ;
    }
    node0 , err := item.(*sm.ListNode).Get(0)
    if(err != nil || node0.Type() != "B" || node0.Size() != 1){
        cs.log.Printf("Error S1F14 format\n")
        cs.sendS9FX(msg,7)
        return ;
    }
    node1 , err := item.(*sm.ListNode).Get(1)
    if(err != nil || node1.Type() != "L" || node1.Size() != 0) {
        cs.log.Printf("Error S1F14 format\n")
        cs.sendS9FX(msg,7)
        return ;

    }

    v := node0.Values()
    if( COMMACK(v.([]byte)[0]) == COMMACK_OK){ //accept
        cs.log.Printf("Enter COMMUNICATE STATE | Local initiated\n")
        cs.comfsm.Emit(CommFSMEvent{EvRecvExpectedS1F14_CommAck0 , nil})
        return;
    } else { //reject
        cs.comfsm.Emit(CommFSMEvent{EvConnTransactionFail,nil})
        cs.log.Printf("S1F14 reject!\n")
    }
    return
}

func (cs *COMMUNICATESTATE)handleS1F13(msg *sm.DataMessage){
    cs.log.Printf("Enter COMMUNICATE STATE | Remote initiated\n")
    // Write error will quit , so don't worry send failed
    item , err := msg.Get()
    if(err != nil || item.Type() != "L" || item.Size() != 0) {
        cs.log.Printf("Error S1F13 format err : %v %v %v\n",err,item.Type(),item.Size())
        cs.sendS9FX(msg,7)
        return ;

    }
    cs.comfsm.Emit(CommFSMEvent{EvRecvS1F13,msg})
    return
}

func (cs *COMMUNICATESTATE)SendS1F13CB(err error,s *SendCtx,r * RecvCtx)(int){
    if(err != nil){
        fn := func (){
            //if(errors.Is(err, ErrTimeout)){
            cs.log.Printf("SendS1F13CB Timeout : Resend S1F13 \n");
            cs.comfsm.Emit(CommFSMEvent{EvConnTransactionFail,nil})
            //}
        }
        cs.iChan <- Evt{ cmd : "executefn" , ctx : fn }
    } else {
        cs.log.Printf("SendS1F13CB get ack\n");
    }
    return 0
}

func (cs *COMMUNICATESTATE)SendS1F13(){
    msg := sm.CreateDataMessage( 1, 13, true,sm.CreateListNode( sm.CreateASCIINode("HMITaker") , sm.CreateASCIINode("1.0")),-1,0, "ALL")
    ctx := &SendCtx{ msg : msg , cb : cs.SendS1F13CB , timeout : time.Now().Unix() + (EstablishCommunicationsTimeout/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    cs.hsms_ss.iChan <- act
    return
}

func (cs *COMMUNICATESTATE)SendS1F14_CommAck0(msg *sm.DataMessage){
    rmsg := sm.CreateDataMessage(1, 14, false,sm.CreateListNode ( sm.CreateBinaryNode( byte(COMMACK_OK) ) , sm.CreateListNode( sm.CreateASCIINode("HMITaker") ,
                                 sm.CreateASCIINode("1.0"))), -1 , msg.SystemBytes(), msg.SourceHost() )
    ctx := &SendCtx{ msg : rmsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    cs.hsms_ss.iChan <- act
    return
}

func (cs *COMMUNICATESTATE)DequeueAllMessagesQueuedToSend(){
    for {
        select {
            case <-cs.hsms_ss.iChan:
            // discard
            default:
                return
        }
    }

}

func (cs *COMMUNICATESTATE) DiscardInbound(reason string) {
    // nothing to do
}

func (cs *COMMUNICATESTATE) Logf(format string, args ...any) {
    cs.log.Printf(format,args)
}


func (cs *COMMUNICATESTATE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , -1 , 0 , msg.SourceHost() )
    ctx := &SendCtx{ msg : errmsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    cs.hsms_ss.iChan <- act
    return
}

func (cs *COMMUNICATESTATE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 1 ){
        if(msg.FunctionCode() == 13) {
            cs.handleS1F13(msg)
            return false
        }
        if(msg.FunctionCode() == 14){
            cs.handleS1F14(msg)
            return false
        }
    }
    ctx := &RecvCtx{ msg : msg }
    cs.oChan <- Evt{ cmd : "recv" , ctx : ctx  }
    return true
}


func (cs *COMMUNICATESTATE)processEvt(evt Evt){
    if(evt.cmd == "uievent"){
        cs.oChan <- evt
        return
    }

    if(evt.cmd == "HSMS_SS_EXIT"){
        cs.log.Printf("COMMUNICATESTATE Get HSMS_SS_EXIT\n");
        cs.comfsm.Emit(CommFSMEvent{EvLinkDisconnected,nil})
        cs.oChan <- Evt{ cmd : "COMMUNICATESTATE_EXIT" , ctx : nil }
        return
    }

    if( evt.cmd == "NOTIFY_SELECTED" ) {
        cs.log.Printf("TODO : resinitial FSM\n");
        cs.comfsm.Emit(CommFSMEvent{EvSystemInit,nil})
        return
    }

    if( evt.cmd == "recv" && cs.comfsm.major.String() == "ENABLED" ){
        msg := evt.ctx.(*RecvCtx).msg.(*sm.DataMessage)
        cs.processMsg(msg)
    } else {
        cs.log.Printf("Communicate state is DISABLED %v| discard %v\n",evt,cs.comfsm.major.String())
    }
}


func (cs *COMMUNICATESTATE)getState()(string){
    return cs.comfsm.major.String()
}


func (cs *COMMUNICATESTATE )handleInput(evt Evt){
    if(evt.cmd == "executefn"){
        fn := evt.ctx.(func())
        fn()
        return
    }
    cs.hsms_ss.iChan <- evt
}

func (cs *COMMUNICATESTATE)stateRun(){
    defer func(){
        cs.hsms_ss.Stop()
        cs.comfsm.stopTimers()
        cs.log.Printf("Exit COMMUNICATESTATE \n");
        cs.wg.Done()
    }()
    for  {
        select {
            case evt := <-cs.hsms_ss.oChan:
                cs.processEvt(evt)
            case evt := <-cs.iChan:
                cs.handleInput(evt)
            case cmd :=<-cs.ctrlChan:
                if(cmd == "quit"){
                    return
                }
            case ev := <-cs.comfsm.events:
                cs.comfsm.handle(ev)
                cs.comfsm.a.TellUI()
                cs.log.Printf("Communicate FSM Event :  %v\n",ev);
        }
    }
    return
}
