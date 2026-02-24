package secs
import (
    "encoding/json"
    "reflect"
    "time"
    "secs/data"
    "secs/logger"
    "errors"
    sm "secs/secs_message"
)

type OFLACK byte
const(
    OFLACK_OK OFLACK = iota
)

type ONLACK byte
const(
    ONLACK_OK ONLACK = iota
    ONLACK_REFUSSE
    ONLACK_ALREADY
)


type CTRLSTATE struct{
    BaseModule
    session map[string]*COMMUNICATESTATE
    ctrl_fsm *CtrlFSM
}

func CreateCTRLSTATE( log *logger.Logger) *CTRLSTATE {
    o := CTRLSTATE { BaseModule : CreateBaseModule(log),
                     session : make(map[string]*COMMUNICATESTATE,100),
                     ctrl_fsm : nil }
    ctrl_fsm , _ :=  CreateCtrlFSM(CtrlConfig{ data.G_STATE.DEFAULT_CTRLMAINSTATE , data.G_STATE.DEFAULT_CTRLSUBSTATE , data.G_STATE.DEFAULT_REJECT_CTRLSUBSTATE , data.G_STATE.DEFAULT_ACCEPT_CTRLSUBSTATE} , &o )
    o.ctrl_fsm = ctrl_fsm
    o.ctrl_fsm.Emit(CtrlEvent {EvEnterControl,nil})
    o.wg.Add(1)
    go o.stateRun()
    o.TellUI()
    return &o
}

func (cs * CTRLSTATE)attachSession( s *COMMUNICATESTATE ){
    cs.session[s.sessionID] = s
}

func (cs * CTRLSTATE)TellUI(){
    uievt := &UIEvt{ EvtType : "CtrlChange" , Source : "CtrlState" , Data : cs.ctrl_fsm.state.Minor.String() + "@" + cs.ctrl_fsm.state.Major.String() }
    jsonData, _ := json.Marshal(uievt)
    cs.oChan <- Evt{ cmd : "uievent" ,ctx : string(jsonData)  }
}


func (cs * CTRLSTATE)trigEvt(e uint32,dvCtx map[uint32]interface{}){
    p := &TrigerEvtCtx{ evtid : e , dvctx : dvCtx  }
    cs.oChan <- Evt{ cmd : "TRIG_EVENT" ,ctx : p  }
    return
}

func (cs * CTRLSTATE) OnTransition(from, to ControlState, ev CtrlEvent, transitionNo int){

    switch ev.Type {
        case EvEnterControl: {
        }
        case EvHostSetOfflineRequest : {//offline - host
            cs.log.Printf("Accept host offline request  %s@%s -> %s@%s\n",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())
            cs.sendS1F16(ev.Parameter.(*sm.DataMessage))
        }
        case EvHostSetOnlineRequest : {//online 
            result := ONLACK_OK
            cs.log.Printf("Accept host online request  %s@%s -> %s@%s\n",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())
            cs.sendS1F18(result,ev.Parameter.(*sm.DataMessage))
        }
        case EvOperatorOnlineSwitch : {
            cs.log.Printf("Accept OP_AttemptOnLine |  %s@%s -> %s%s\n",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())
            cs.sendS1F1();
        }
        case EvOperatorOfflineSwitch : {
            cs.log.Printf("Accept OP_OffLine |  %s@%s -> %s@%s\n",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())
        }
        case EvOperatorSetRemote : {
            cs.log.Printf("Accept OP_Remote |  %s@%s -> %s@%s\n",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())
        }
        case EvOperatorSetLocal : {
            cs.log.Printf("Accept OP_Local |  %s@%s -> %s@%s\n",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())

        }
        case EvAttemptOnlineFailed : {
            cs.log.Printf("Rejct ATTEMPTONLINE  %s@%s -> %s@%s",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())
        }
        case EvAttemptOnlineAccepted : {
            cs.log.Printf("Accept ATTEMPTONLINE  %s@%s -> %s@%s\n",from.Minor.String(),from.Major.String(),to.Minor.String(),to.Major.String())
        }
    }
    cs.TrigCtrlCahnged(from, to)
    cs.TellUI()
}
func (cs * CTRLSTATE) OnInvalidTransition(from ControlState, ev CtrlEvent, reason error) {
    switch ev.Type {
        case EvEnterControl: {
        }
        case EvHostSetOfflineRequest : {//offline - host
            cs.log.Printf("Reject host offline request | current control state %s@%s\n",from.Minor.String(),from.Major.String())
            cs.sendS1F16(ev.Parameter.(*sm.DataMessage))
        }
        case EvHostSetOnlineRequest : {//online
            result := ONLACK_ALREADY
            cs.log.Printf("Reject host online request | current control state %s@%s\n",from.Minor.String(),from.Major.String())
            cs.sendS1F18(result,ev.Parameter.(*sm.DataMessage))
        }
        case EvOperatorOnlineSwitch : {
            cs.log.Printf("Reject OP_AttemptOnLine | current : %s@%s\n",from.Minor.String(),from.Major.String())
        }
        case EvOperatorOfflineSwitch : {
            cs.log.Printf("Reject OP_OffLine | current : %s@%s\n",from.Minor.String(),from.Major.String())
        }
        case EvOperatorSetRemote : {
            cs.log.Printf("Reject OP_Remote | current : %s@%s\n",from.Minor.String(),from.Major.String())
        }
        case EvOperatorSetLocal : {
            cs.log.Printf("Reject OP_Local | current : %s@%s\n",from.Minor.String(),from.Major.String())
        }
        case EvAttemptOnlineFailed : {
            cs.log.Printf(">handleS1F0() Keep %s@%s \n",from.Minor.String(),from.Major.String())
        }
        case EvAttemptOnlineAccepted : {
            cs.log.Printf("handleS1F2() Keep %s@%s \n",from.Minor.String(),from.Major.String())
        }
    }

}
func (cs * CTRLSTATE) OnStateInitialized(state ControlState, ev CtrlEvent, transitionNo int)  {}


/*
0 : no change,//previous control state use
1 : Offline/ Equipment Offline,
2 : Offline/Attempt Online,
3 : Offline/Host offline,
4 : Online Local,
5 : Online Remote
*/

func (cs * CTRLSTATE)stateToCode(CTRLSTATE string,ctrlSubState string)(int){
    if(CTRLSTATE == "OFFLINE"){
        if(ctrlSubState == "EQUIPMENT"){
            return 1;
        }
        if(ctrlSubState == "ATTEMPTONLINE"){
            return 2;
        }
        if(ctrlSubState == "HOST"){
            return 3;
        }
    }

    if(CTRLSTATE == "ONLINE"){

        if(ctrlSubState == "LOCAL"){
            return 4;
        }
        if(ctrlSubState == "REMOTE"){
            return 5;
        }
    }
    return 0;//unknown

}

func (cs * CTRLSTATE)codeToState(code int)(string , string){
    switch code {
    case 1:
        return "OFFLINE" , "EQUIPMENT"
    case 2:
        return "OFFLINE" , "ATTEMPTONLINE"
    case 3:
        return "OFFLINE" , "HOST"
    case 4:
        return "ONLINE" , "LOCAL"
    case 5:
        return "ONLINE" , "REMOTE"
    default:
        return "" , ""
    }
}



func (cs * CTRLSTATE)TrigCtrlCahnged(from, to ControlState){
    stateCodeNow := cs.stateToCode(from.Major.String() ,from.Minor.String())
    stateCodeWill := cs.stateToCode(to.Major.String(),to.Minor.String())
    if(stateCodeNow != stateCodeWill){
        //changed
        //fill related sv
        data.SetVidValue(3 , sm.CreateUintNode(4,stateCodeWill))
        data.SetVidValue(4 , sm.CreateUintNode(4,stateCodeNow ))
        dvContext := make(map[uint32]interface{})
        vidList := data.GetDvByName( "CURRENT_STATE_NAME")
        //Attemponline don't need issue event
        if(stateCodeWill == 1 ||  stateCodeWill == 3){
            dvContext[ vidList[0] ] = sm.CreateASCIINode("OFFLINE")
            evtids := data.GetEvtByName("CONTROL_STATE_OFFLINE")
            cs.trigEvt(evtids[0],dvContext) //offline
        } else if( stateCodeWill == 2) {

        } else if(stateCodeWill == 4){
            dvContext[ vidList[0] ] =  sm.CreateASCIINode("ONLINE_LOCAL")
            evtids := data.GetEvtByName("CONTROL_STATE_LOCAL")
            cs.trigEvt(evtids[0],dvContext) //local
        } else if(stateCodeWill == 5){
            dvContext[ vidList[0] ] =  sm.CreateASCIINode("ONLINE_REMOTE")
            evtids := data.GetEvtByName("CONTROL_STATE_REMOTE")
            cs.trigEvt(evtids[0],dvContext) //remote
       }
    }
    return
}

func (cs *CTRLSTATE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , -1 ,0,msg.SourceHost() )
    ctx := &SendCtx{ msg : errmsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    cs.session[msg.SourceHost()].iChan <- act
    return
}

func (cs * CTRLSTATE) GenericCB(err error,s *SendCtx,r * RecvCtx)(int){
    if(err != nil ){
        if(errors.Is(err,ErrTimeout)){
            cs.log.Printf("CTRLSTATE Timeout %v\n",s);
        } else {
            cs.log.Printf("CTRLSTATE Unknown Error %v\n",err);
        }
    } else {
        cs.log.Printf("CTRLSTATE get ack %v\n",r);
    }
    return 0
}

/*send by host & equipment */
func (cs * CTRLSTATE)sendS1F1(){
    rmsg := sm.CreateDataMessage( 1, 1, true, sm.CreateEmptyElementType(), -1 , 0 , "ALL" )
    ctx := &SendCtx{ msg : rmsg , cb : cs.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    evt := Evt{ cmd : "send" , ctx : ctx }
    cs.log.Printf("ask Host online Permission\n")
    for _, comm := range cs.session {
        comm.iChan<-evt
    }
}


func (cs * CTRLSTATE)handleS1F15(msg *sm.DataMessage){
    item , err := msg.Get()
    if(err != nil || item.Type()!= "empty" || item.Size() != 0 ){
        cs.log.Printf("error S1F15 format\n");
        cs.sendS9FX(msg, 7)
        return ;
    }
    cs.ctrl_fsm.Emit(CtrlEvent {EvHostSetOfflineRequest,msg})
}

func (cs * CTRLSTATE)handleS1F17(msg *sm.DataMessage){
    item , err := msg.Get()
    if(err != nil || item.Type()!= "empty" || item.Size() != 0 ){
        cs.log.Printf("error S1F17 format\n");
        cs.sendS9FX(msg , 7)
        return ;
    }
    cs.ctrl_fsm.Emit(CtrlEvent {EvHostSetOnlineRequest,msg})
}

/* send by Equipment only */
func (cs * CTRLSTATE)sendS1F16(msg *sm.DataMessage){
    rmsg := sm.CreateDataMessage( 1, 16, false, sm.CreateBinaryNode( byte(OFLACK_OK) ) , -1 , msg.SystemBytes() , msg.SourceHost() )
    ctx := &SendCtx{ msg : rmsg , cb : cs.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    cs.session[msg.SourceHost()].iChan <- act
    cs.log.Printf("do request offline\n")
    return
}

/* send by Equipment only */
func (cs * CTRLSTATE)sendS1F18(result ONLACK,msg *sm.DataMessage){
    rmsg := sm.CreateDataMessage( 1, 18, false, sm.CreateBinaryNode( byte(result) ) , -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : rmsg , cb : cs.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    cs.session[msg.SourceHost()].iChan <- act
    cs.log.Printf("do request online\n")
}


func (cs * CTRLSTATE)sendStopTransaction(msg *sm.DataMessage) {
    errmsg := sm.CreateDataMessage( msg.StreamCode() ,  0 ,false, sm.CreateEmptyElementType() , -1 , msg.SystemBytes() , msg.SourceHost() )
    ctx := &SendCtx{ msg : errmsg , cb : cs.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    cs.session[ msg.SourceHost()].iChan <- act
}

//3
func (cs * CTRLSTATE)OP_AttemptOnLine(){
    if( len(cs.session) > 0 ){
        cs.ctrl_fsm.Emit(CtrlEvent {EvOperatorOnlineSwitch,nil})
    }
}

//4
func (cs * CTRLSTATE)handleS1F0(){
    cs.ctrl_fsm.Emit(CtrlEvent {EvAttemptOnlineFailed,nil})
}

//5
func (cs * CTRLSTATE)handleS1F2(msg *sm.DataMessage){
    item , err := msg.Get()
    if(err != nil || item.Type()!= "L" || item.Size() != 0 ){
        cs.log.Printf("error S1F2 format\n");
        cs.sendS9FX(msg, 7)
        return ;
    }
    cs.ctrl_fsm.Emit(CtrlEvent {EvAttemptOnlineAccepted,msg})
}

//6 12
func (cs * CTRLSTATE)OP_OffLine(){
    cs.ctrl_fsm.Emit(CtrlEvent {EvOperatorOfflineSwitch,nil})
}

//9
func (cs * CTRLSTATE)OP_Local(){
    cs.ctrl_fsm.Emit(CtrlEvent {EvOperatorSetLocal,nil})
}

//8
func (cs * CTRLSTATE)OP_Remote(){
    cs.ctrl_fsm.Emit(CtrlEvent {EvOperatorSetRemote,nil})
}

func (cs * CTRLSTATE)Operate_Ctrl(cmd string){
    if(cmd == "ATTEMPTONLINE"){
        cs.OP_AttemptOnLine()
    }
    if(cmd == "OFFLINE"){
        cs.OP_OffLine()
    }
    if(cmd == "ONLINE_LOCAL"){
        cs.OP_Local()
    }
    if(cmd == "ONLINE_REMOTE"){
        cs.OP_Remote()
    }
}

func (cs *CTRLSTATE)processMsg(msg *sm.DataMessage)(bool){
    if( msg.StreamCode() == 1){
        if(msg.FunctionCode() == 0 ){
            cs.handleS1F0()
            return true
        }
        if(msg.FunctionCode() == 2 ){
            cs.handleS1F2(msg)
            return true
        }

        if(msg.FunctionCode() == 15){
            cs.handleS1F15(msg)
            return true
        }

        if(msg.FunctionCode() == 17 ){
            cs.handleS1F17(msg)
            return true
        }
    }
    if(cs.ctrl_fsm.state.Major.String() == "ONLINE"){
        return false //need more process
    } else {
        if(msg.WaitBit()){
            /*S1F0*/
            cs.sendStopTransaction(msg)
        }
        cs.log.Printf("checkState() failed ignore : %v | current is offline\n",msg)
        return true
    }
    return false
}

func (cs *CTRLSTATE)processEvt(evt Evt ,sessionID string){
    if(evt.cmd == "uievent"){
        cs.oChan <- evt
        return
    }
    if(evt.cmd == "COMMUNICATESTATE_EXIT"){
        cs.log.Printf("CTRLSTATE Get COMMUNICATESTATE_EXIT notify => delete session %s\n",sessionID);
        cs.session[sessionID].Stop()
        delete (cs.session,sessionID)
        if( len(cs.session) == 0 ){
            cs.log.Printf("All host leave!\n");
            //cs.updateCTRLSTATE("OFFLINE","EQUIPMENT")
        }
        return
    }

    if(evt.cmd == "recv"){
        ctx := evt.ctx.(*RecvCtx)
        dm := ctx.msg.(*sm.DataMessage)
        //cs.log.Printf("----------> got %+v from session %s\n", dm.ToSml(), sessionID)
        ctx.msg = dm.SetSourceHost(sessionID)
        msg := ctx.msg.(*sm.DataMessage)
        evt.ctx = ctx
        if(!cs.processMsg(msg)){
            cs.oChan <- evt
        }
        return
    }
}

func (cs *CTRLSTATE) IsCtrlStateChangEvt(evt Evt) (bool){
    msg := evt.ctx.(*SendCtx).msg.(*sm.DataMessage)
    if( msg.StreamCode() == 6){
        if(msg.FunctionCode() == 11 ){
            item , err := msg.Get()
            if(err != nil || item.Type()!= "L" || item.Size() != 3 ){
                return false;
            }
            evtIDNode , err := item.(*sm.ListNode).Get(1)
            if(evtIDNode.Type() != "U4" || err != nil ){
                return false
            }
            evtID := uint32(evtIDNode.Values().([]uint64)[0])
            evtTarget := data.GetEvtByName("CONTROL_STATE_OFFLINE" , "CONTROL_STATE_LOCAL" ,"CONTROL_STATE_REMOTE")


            if(evtID == evtTarget[0] || evtID == evtTarget[1] || evtID == evtTarget[2]){
                _ , codeNode , _  , _  , _ , _ := data.GetVidElementType(3)
                code := int(codeNode.(sm.ElementType).Values().([]uint64)[0])
                cs.log.Printf("IsCtrlStateChangEvt detect statecode : %d\n" , code);
                cs.TellUI()
                return true
            }
        }
    }
    return false
}

func waitAny(sessions map[string]*COMMUNICATESTATE) (Evt, string, bool) {
    cases := make([]reflect.SelectCase, 0, len(sessions))
    keys := make([]string, 0, len(sessions))
    for id, s := range sessions {
        cases = append(cases, reflect.SelectCase{
            Dir:  reflect.SelectRecv,
            Chan: reflect.ValueOf(s.oChan),
        })
        keys = append(keys, id)
    }
    cases = append(cases, reflect.SelectCase{
        Dir: reflect.SelectDefault, // this is the trick
    })
    i, v, ok := reflect.Select(cases)
    if i == len(cases)-1 { // default chosen
        return Evt{}, "", false
    }
    return v.Interface().(Evt), keys[i], ok
}

func (cs *CTRLSTATE)SetCommunicate(enable bool) {
    for _ , s := range cs.session {
        fn := func() {
            s.OP_SetComEnabled(enable)
        }
        s.iChan <- Evt{ cmd : "executefn" , ctx : fn }
    }
}

func (cs *CTRLSTATE)ProcessEvt(evt Evt){
    if(evt.cmd == "send"){
        /*
        cs.IsCtrlStateChangEvt(evt) for send ctrl change event first then change ctrl state
       */
        if(cs.IsCtrlStateChangEvt(evt) == false && cs.ctrl_fsm.state.Major.String() == "OFFLINE" ){
            cs.log.Printf("State is offline,don't send anything back\n");
            return
        }
        sourceHost := evt.ctx.(*SendCtx).msg.(*sm.DataMessage).SourceHost()
        cs.log.Printf("send back source host [%s]\n",sourceHost);
        if( sourceHost == "ALL"){
            for _, comm := range cs.session {
                comm.iChan<-evt
            }
        } else {
            cs.session[sourceHost].iChan <- evt
        }
        return
    }
    if(evt.cmd == "executefn"){
        fn := evt.ctx.(func())
        fn()
        return
    }
}

func (cs *CTRLSTATE)stateRun(){
    ticker := time.NewTicker(10 * time.Millisecond)
    defer func(){
        for _ , v := range cs.session {
            v.Stop()
        }
        cs.session = make(map[string]*COMMUNICATESTATE)
        ticker.Stop()
        cs.log.Printf("Exit CTRLSTATE\n");
        cs.wg.Done()
    }()

    for  {
        select {
            case evt := <-cs.iChan:
                cs.ProcessEvt(evt)
            case <-ticker.C:
                evt, sessionID, ok := waitAny(cs.session)
                if ok {
                    cs.processEvt(evt,sessionID)
                }
            case cmd :=<-cs.ctrlChan:
                if(cmd == "quit"){
                    return
                }
        }
    }
    return
}
