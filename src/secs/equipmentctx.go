package secs

import (
    "encoding/json"
    //"log/slog"
    "net"
    "time"
    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

const T1 = 1000 //Inter char timeout (RS232)
const T2 = 3000 //Protocol timeout , ENQ - EOT (RS232)
const T3 = 45000 //Reply timeout default 45 seconds
const T4 = 10000 //Block timeout , between blocks (RS232)
const T5 = 10000 //Separation timeout
const T6 = 5000  //Control timeout
const T7 = 10000 //Not selected timeout
const T8 = 5000 //inter character timeout
const RTY = 3 //Retry
const LNKTEST_DUR = 60000//linktest.req timing
const EstablishCommunicationsTimeout = 10000 //send S1F13 and wait S1F14

type MSGMODULE interface {
    PutEvt(e Evt)
}

type Evt struct{
    cmd string
    msg any //msg type depend on cmd
    waitAlarm any //use any. prevent compile error
    ts  int64
}



type EquipmentContext struct {
    BaseContext
    log *logger.Logger
    UICmdChan chan string
    UIEvtChan chan string
    ctrlState * CTRLSTATE;
    evtModule * EVENTMODULE
    commonModule * COMMONMODULE
    ecModule * EQCONSTMODULE
    tdcModule * TDCMODULE
    alarmModule * ALARMMODULE
    terminalModule * TERMINALMODULE
    rcModule * RCMODULE
    lmtModule* LIMITMONITORMODULE
    dstModule * DSTMODULE
}


func NewEquipmentContext(deviceID int, mode string, addr string  ,eqLog *logger.Logger) *EquipmentContext {
    baseLog := eqLog.With( "module" , "EquipmentCtx" , "deviceID", deviceID)
    ec := &EquipmentContext{
        BaseContext: BaseContext{
                             oChan : make(chan Evt,10 ) ,
                             iChan : make(chan Evt,10),
                             run : false,
                             deviceID : deviceID,
                             log: baseLog,

        },
        log: baseLog,
        UICmdChan : make(chan string,10),
        UIEvtChan : make(chan string,10),
        ctrlState : NewCTRLSTATE(deviceID, baseLog.With("module", "CTRLSTATE")),
        evtModule : NewEVENTMODULE(deviceID, baseLog.With("module", "EVENTMODULE")),
        commonModule : NewCOMMONMODULE(deviceID, baseLog.With("module", "COMMONMODULE")),
        ecModule : NewEQCONSTMODULE(deviceID, baseLog.With("module", "EQCONSTMODULE")),
        tdcModule : NewTDCMODULE(deviceID, baseLog.With("module", "TDCMODULE")),
        alarmModule : NewALARMMODULE(deviceID, baseLog.With("module", "ALARMMODULE")),
        terminalModule : NewTERMINALMODULE(deviceID, baseLog.With("module", "TERMINALMODULE")),
        rcModule : NewRCMODULE(deviceID, baseLog.With("module", "RCMODULE")),
        lmtModule : NewLIMITMONITORMODULE(deviceID, baseLog.With("module", "LIMITMONITORMODULE")),
        dstModule : NewDSTMODULE(deviceID, baseLog.With("module", "DSTMODULE")),
    }
    ec.BaseContext.attacher = ec
    go ec.stateRun(mode , addr )
    return ec;
}

func (ec *EquipmentContext)GetCtrlInput()(chan Evt){
     return ec.ctrlState.iChan;
}

func (ec *EquipmentContext)trigEvent(e Evt){
    ec.evtModule.PutEvt(e)
}

func (ec *EquipmentContext)buildStopTransaction(msg *sm.DataMessage){
    errmsg := sm.CreateDataMessage( msg.StreamCode() , 0 ,false, sm.CreateEmptyElementType() , ec.deviceID , msg.SystemBytes() , msg.SourceHost()  )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    ec.oChan <- act
    return
}

func (ec *EquipmentContext)buildSXF0(msg *sm.DataMessage){
    //unrecognize FUNCTION
    if(msg.MsgType() == sm.TypeDataMessage){
        if(msg.WaitBit()){
            ec.buildStopTransaction(msg) //this could prevent remote T3 timeout
        }
    }
}

func (ec *EquipmentContext)regProcessModule(){
    /*clean route path */
    for s := 0 ; s < 255 ; s++ {
        for f := 0 ; f < 255 ; f++ {
            ec.dispatchMap[s][f] = nil
        }
    }

    if(ec.ctrlState.ctrlState == "OFFLINE"){
        //set report ack direct to evtModule ,prevent OFFLINE last report cause T3 timeout
        ec.dispatchMap[6][12] = ec.evtModule
        return
    }

    ec.dispatchMap[1][23] = ec.evtModule
    ec.dispatchMap[2][33] = ec.evtModule
    ec.dispatchMap[2][35] = ec.evtModule
    ec.dispatchMap[2][37] = ec.evtModule
    ec.dispatchMap[6][12] = ec.evtModule //report ack
    ec.dispatchMap[6][15] = ec.evtModule
    ec.dispatchMap[6][19] = ec.evtModule

    ec.dispatchMap[1][1] = ec.commonModule
    ec.dispatchMap[1][3] = ec.commonModule
    ec.dispatchMap[1][11] = ec.commonModule
    ec.dispatchMap[2][17] = ec.commonModule //clock
    ec.dispatchMap[2][31] = ec.commonModule //clock

    ec.dispatchMap[2][13] = ec.ecModule
    ec.dispatchMap[2][15] = ec.ecModule
    ec.dispatchMap[2][29] = ec.ecModule

    ec.dispatchMap[2][23] = ec.tdcModule
    ec.dispatchMap[6][2] = ec.tdcModule

    ec.dispatchMap[5][2] = ec.alarmModule
    ec.dispatchMap[5][3] = ec.alarmModule
    ec.dispatchMap[5][5] = ec.alarmModule
    //TERMINAL SERVICE
    ec.dispatchMap[10][2] = ec.terminalModule
    ec.dispatchMap[10][3] = ec.terminalModule
    //RemoteControl
    ec.dispatchMap[2][41] = ec.rcModule
    ec.dispatchMap[2][49] = ec.rcModule
    //limit module
    ec.dispatchMap[2][45] = ec.lmtModule
    ec.dispatchMap[2][47] = ec.lmtModule
    //dataset transfer module
    ec.dispatchMap[13][1] = ec.dstModule
    ec.dispatchMap[13][2] = ec.dstModule
    ec.dispatchMap[13][3] = ec.dstModule
    ec.dispatchMap[13][4] = ec.dstModule
    ec.dispatchMap[13][5] = ec.dstModule
    ec.dispatchMap[13][6] = ec.dstModule
    ec.dispatchMap[13][7] = ec.dstModule
    ec.dispatchMap[13][8] = ec.dstModule

}

func (ec *EquipmentContext)processUIEvt(uievt string){
    ec.UIEvtChan <- uievt
}


func (ec *EquipmentContext)stateTrig(evt Evt){
    ec.log.Printf("evt %v",evt)
    if( evt.cmd == "recv" ) {
        ec.regProcessModule();
        if(ec.dispatchHSMSDataMsg(evt)){
            return
        }
        ec.sendUnknownError(evt.msg.(*sm.DataMessage))
    } else if(evt.cmd == "uievent"){
        if( ec.UIEvtChan != nil ){
            ec.processUIEvt(evt.msg.(string))
        }
    } else if(evt.cmd == "TRIG_EVENT" || evt.cmd == "TRIG_EVENT_FORCE"){
        ec.trigEvent(evt) //just proxy to eventMod
        return
    } else {
        if(evt.cmd == "READERROR" || evt.cmd == "T8_TIMEOUT" || evt.cmd == "WRITEERROR"){
            ec.log.Printf("Error | Event : %s",evt.cmd)
            ec.ctrlState.iChan <-Evt{ cmd : "quit" , msg : nil }
            return
        }
    }
}

func (ec *EquipmentContext )doEvt(act Evt){
    if(act.cmd == "quit"){
        ec.ctrlState.iChan <-Evt{ cmd : "quit" , msg : nil }
        return
    }

    if(act.cmd == "send" || act.cmd == "sendforce"){//proxy only
        ec.ctrlState.iChan <- act
        return
    }

    if(act.cmd == "TRIG_EVENT"){
        ec.trigEvent(act) //just proxy to eventMod
        return
    }

    if(act.cmd == "uievent"){
        if( ec.UIEvtChan != nil ){
            ec.processUIEvt(act.msg.(string))
        }
        return
    }

    ec.log.Printf("doAct %v Failed",act);
}

func (ec *EquipmentContext )stateRun(mode string,addr string){
    ec.run = true
    quit := make(chan struct{})
    go ec.Connect(mode,addr,quit)
    for ec.run {
        select {
            case act := <-ec.evtModule.oChan:
                ec.doEvt(act);
            case act := <-ec.commonModule.oChan:
                ec.doEvt(act);
            case act := <-ec.ecModule.oChan:
                ec.doEvt(act);
            case act := <-ec.tdcModule.oChan:
                ec.doEvt(act);
            case act := <-ec.alarmModule.oChan:
                ec.doEvt(act);
            case act := <-ec.terminalModule.oChan:
                ec.doEvt(act);
            case act := <-ec.rcModule.oChan:
                ec.doEvt(act);
            case act := <-ec.lmtModule.oChan:
                ec.doEvt(act);
            case act := <-ec.dstModule.oChan:
                ec.doEvt(act);
            case evt := <-ec.ctrlState.oChan:
                ec.stateTrig(evt)
            default:
                time.Sleep(100 * time.Millisecond)
        }
    }
    close(ec.UICmdChan)
    close(ec.UIEvtChan)
    close(quit)
    ec.ctrlState.StateStop()
    ec.evtModule.moduleStop()
    ec.commonModule.moduleStop()
    ec.ecModule.moduleStop()
    ec.tdcModule.moduleStop()
    ec.alarmModule.moduleStop()
    ec.terminalModule.moduleStop()
    ec.rcModule.moduleStop()
    ec.lmtModule.moduleStop()
    ec.dstModule.moduleStop()
    ec.log.Printf("Exit EquipmentContext")
}

func (ec *EquipmentContext )StateStop(){
    ec.run = false
}

////////////

func (ec *EquipmentContext)AttachSession(conn net.Conn,mode string){
    ts := NewTransport(conn, ec.log.With("component", "transport", "session_mode", mode));
    ss := NewHSMS_SS( mode , ts, ec.log.With("component", "hsms_ss", "session_mode", mode));
    /* communicate state will attach to ctrlstate */
    NewCOMMUNICATESTATE( ec.deviceID , data.G_STATE.DEFAULT_COMSTATE , ss, ec.ctrlState, ec.log.With("component", "communicate", "session_mode", mode));
}

func (ec *EquipmentContext)Operate_Ctrl(value int){
    if(value == 0){
        ec.ctrlState.OP_AttemptOnLine()
    }
    if(value == 1){
        ec.ctrlState.OP_OffLine()
    }
    if(value == 2){
        ec.ctrlState.OP_Local()
    }
    if(value == 3){
        ec.ctrlState.OP_Remote()
    }
}

////////////////////API
func (ec *EquipmentContext) GetRun() bool{
    return ec.run
}

func (ec *EquipmentContext)SetVidUint(vid uint32 ,v uint32){
    ec.log.Printf("SetVidUint %d : %d",vid,v);
    data.SetVidValue(vid,sm.CreateUintNode(4,v))
}

/* TODO : limit id should be fixed in config and can't not dynamically add by host*/
func (ec *EquipmentContext)SetVidLimit(vid uint32 ,limitid uint32,upperDB uint32,lowerDB uint32){
    ec.log.Printf("SetVidLimit vid : %d | limitid : %d | upperdb : %d | lowerdb : %d",vid,limitid,upperDB,lowerDB);
    ec.lmtModule.setLimits( vid , limitid , sm.CreateUintNode(4,upperDB) , sm.CreateUintNode(4,lowerDB)  )
}

func (ec *EquipmentContext)SetCommunicate(enable bool){
    ec.log.Printf("SetCommunicate %t",enable);
    ec.ctrlState.SetCommunicate(enable)
}

func (ec *EquipmentContext)SendText(text string){
    ec.log.Printf("SendText %s",text);
    ec.terminalModule.sendS10F1(text)
}

func (ec *EquipmentContext)SendRecognizeEvent(){
    ec.log.Printf("SendRecognizeEvent");
    ec.terminalModule.sendRecognizeEvent()
}


func (ec *EquipmentContext)SetAlarm(id uint64,v int){
    ec.alarmModule.setAlarm(id,v)
}



func (ec *EquipmentContext)SetEC(s string){
    raw := []byte(s)
    var c data.NodeValue
    json.Unmarshal( raw,&c)
    node ,_ := c.EncodeSecs();
    if(node == nil){
        node = sm.CreateEmptyElementType()
    }
    ecs := make(map[uint32]interface{} )
    evtIdLst := data.GetEvtByName( "EQ_CONST_CHANGED")
    for k := 0; k < node.Size() ; k++ {
        ecNode , err := node.(*sm.ListNode).Get(k);
        if(ecNode.Type() != "L" || ecNode.Size() != 2  || err != nil ){
            ec.log.Printf("error SetEC format");
            return;
        }
        ecIDNode , err := ecNode.(*sm.ListNode).Get(0)
        if(ecIDNode.Type() != "U4" || ecIDNode.Size() != 1  || err != nil ){
            ec.log.Printf("error SetEC format");
            return;
        }
        ecID := uint32(ecIDNode.Values().([]uint64)[0])
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
        ec.ecModule.trigEvt(evtIdLst[0],dvContext)
    }
    ret := data.SetEC(ecs)
    ec.log.Printf("ret : %v",ret);
}

func (ec *EquipmentContext)GetUIEvt()(string,bool){
    select {
        case s := <- ec.UIEvtChan :
            return s , true
        default:
            return "" , false
    }
    return "" , false
}

func (ec *EquipmentContext)PutUICmd(data string){
    ec.UICmdChan <- data
}
