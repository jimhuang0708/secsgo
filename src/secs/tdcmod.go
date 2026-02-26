package secs

import (
    "strconv"
    "time"
    "secs/data"
    "secs/logger"
    "errors"
    sm "secs/secs_message"
)

type TIAACK byte
const(
    TIAACK_OK TIAACK = iota
    TIAACK_TOO_MANY_SVID
    TIAACK_NO_MORE_TRACE_ALLOWED
    TIAACK_INVALID_PERIOD
    TIAACK_UNKNOWN_SVID
    TIAACK_BAD_REPGSZ
)

/*
* Trace Data Collection module
*/
type TDCJOB struct{
    trid uint32
    dsper uint32
    totsmp uint32
    repgsz uint32
    svidLst []uint32
    fireSampleTick uint32
    fireReportTick uint32
    sampleLen uint32
    samples []interface{}
}

type TDCMODULE struct{
    BaseModule
    jobs  map[uint32]*TDCJOB
}

func CreateTDCMODULE( log *logger.Logger) *TDCMODULE {
    o := TDCMODULE{ BaseModule : CreateBaseModule(log),
                    jobs : make(map[uint32]*TDCJOB)  }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (tm * TDCMODULE) PutEvt(e Evt) {
    tm.iChan <- e
}

func (tm * TDCMODULE) GenericCB(err error,s *SendCtx,r * RecvCtx)(int){
    if(err != nil ){
        if(errors.Is(err,ErrTimeout)){
            tm.log.Printf("TDCMODULE Timeout %v\n",s);
        } else {
            tm.log.Printf("TDCMODULE Unknown Error %v\n",err);
        }
    } else {
        tm.log.Printf("TDCMODULE get ack %v\n",r);
    }
    return 0
}

func (tm * TDCMODULE)sendS6F1(samples []interface{},job * TDCJOB ){
    tridNode := sm.CreateUintNode(4,job.trid)
    lenNode := sm.CreateUintNode(4,job.sampleLen)
    timestr := time.Now().Format(time.RFC3339)
    timeNode := sm.CreateASCIINode(timestr)
    sampleNode :=  sm.CreateListNode(samples...)
    rootNode := sm.CreateListNode(tridNode,lenNode,timeNode,sampleNode)
    msg := sm.CreateDataMessage( 6, 1, true, rootNode, -1,0, "ALL")
    ctx := &SendCtx{ msg : msg , cb : tm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    tm.oChan <- act
    return
}



func (tm * TDCMODULE)toSeconds(str string)(uint32){
    s,_ := strconv.Atoi(str[4:6])
    m,_ := strconv.Atoi(str[2:4])
    h,_ := strconv.Atoi(str[0:2])
    return uint32((h *3600) + (m *60) + s)
}

func (tm * TDCMODULE)removeTrace(trid uint32){
    delete(tm.jobs,trid)
}

func (tm * TDCMODULE)handleS6F2(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "B" ||  item.Size() != 1 || err != nil ){
        tm.log.Printf("Error S6F2 format\n")
        tm.sendS9FX(msg, 7)
    }
}

func (tm * TDCMODULE)handleS2F23(msg *sm.DataMessage){
    doErr := func(){
        tm.log.Printf("Error S2F23 format\n")
        tm.sendS9FX(msg, 7)
    }
    item , err := msg.Get()
    if( item.Type() != "L" ||  item.Size() != 5 || err != nil ){
        doErr();return;
    }
    //Set TOTSMP=0 to terminate a trace.
    tridNode , err   := item.(*sm.ListNode).Get(0)
    if( tridNode.Type() != "U4" || tridNode.Size() != 1 || err != nil ){
        doErr();return;
    }
    dsperNode , err  := item.(*sm.ListNode).Get(1)
    if( dsperNode.Type() != "A" || dsperNode.Size() != 6 || err != nil ){
        tm.log.Printf("dsperNode only support 6 bytes format\n");
        doErr();return;
    }
    totsmpNode , err := item.(*sm.ListNode).Get(2)
    if( totsmpNode.Type() != "U4" || totsmpNode.Size() != 1 || err != nil ){
        doErr();return;
    }
    repgszNode , err := item.(*sm.ListNode).Get(3)
    if( repgszNode.Type() != "U4" || repgszNode.Size() != 1 || err != nil ){
        doErr();return;
    }
    svlstNode , err  := item.(*sm.ListNode).Get(4)
    if( svlstNode.Type() != "L" || err != nil ){//size 0 means stop
        doErr();return;
    }

    trid := uint32(tridNode.Values()[0].(uint64))
    ///
    out := make([]byte, len(dsperNode.Values()))
    for i, v := range dsperNode.Values() {
        out[i] = v.(byte)
    }
    dsper := string(out)
    ////
    totsmp := uint32(totsmpNode.Values()[0].(uint64))
    repgsz := uint32(repgszNode.Values()[0].(uint64))
    svidLst := make([]uint32,0)
    if( totsmp == 0 ){
        tm.removeTrace(trid)
        replyMsg := sm.CreateDataMessage( 2, 24, false, sm.CreateBinaryNode( byte(TIAACK_OK) ) , -1 , msg.SystemBytes() , msg.SourceHost())
        ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
        act := Evt{ cmd : "send" , ctx : ctx }
        tm.oChan <- act
        return
    }



    for k := 0 ; k < svlstNode.Size() ; k++ {
        svNode , err := svlstNode.(*sm.ListNode).Get(k);
        if(svNode.Type() != "U4" || err != nil){
            doErr();return;
        }
        svID := uint32(svNode.Values()[0].(uint64));
        exist := data.IsVidExist(svID)
        tm.log.Printf("exist %v %v\n",exist,svID);
        if(exist){
            svidLst = append(svidLst , svID)
        } else {
            replyMsg :=  sm.CreateDataMessage( 2, 24, false, sm.CreateBinaryNode( byte(TIAACK_UNKNOWN_SVID) ) , -1 , msg.SystemBytes() , msg.SourceHost())
            ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
            act := Evt{ cmd : "send" , ctx : ctx }
            tm.oChan <- act
            return
        }
    }
    second_cnt :=  tm.toSeconds(dsper)
    j := &TDCJOB{ trid : trid ,  dsper : second_cnt ,  totsmp : totsmp , repgsz : repgsz , svidLst : svidLst , fireSampleTick : second_cnt , fireReportTick : repgsz ,sampleLen : 0 ,samples : make([]interface{},0) }
    tm.jobs[trid] = j
    /*
    0 - ok
    1 - too many SVIDs
    2 - no more traces allowed
    3 - invalid period
    4 - unknown SVID
    5 - bad REPGSZ
    */

    tm.log.Printf("%v %v %v %v %v %d\n",trid,dsper,totsmp,repgsz,svidLst,second_cnt)
    replyMsg := sm.CreateDataMessage( 2, 24, false, sm.CreateBinaryNode( byte(TIAACK_OK) ) , -1 , msg.SystemBytes(),msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    tm.oChan <- act
}

func (tm * TDCMODULE)processMsg(msg *sm.DataMessage)(bool){

    if(msg.StreamCode() == 2){
        if(msg.FunctionCode() == 23){
            tm.handleS2F23(msg)
        }
    }
    if(msg.StreamCode() == 6){
        if(msg.FunctionCode() == 2){
            tm.handleS6F2(msg)
        }
    }


    return true
}

func (tm * TDCMODULE)processEvt(evt Evt){
    if(evt.cmd == "recv"){
        msg := evt.ctx.(*RecvCtx).msg.(*sm.DataMessage)
        tm.processMsg(msg)
    }
}

func (tm * TDCMODULE)doJob(job *TDCJOB){
    if job.fireSampleTick > 0 {
        job.fireSampleTick--
    }

    if job.fireSampleTick == 0 {
        tm.log.Printf("sample fired\n")
        nodes := data.GetSVElementTypeLst(job.svidLst)
        for k := 0 ; k < nodes.Size() ; k++ {
            n , _  := nodes.(*sm.ListNode).Get(k)
            job.samples = append(job.samples,n)
        }
        job.sampleLen = job.sampleLen + 1
        job.fireReportTick--
        job.fireSampleTick = job.dsper

    }
    if job.fireReportTick == 0 {
        job.totsmp = job.totsmp - job.repgsz
        tm.log.Printf("fired %v \n",job.samples)
        tm.sendS6F1( job.samples ,job )
        job.samples = make([]interface{},0)
        job.fireReportTick = job.repgsz
    }


}

func (tm * TDCMODULE)doJobs(){
    for k , _ := range tm.jobs {
        tm.doJob(tm.jobs[k])
    }
    newJobs := make(map[uint32]*TDCJOB)
    for k,_ := range tm.jobs {
        if( tm.jobs[k].totsmp > 0 ){//drop <= 0 jobs
            newJobs[k] =  tm.jobs[k]
        }
    }
    tm.jobs = newJobs
}

func (tm * TDCMODULE)stateRun(){
    defer func(){
        tm.log.Printf("Exit TDCMODULE \n");
        tm.wg.Done()
    }()
    jobs_ticker := time.NewTicker(1*time.Second)
    for {
        select {
            case evt := <-tm.iChan:
                tm.processEvt(evt)
            case <-jobs_ticker.C:
                tm.doJobs()
            case cmd := <-tm.ctrlChan:
                if(cmd == "quit"){
                    return
                }


        }
    }
    return
}
