package secs
import (
    "fmt"
    sm "secs/secs_message"
    "sync"
    "encoding/binary"
    "crypto/rand"
    "time"
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
    hsms_ss * HSMS_SS
    iChan chan Evt
    oChan chan Evt
    comState string
    comEnabledSubState string
    run      string
    timer_Wait_Delay *time.Timer
    wg *sync.WaitGroup
    deviceID int
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

func NewCOMMUNICATESTATE(deviceID int,comState string,hsms_ss * HSMS_SS,cs * CTRLSTATE) *COMMUNICATESTATE {
    o := COMMUNICATESTATE{
                             comState : comState,
                             comEnabledSubState : "NOTCOMMUNICATE",
                             run : "stop",
                             iChan : make(chan Evt,10),
                             oChan : make(chan Evt,10 ) ,
                             timer_Wait_Delay : nil,
                             wg : new(sync.WaitGroup),
                             deviceID : deviceID,
                             hsms_ss : hsms_ss,
                             sessionID : RandUint64String(),
                         }
    cs.attachSession(&o);
    o.wg.Add(1)
    go o.stateRun()
    return &o
}


func (cs *COMMUNICATESTATE)OP_SetComEnabled(enable bool){
    if(enable){
        if( cs.comState == "DISABLED"){
            fmt.Printf("CommunicationState change DISABLED -> ENABLED \n");
            cs.comState = "ENABLED"
            cs.comEnabledSubState = "WAIT_DELAY"
            cs.restartS1F13()
        } else {
            fmt.Printf("CommunicationState already ENABLED \n");
        }
    } else {
        if( cs.comState == "ENABLED"){
            cs.comState = "DISABLED"
            cs.comEnabledSubState = "NOTCOMMUNICATE"
            cs.stop_Wait_Delay()
            fmt.Printf("CommunicationState change to DISABLED \n");
        } else {
            fmt.Printf("CommunicationState already DISABLED \n");
        }
    }
}

func (cs *COMMUNICATESTATE)handleS1F14(msg *sm.DataMessage){
    fmt.Printf("COMMUNICATE STATE %v\n",msg)
    item , err := msg.Get()
    if(err != nil || item.Type() != "L" || item.Size() != 2 ) {
        fmt.Printf("Error S1F14 format\n")
        cs.sendS9FX(msg,7)
        return ;
    }
    node0 , err := item.(*sm.ListNode).Get(0)
    if(err != nil || node0.Type() != "B" || node0.Size() != 1){
        fmt.Printf("Error S1F14 format\n")
        cs.sendS9FX(msg,7)
        return ;
    }
    node1 , err := item.(*sm.ListNode).Get(1)
    if(err != nil || node1.Type() != "L" || node1.Size() != 0) {
        fmt.Printf("Error S1F14 format\n")
        cs.sendS9FX(msg,7)
        return ;

    }

    v := node0.Values()
    if(  v.([]byte)[0] == 0){ //accept
        fmt.Printf("Enter COMMUNICATE STATE | Local initiated\n")
        cs.comEnabledSubState = "COMMUNICATE"
        cs.stop_Wait_Delay()
        return;
    } else { //reject
        fmt.Printf("S1F14 invalid formart just restartS1F13 timer!\n")
        cs.comEnabledSubState = "WAIT_DELAY"
        cs.restartS1F13();
    }
    return
}

func (cs *COMMUNICATESTATE)handleS1F13(msg *sm.DataMessage){
    fmt.Printf("Enter COMMUNICATE STATE | Remote initiated\n")
    // Write error will quit , so don't worry send failed
    item , err := msg.Get()
    if(err != nil || item.Type() != "L" || item.Size() != 0) {
        fmt.Printf("Error S1F13 format err : %v %v %v\n",err,item.Type(),item.Size())
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
    msg := sm.CreateDataMessage( 1, 13, true,sm.CreateListNode( sm.CreateASCIINode("HMITaker") , sm.CreateASCIINode("1.0")),cs.deviceID,0, "ALL")

    alarmEvt := Evt{ cmd : "WAITS1F14_TIMEOUT" , msg : msg ,ts : time.Now().Unix() }
    wi := WaitItem {  evt : alarmEvt ,ts : time.Now().Unix() + (EstablishCommunicationsTimeout/1000) , evtChan : cs.iChan }
    act := Evt{ cmd : "send" , msg : msg,ts : time.Now().Unix() , waitAlarm : wi }
    cs.hsms_ss.iChan <- act
    return
}

func (cs *COMMUNICATESTATE)sendS1F14(msg *sm.DataMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(1, 14, false,
                               sm.CreateListNode ( sm.CreateBinaryNode( interface{}(byte(0))) ,  sm.CreateListNode( sm.CreateASCIINode("HMITaker") , sm.CreateASCIINode("1.0"))),
                               cs.deviceID , msg.SystemBytes(), msg.SourceHost() ),ts : time.Now().Unix()}
    cs.hsms_ss.iChan <- act
    return
}


func (cs *COMMUNICATESTATE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , cs.deviceID , 0 , msg.SourceHost() )
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
        fmt.Printf("COMMUNICATESTATE get disconnect notify from lower layer\n");
        cs.oChan <- evt
        cs.run = "stop"
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
        fmt.Printf("Communicate state is DISABLED |  discard anything\n")
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
     cs.run = "stop"
     cs.wg.Wait()
}

func (cs *COMMUNICATESTATE )handleInput(evt Evt){
    if(evt.cmd == "WAITS1F14_TIMEOUT"){
        fmt.Printf("Resend S1F13\n");
        cs.communicateTimeout()
        return
    }
    cs.hsms_ss.iChan <- evt
}

func (cs *COMMUNICATESTATE)stateRun(){
    defer cs.wg.Done()
    cs.run = "run"
    cs.timer_Wait_Delay = time.NewTimer(S1F13_Duration * time.Millisecond)
    cs.stop_Wait_Delay()
    for cs.run == "run" {
        select {
            case evt := <-cs.hsms_ss.oChan:
                cs.processEvt(evt)

            case evt := <-cs.iChan:
                cs.handleInput(evt)

            case <-cs.timer_Wait_Delay.C:
                fmt.Printf("S1F13 timer fired\n")
                cs.comEnabledSubState = "WAIT_CRA"
                cs.sendS1F13()
            default:
                time.Sleep(100 * time.Millisecond)
        }
    }
    cs.run = "stop"
    cs.hsms_ss.StateStop()
    fmt.Printf("Exit COMMUNICATESTATE \n");
    return
}
