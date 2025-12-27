package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "secs/data"
    "sync"
    "strconv"
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
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    jobs  map[uint32]*TDCJOB
    deviceID int
}

func NewTDCMODULE(deviceID int) *TDCMODULE {
    o := TDCMODULE{
                         run : "stop",
                         iChan : make(chan Evt,10),
                         oChan : make(chan Evt,10 ) ,
                         wg : new(sync.WaitGroup),
                         jobs : make(map[uint32]*TDCJOB),
                         deviceID : deviceID,
                  }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (tm * TDCMODULE) PutEvt(e Evt) {
    tm.iChan <- e
}

func (tm * TDCMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , tm.deviceID ,0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    tm.oChan <- act
    return
}


func (tm * TDCMODULE)sendS6F1(samples []interface{},job * TDCJOB ){
    tridNode := sm.CreateUintNode(4,job.trid)
    lenNode := sm.CreateUintNode(4,job.sampleLen)
    timestr := time.Now().Format(time.RFC3339)
    timeNode := sm.CreateASCIINode(timestr)
    sampleNode :=  sm.CreateListNode(samples...)
    rootNode := sm.CreateListNode(tridNode,lenNode,timeNode,sampleNode)
    msg := sm.CreateDataMessage( 6, 1, true,
                                  rootNode,
                                  tm.deviceID,0, "ALL")
    act := Evt{ cmd : "send" , msg : msg,ts : time.Now().Unix() }
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
        fmt.Printf("Error S6F2 format\n")
        tm.sendS9FX(msg, 7)
    }
}

func (tm * TDCMODULE)handleS2F23(msg *sm.DataMessage){
    doErr := func(){
        fmt.Printf("Error S2F23 format\n")
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
        fmt.Printf("dsperNode only support 6 bytes format\n");
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

    trid := uint32(tridNode.Values().([]uint64)[0])
    dsper := dsperNode.Values().(string)
    totsmp := uint32(totsmpNode.Values().([]uint64)[0])
    repgsz := uint32(repgszNode.Values().([]uint64)[0])
    svidLst := make([]uint32,0)
    if( totsmp == 0 ){
        tm.removeTrace(trid)
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 24, false,
                                     sm.CreateBinaryNode( interface{}(byte(0))) ,
                                     tm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
        tm.oChan <- act
        return
    }



    for k := 0 ; k < svlstNode.Size() ; k++ {
        svNode , err := svlstNode.(*sm.ListNode).Get(k);
        if(svNode.Type() != "U4" || err != nil){
            doErr();return;
        }
        svID := uint32(svNode.Values().([]uint64)[0]);
        exist := data.IsVidExist(svID)
        fmt.Printf("exist %v %v\n",exist,svID);
        if(exist){
            svidLst = append(svidLst , svID)
        } else {
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 24, false,
                                         sm.CreateBinaryNode( interface{}(byte(4))) , //unknow vid return 4
                                         tm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix() }
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

    fmt.Printf("%v %v %v %v %v %d\n",trid,dsper,totsmp,repgsz,svidLst,second_cnt)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 24, false,
                                     sm.CreateBinaryNode( interface{}(byte(0))) ,
                                     tm.deviceID , msg.SystemBytes(),msg.SourceHost()),ts : time.Now().Unix()}
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
    msg := evt.msg.(*sm.DataMessage)
    tm.processMsg(msg)
}

func (tm * TDCMODULE)doJob(job *TDCJOB){
    if job.fireSampleTick > 0 {
        job.fireSampleTick--
    }

    if job.fireSampleTick == 0 {
        fmt.Printf("sample fired\n")
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
        fmt.Printf("fired %v \n",job.samples)
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

func (tm * TDCMODULE)moduleStop(){
    tm.run = "stop"
    tm.iChan <- Evt{ cmd : "quit"}
    tm.wg.Wait()
}

func (tm * TDCMODULE)stateRun(){
    defer tm.wg.Done()
    tm.run = "run"
    jobs_ticker := time.NewTicker(1*time.Second)
    for tm.run == "run" {
        select {
            case evt := <-tm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                tm.processEvt(evt)
            case <-jobs_ticker.C:
                tm.doJobs()
        }
    }
    tm.run = "stop"
    fmt.Printf("Exit TDCMODULE \n");
    return
}
