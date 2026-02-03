package secs
import (
    "encoding/json"
    "reflect"
    "time"
    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

type CTRLSTATE struct{
    BaseModule
    session map[string]*COMMUNICATESTATE
    ctrlState string
    ctrlSubState string
    ctrlRejectSubstate string
    ctrlAcceptSubstate string
}

func CreateCTRLSTATE( log *logger.Logger) *CTRLSTATE {
    o := CTRLSTATE { BaseModule : CreateBaseModule(log),
                     session : make(map[string]*COMMUNICATESTATE,100),
                     ctrlState : data.G_STATE.DEFAULT_CTRLSTATE,
                     ctrlSubState : data.G_STATE.DEFAULT_CTRLSUBSTATE,
                     ctrlRejectSubstate : data.G_STATE.DEFAULT_REJECT_CTRLSUBSTATE,
                     ctrlAcceptSubstate : data.G_STATE.DEFAULT_ACCEPT_CTRLSUBSTATE }
    o.wg.Add(1)
    go o.stateRun()
    o.TellUI()
    return &o
}

func (cs * CTRLSTATE)attachSession( s *COMMUNICATESTATE ){
    cs.session[s.sessionID] = s
}

func (cs * CTRLSTATE)TellUI(){
    uievt := &UIEvt{ EvtType : "CtrlChange" , Source : "CtrlState" , Data : cs.ctrlSubState + "@" + cs.ctrlState }
    jsonData, _ := json.Marshal(uievt)
    cs.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
}


func (cs * CTRLSTATE)trigEvt(e uint32,dvCtx map[uint32]interface{}){
    p := make(map[string]interface{})
    p["evtid"] = e
    p["dvctx"] = dvCtx
    cs.oChan <- Evt{ cmd : "TRIG_EVENT" , msg : p ,ts : time.Now().Unix()  }
    return
}

func (cs * CTRLSTATE)trigEvtForce(e uint32,dvCtx map[uint32]interface{}){
    p := make(map[string]interface{})
    p["evtid"] = e
    p["dvctx"] = dvCtx
    cs.oChan <- Evt{ cmd : "TRIG_EVENT_FORCE" , msg : p ,ts : time.Now().Unix()  }
    return
}


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
    return -1;//unknown

}

func (cs * CTRLSTATE)updateCTRLSTATE(CTRLSTATE string,ctrlSubState string){
    stateCodeNow := cs.stateToCode(cs.ctrlState,cs.ctrlSubState)
    stateCodeWill := cs.stateToCode(CTRLSTATE,ctrlSubState)
    cs.ctrlState = CTRLSTATE
    cs.ctrlSubState = ctrlSubState
    if(stateCodeNow != stateCodeWill){
        //changed
        //fill related sv
        data.SetVidValue(3 , sm.CreateUintNode(4,stateCodeWill))
        data.SetVidValue(4 , sm.CreateUintNode(4,stateCodeNow ))
        dvContext := make(map[uint32]interface{})
        vidList := data.GetDvByName( "CURRENT_STATE_NAME")
        if(stateCodeWill == 1 || stateCodeWill == 2 || stateCodeWill == 3){
            dvContext[ vidList[0] ] = sm.CreateASCIINode("OFFLINE")
            cs.trigEvtForce(302,dvContext) //offline
        } else if(stateCodeWill == 4){
            dvContext[ vidList[0] ] =  sm.CreateASCIINode("ONLINE_LOCAL")
            cs.trigEvtForce(300,dvContext) //local
        } else if(stateCodeWill == 5){
            dvContext[ vidList[0] ] =  sm.CreateASCIINode("ONLINE_REMOTE")
            cs.trigEvtForce(301,dvContext) //remote
       }
    }
    cs.TellUI()
    return
}

func (cs *CTRLSTATE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , -1 ,0,msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg, ts : time.Now().Unix() }
    cs.session[msg.SourceHost()].iChan <- act
    return
}

/*send by host & equipment */
func (cs * CTRLSTATE)sendS1F1(){
    evt := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 1, true, sm.CreateEmptyElementType(), -1 , 0 , "ALL" ),
                ts : time.Now().Unix() }
    cs.log.Printf("ask Host online Permission\n")

    for _, comm := range cs.session {
        comm.iChan<-evt
    }
}


func (cs * CTRLSTATE)handleS1F15(msg *sm.DataMessage){
    result := byte(0)
    item , err := msg.Get()
    if(err != nil || item.Type()!= "empty" || item.Size() != 0 ){
        cs.log.Printf("error S1F15 format\n");
        cs.sendS9FX(msg, 7)
        return ;
    }

    if(cs.ctrlState == "ONLINE"){
        cs.log.Printf("Accept host offline request  %s@%s -> %s@%s\n",cs.ctrlSubState,cs.ctrlState,"HOST","OFFLINE")
        cs.updateCTRLSTATE("OFFLINE","HOST")
        result = 0 //accept
    }
    //if( refusecondition ){ currently no refused .
    //    result = 1 //refuse
    //}
    if(cs.ctrlState == "OFFLINE"){
        cs.log.Printf("Reject host offline request | reason : already offline | %s@%s\n",cs.ctrlSubState,cs.ctrlState)
        result = 2 //already offline
    }
    cs.sendS1F16(result,msg)
}

func (cs * CTRLSTATE)handleS1F17(msg *sm.DataMessage){
    result := byte(0)
    item , err := msg.Get()
    if(err != nil || item.Type()!= "empty" || item.Size() != 0 ){
        cs.log.Printf("error S1F17 format\n");
        cs.sendS9FX(msg , 7)
        return ;
    }
    if(cs.ctrlState == "OFFLINE" && cs.ctrlSubState == "HOST"){
        cs.log.Printf("Accept host online request  %s@%s -> %s@%s\n",cs.ctrlSubState,cs.ctrlState,cs.ctrlAcceptSubstate,cs.ctrlState)
        cs.updateCTRLSTATE("ONLINE",cs.ctrlAcceptSubstate)
        result = 0 //accept
    }
    if(cs.ctrlState == "OFFLINE" && cs.ctrlSubState == "EQUIPMENT"){
        cs.log.Printf("Reject host online request | reason : equipment offline | %s@%s\n",cs.ctrlSubState,cs.ctrlState)
        result = 1 //refuse
    }
    if(cs.ctrlState == "ONLINE"){
        cs.log.Printf("Reject host online request | reason : already online | %s@%s\n",cs.ctrlSubState,cs.ctrlState)
        result = 2 //already online
    }

    cs.sendS1F18(result,msg)
}

/* send by Equipment only */
func (cs * CTRLSTATE)sendS1F16(result byte,msg *sm.DataMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 16, false,
                sm.CreateBinaryNode( byte(result) ) , -1 , msg.SystemBytes() , msg.SourceHost() ),ts : time.Now().Unix()}
    cs.session[msg.SourceHost()].iChan <- act
    cs.log.Printf("do request offline\n")
    return
}

/* send by Equipment only */
func (cs * CTRLSTATE)sendS1F18(result byte,msg *sm.DataMessage){
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 18, false,
                sm.CreateBinaryNode( byte(result) ) , -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    cs.session[msg.SourceHost()].iChan <- act
    cs.log.Printf("do request online\n")
}


func (cs * CTRLSTATE)sendStopTransaction(msg *sm.DataMessage) {
    errmsg := sm.CreateDataMessage( msg.StreamCode() ,
                                     0 ,false, sm.CreateEmptyElementType() , -1 , msg.SystemBytes() , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    cs.session[ msg.SourceHost()].iChan <- act
}

//3
func (cs * CTRLSTATE)OP_AttemptOnLine(){
    if(cs.ctrlState == "OFFLINE" && cs.ctrlSubState == "EQUIPMENT" && len(cs.session) > 0 ){
        cs.log.Printf("Accept OP_AttemptOnLine |  %s@%s -> %s%s\n",cs.ctrlSubState,cs.ctrlState,"ATTEMPTONLINE",cs.ctrlState)
        cs.updateCTRLSTATE(cs.ctrlState,"ATTEMPTONLINE")
        cs.sendS1F1();
    } else {
        cs.log.Printf("Reject OP_AttemptOnLine | current : %s@%s\n",cs.ctrlSubState,cs.ctrlState)
    }
}

//4
func (cs * CTRLSTATE)handleS1F0(){
    if(cs.ctrlState == "OFFLINE" && cs.ctrlSubState == "ATTEMPTONLINE"){
        cs.log.Printf("Rejct ATTEMPTONLINE  %s@%s -> %s@%s",cs.ctrlSubState,cs.ctrlState,cs.ctrlRejectSubstate,cs.ctrlState)
        cs.updateCTRLSTATE(cs.ctrlState,cs.ctrlRejectSubstate)
    } else {
        cs.log.Printf(">handleS1F0() Keep %s@%s \n",cs.ctrlState,cs.ctrlSubState)
    }
}

//5
func (cs * CTRLSTATE)handleS1F2(msg *sm.DataMessage){
    if(cs.ctrlState == "OFFLINE" && cs.ctrlSubState == "ATTEMPTONLINE"){
        item , err := msg.Get()
        if(err != nil || item.Type()!= "L" || item.Size() != 0 ){
            cs.log.Printf("error S1F2 format\n");
            cs.sendS9FX(msg, 7)
            return ;
        }
        cs.log.Printf("Accept ATTEMPTONLINE  %s@%s -> %s@%s\n",cs.ctrlSubState,cs.ctrlState,cs.ctrlAcceptSubstate,"ONLINE")
        cs.updateCTRLSTATE("ONLINE",cs.ctrlAcceptSubstate)
    } else {
        cs.log.Printf("handleS1F2() Keep %s@%s \n",cs.ctrlState,cs.ctrlSubState)
    }
}

//6 12
func (cs * CTRLSTATE)OP_OffLine(){
    if(cs.ctrlState == "ONLINE"  || ( cs.ctrlState == "OFFLINE"  && cs.ctrlSubState == "HOST") ){
        cs.log.Printf("Accept OP_OffLine |  %s@%s -> %s@%s\n",cs.ctrlSubState,cs.ctrlState,"EQUIPMENT","OFFLINE")
        cs.updateCTRLSTATE("OFFLINE","EQUIPMENT")
    } else {
        cs.log.Printf("Reject OP_OffLine | current : %s@%s\n",cs.ctrlSubState,cs.ctrlState)
    }
}

//9
func (cs * CTRLSTATE)OP_Local(){
    if(cs.ctrlState == "ONLINE"  && cs.ctrlSubState == "REMOTE"  ){
        cs.log.Printf("Accept OP_Local |  %s@%s -> %s@%s\n",cs.ctrlSubState,cs.ctrlState,"LOCAL",cs.ctrlState)
        cs.updateCTRLSTATE(cs.ctrlState,"LOCAL")
    } else {
        cs.log.Printf("Reject OP_Local | current : %s@%s\n",cs.ctrlSubState,cs.ctrlState)
    }
}

//8
func (cs * CTRLSTATE)OP_Remote(){
    if(cs.ctrlState == "ONLINE"  && cs.ctrlSubState == "LOCAL" ){
        cs.log.Printf("Accept OP_Remote |  %s@%s -> %s@%s\n",cs.ctrlSubState,cs.ctrlState,cs.ctrlSubState,cs.ctrlState)
        cs.updateCTRLSTATE(cs.ctrlState,"REMOTE")
    } else {
        cs.log.Printf("Reject OP_Remote | current : %s@%s\n",cs.ctrlSubState,cs.ctrlState)
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
    if(cs.ctrlState == "ONLINE"){
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
    if(evt.cmd == "disconnect"){
        cs.log.Printf("CTRLSTATE get disconnect notify => delete session %s\n",sessionID);
        delete (cs.session,sessionID)
        if( len(cs.session) == 0 ){
            cs.log.Printf("All host leave!\n");
            cs.updateCTRLSTATE("OFFLINE","EQUIPMENT")
        }
        return
    }

    if(evt.cmd == "recv"){
        dm := evt.msg.(*sm.DataMessage)
        //cs.log.Printf("----------> got %+v from session %s\n", dm.ToSml(), sessionID)
        evt.msg = dm.SetSourceHost(sessionID)
        msg := evt.msg.(*sm.DataMessage)
        if(!cs.processMsg(msg)){
            cs.oChan <- evt
        }
        return
    }
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

func (cs *CTRLSTATE)SetCommunicate(v bool) {
    for _ , s := range cs.session {
        s.OP_SetComEnabled(v)
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
                /*
                  after enter offline state , equipment have to send offline event
                  so use sendforce to send event
                */
                if(cs.ctrlState == "OFFLINE" && evt.cmd != "sendforce" ){
                    cs.log.Printf("State is offline,don't send anything back\n");
                    break
                }
                sourceHost := evt.msg.(*sm.DataMessage).SourceHost()
                cs.log.Printf("send back source host [%s]\n",sourceHost);
                if( sourceHost == "ALL"){
                    for _, comm := range cs.session {
                        comm.iChan<-evt
                    }
                } else {
                    cs.session[sourceHost].iChan <- evt
                }
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
