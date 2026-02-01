package secs

import (
    "crypto/rand"
    "encoding/binary"
    "encoding/json"
    "fmt"
    "time"
    "secs/logger"
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
    comState string
    comEnabledSubState string
    timer_Wait_Delay *time.Timer
    sessionID string
}

func RandUint64String() string {
    var b [8]byte
    if _, err := rand.Read(b[:]); err != nil {
	panic(err)
    }
    n := binary.LittleEndian.Uint64(b[:])
    return fmt.Sprintf("%d", n)
}

func CreateCOMMUNICATESTATE(comState string,hsms_ss * HSMS_SS,cs * CTRLSTATE, log *logger.Logger) *COMMUNICATESTATE {
    o := COMMUNICATESTATE{   BaseModule : CreateBaseModule(log),
                             comState : comState,
                             comEnabledSubState : "NOTCOMMUNICATE",
                             timer_Wait_Delay : nil,
                             hsms_ss : hsms_ss,
                             sessionID : RandUint64String() }
    cs.attachSession(&o);
    o.wg.Add(1)
    go o.stateRun()
    o.TellUI()
    return &o
}

func (cs * COMMUNICATESTATE)TellUI(){ //notify UI comstate changed
    uievt := &UIEvt{ EvtType : "CommunicateChange" , Source : "ComState" , Data : cs.comState }
    jsonData, _ := json.Marshal(uievt)
    cs.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
}

func (cs *COMMUNICATESTATE)OP_SetComEnabled(enable bool){
    if(enable){
        if( cs.comState == "DISABLED"){
            cs.log.Printf("CommunicationState change DISABLED -> ENABLED \n");
            cs.comState = "ENABLED"
            cs.comEnabledSubState = "WAIT_DELAY"
            cs.restartS1F13()
            cs.TellUI()
        } else {
            cs.log.Printf("CommunicationState already ENABLED \n");
        }
    } else {
        if( cs.comState == "ENABLED"){
            cs.comState = "DISABLED"
            cs.comEnabledSubState = "NOTCOMMUNICATE"
            cs.stop_Wait_Delay()
            cs.log.Printf("CommunicationState change to DISABLED \n");
            cs.TellUI()
        } else {
            cs.log.Printf("CommunicationState already DISABLED \n");
        }
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
    if(  v.([]byte)[0] == 0){ //accept
        cs.log.Printf("Enter COMMUNICATE STATE | Local initiated\n")
        cs.comEnabledSubState = "COMMUNICATE"
        cs.stop_Wait_Delay()
        return;
    } else { //reject
        cs.log.Printf("S1F14 invalid format just restartS1F13 timer!\n")
        cs.comEnabledSubState = "WAIT_DELAY"
        cs.restartS1F13();
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
    cs.comEnabledSubState = "COMMUNICATE"
    cs.sendS1F14(msg)
    cs.stop_Wait_Delay()
    return
}


func (cs *COMMUNICATESTATE)communicateTimeout(){
    cs.comEnabledSubState = "WAIT_DELAY"
    cs.restartS1F13()
    return
}

func (cs *COMMUNICATESTATE)sendS1F13(){
    msg := sm.CreateDataMessage( 1, 13, true,sm.CreateListNode( sm.CreateASCIINode("HMITaker") , sm.CreateASCIINode("1.0")),-1,0, "ALL")

    alarmEvt := Evt{ cmd : "WAITS1F14_TIMEOUT" , msg : msg ,ts : time.Now().Unix() }
    wi := WaitItem {  evt : alarmEvt ,ts : time.Now().Unix() + (EstablishCommunicationsTimeout/1000) , evtChan : cs.iChan }
    act := Evt{ cmd : "send" , msg : msg,ts : time.Now().Unix() , waitAlarm : wi }
    cs.hsms_ss.iChan <- act
    return
}

func (cs *COMMUNICATESTATE)sendS1F14(msg *sm.DataMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(1, 14, false,
                               sm.CreateListNode ( sm.CreateBinaryNode( byte(0) ) ,  sm.CreateListNode( sm.CreateASCIINode("HMITaker") , sm.CreateASCIINode("1.0"))),
                               -1 , msg.SystemBytes(), msg.SourceHost() ),ts : time.Now().Unix()}
    cs.hsms_ss.iChan <- act
    return
}

func (cs *COMMUNICATESTATE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , -1 , 0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
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
    cs.oChan <- Evt{ cmd : "recv" , msg : msg , ts : time.Now().Unix() }
    return true
}


func (cs *COMMUNICATESTATE)processEvt(evt Evt){
    if(evt.cmd == "uievent"){
        cs.oChan <- evt
        return
    }

    if(evt.cmd == "disconnect"){
        cs.log.Printf("COMMUNICATESTATE get disconnect notify from lower layer\n");
        cs.oChan <- evt
        cs.run = false
        return
    }


    if( cs.comState == "ENABLED" ){
        if( evt.cmd == "NOTIFY_SELECTED" ) {
            cs.comEnabledSubState = "WAIT_DELAY"
            cs.restartS1F13()
            return
        }
        msg := evt.msg.(*sm.DataMessage)
        cs.processMsg(msg)
    } else {
        cs.log.Printf("Communicate state is DISABLED |  discard anything\n")
    }
}

func (cs *COMMUNICATESTATE)restartS1F13() {
    cs.stop_Wait_Delay()
    cs.timer_Wait_Delay.Reset(S1F13_Duration * time.Millisecond)
}

func (cs *COMMUNICATESTATE)stop_Wait_Delay() {
    if !cs.timer_Wait_Delay.Stop() {
        select {
            case <-cs.timer_Wait_Delay.C:
            default:
        }
    }
}

func (cs *COMMUNICATESTATE)getState()(string){
    return cs.comState
}

func (cs *COMMUNICATESTATE )StateStop(){
     cs.run = false
     cs.wg.Wait()
}

func (cs *COMMUNICATESTATE )handleInput(evt Evt){
    if(evt.cmd == "WAITS1F14_TIMEOUT"){
        cs.log.Printf("Resend S1F13\n");
        cs.communicateTimeout()
        return
    }
    cs.hsms_ss.iChan <- evt
}

func (cs *COMMUNICATESTATE)stateRun(){
    defer cs.wg.Done()
    cs.run = true
    cs.timer_Wait_Delay = time.NewTimer(S1F13_Duration * time.Millisecond)
    cs.stop_Wait_Delay()
    for cs.run == true {
        select {
            case evt := <-cs.hsms_ss.oChan:
                cs.processEvt(evt)

            case evt := <-cs.iChan:
                cs.handleInput(evt)

            case <-cs.timer_Wait_Delay.C:
                cs.log.Printf("S1F13 timer fired\n")
                cs.comEnabledSubState = "WAIT_CRA"
                cs.sendS1F13()
            default:
                time.Sleep(100 * time.Millisecond)
        }
    }
    cs.run = false
    cs.hsms_ss.StateStop()
    cs.log.Printf("Exit COMMUNICATESTATE \n");
    return
}
