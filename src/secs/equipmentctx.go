package secs

import (
    //"errors"
    "encoding/json"
    //"log/slog"
    "net"
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

type SendCtx struct {
    msg sm.HSMSMessage
    cb func(error,*SendCtx,*RecvCtx)(int)
    timeout int64
}

type RecvCtx struct {
    msg sm.HSMSMessage
}

type TrigerEvtCtx struct{
    evtid uint32
    dvctx map[uint32]interface{}
}

type UIEvtCtx struct {
    datatype string
    data any
}

type Evt struct{
    cmd string
    ctx any
}



type EquipmentContext struct {
    BaseContext
    log *logger.Logger
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


func CreateEquipmentContext(deviceID int, mode string, addr string  ,eqLog *logger.Logger) *EquipmentContext {
    baseLog := eqLog.With( "Context" , "EquipmentCtx" , "deviceID", deviceID ,"Session_mode", mode)
    ec := &EquipmentContext{
        BaseContext: CreateBaseContext(deviceID,baseLog),
        log: baseLog,
        ctrlState : CreateCTRLSTATE( baseLog.With("Module", "CTRLSTATE")),
        evtModule : CreateEVENTMODULE( baseLog.With("Module", "EVENTMODULE")),
        commonModule : CreateCOMMONMODULE( baseLog.With("Module", "COMMONMODULE")),
        ecModule : CreateEQCONSTMODULE( baseLog.With("Module", "EQCONSTMODULE")),
        tdcModule : CreateTDCMODULE( baseLog.With("Module", "TDCMODULE")),
        alarmModule : CreateALARMMODULE( baseLog.With("Module", "ALARMMODULE")),
        terminalModule : CreateTERMINALMODULE( baseLog.With("Module", "TERMINALMODULE")),
        rcModule : CreateRCMODULE( baseLog.With("Module", "RCMODULE")),
        lmtModule : CreateLIMITMONITORMODULE( baseLog.With("Module", "LIMITMONITORMODULE")),
        dstModule : CreateDSTMODULE( baseLog.With("Module", "DSTMODULE")),
    }
    ec.BaseContext.attacher = ec
    ec.BaseContext.msgsender = ec
    go ec.stateRun(mode , addr )
    return ec;
}

func (ec *EquipmentContext)trigEvent(e Evt){
    ec.evtModule.PutEvt(e)
}

func (ec *EquipmentContext)SendMsg(e Evt){
    ec.ctrlState.iChan <-e
}


func (ec *EquipmentContext)regProcessModule(){
    /*clean route path */
    for s := 0 ; s < 255 ; s++ {
        for f := 0 ; f < 255 ; f++ {
            ec.dispatchMap[s][f] = nil
        }
    }

    if(ec.ctrlState.ctrl_fsm.state.Major.String() == "OFFLINE"){
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
    if( ec.UIEvtChan != nil ){
        select {
            case ec.UIEvtChan <- uievt: // not full
            default:
                // full → pop oldest
                <-ec.UIEvtChan
                ec.UIEvtChan <- uievt
        }
    }
}



func (ec *EquipmentContext)stateTrig(evt Evt){
    ec.log.Printf("evt %v",evt)
    if( evt.cmd == "recv" ) {
        ec.regProcessModule();
        if(ec.dispatchHSMSDataMsg(evt)){
            return
        }
        ec.sendUnknownError(evt.ctx.(*RecvCtx).msg.(*sm.DataMessage))
    } else if(evt.cmd == "uievent"){
        if( ec.UIEvtChan != nil ){
            ec.processUIEvt(evt.ctx.(string))
        }
    } else if(evt.cmd == "TRIG_EVENT" || evt.cmd == "TRIG_EVENT_FORCE"){
        ec.trigEvent(evt) //just proxy to eventMod
        return
    } else {
        if(evt.cmd == "READERROR" || evt.cmd == "T8_TIMEOUT" || evt.cmd == "WRITEERROR"){
            ec.log.Printf("Error | Event : %s",evt.cmd)
            ec.ctrlState.iChan <-Evt{ cmd : "quit" , ctx : nil }
            return
        }
    }
}

func (ec *EquipmentContext )doEvt(act Evt){
    if(act.cmd == "quit"){
        ec.ctrlState.iChan <-Evt{ cmd : "quit" , ctx : nil }
        return
    }

    if(act.cmd == "send"){//proxy only
        ec.ctrlState.iChan <- act
        return
    }

    if(act.cmd == "TRIG_EVENT"){
        ec.trigEvent(act) //just proxy to eventMod
        return
    }

    if(act.cmd == "uievent"){
        if( ec.UIEvtChan != nil ){
            ec.processUIEvt(act.ctx.(string))
        }
        return
    }

    ec.log.Printf("doAct %v Failed",act);
}

func (ec *EquipmentContext )stateRun(mode string,addr string){
    quit := make(chan struct{})
    defer func() {
        close(quit)
        ec.wg.Wait()
        ec.ctrlState.Stop()
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
    }()

    ec.wg.Add(1)
    go ec.Connect(mode,addr,quit)
    for {
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
            case cmd :=<-ec.ctrlChan:
                if(cmd == "quit"){
                    return
                }
        }
    }
}


func (ec *EquipmentContext)AttachSession(conn net.Conn,mode string){
    ts := CreateTransport(conn, ec.log.With("Component", "transport"));
    ss := CreateHSMS_SS( mode , ts, ec.deviceID ,ec.log.With("Component", "hsms_ss"));
    /* communicate state will attach to ctrlstate */
    CreateCOMMUNICATESTATE( data.G_STATE.DEFAULT_COMSTATE , ss, ec.ctrlState, ec.log.With("Component", "communicate"));
}

////////////////////API
func (ec *EquipmentContext)SetVidUint(vid uint32 ,v uint32){
    ec.log.Printf("SetVidUint %d : %d",vid,v);
    ok := data.SetVidValue(vid,sm.CreateUintNode(4,v))
    if(ok == false){
        ec.log.Printf("SetVidUint failed\n");
    }
}

func (ec *EquipmentContext)Operate_Ctrl(op string){
    fn := func() {
        ec.ctrlState.Operate_Ctrl(op)
    }
    ec.ctrlState.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}

func (ec *EquipmentContext)SetCommunicate(enable bool){
    ec.log.Printf("SetCommunicate %t",enable);
    fn := func() {
        ec.ctrlState.SetCommunicate(enable)
    }
    ec.ctrlState.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}

/* TODO : limit id should be fixed in config and can't not dynamically add by host*/
func (ec *EquipmentContext)SetVidLimit(vid uint32 ,limitid uint32,upperDB uint32,lowerDB uint32){
    ec.log.Printf("SetVidLimit vid : %d | limitid : %d | upperdb : %d | lowerdb : %d",vid,limitid,upperDB,lowerDB);
    fn := func() {
        ec.lmtModule.setLimits( vid, limitid, sm.CreateUintNode(4, upperDB), sm.CreateUintNode(4, lowerDB) )
    };
    ec.lmtModule.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}

func (ec *EquipmentContext)SendText(text string){
    ec.log.Printf("SendText %s",text);
    fn := func() {
        ec.terminalModule.sendS10F1(text)
    }
    ec.terminalModule.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}

func (ec *EquipmentContext)SendRecognizeEvent(){
    ec.log.Printf("SendRecognizeEvent");
    fn := func() {
        ec.terminalModule.sendRecognizeEvent()
    }
    ec.terminalModule.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}


func (ec *EquipmentContext)SetAlarm(id uint64,v int){
    fn := func() {
        ec.alarmModule.setAlarm(id,v)
    }
    ec.alarmModule.iChan <- Evt{ cmd : "executefn" , ctx : fn }
}

func (ec *EquipmentContext)SetEC(s string){
    raw := []byte(s)
    var c sm.ElementWrapper
    json.Unmarshal( raw,&c)
    node := c.Element;
    if(node == nil){
        node = sm.CreateEmptyElementType()
    }
    fn := func() {
        ec.ecModule.operatorSetECS(node)
    }
    ec.ecModule.iChan <- Evt{ cmd : "executefn" , ctx : fn }
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
