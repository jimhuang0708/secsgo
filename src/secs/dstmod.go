// Data Set Transfers module(STREAM13)
package secs

import (
    "time"
//    "secs/data"
    "secs/logger"
    "os"
    "io"
    "errors"
    sm "secs/secs_message"
)

type DSTMODULE struct{
    BaseModule
}

type DSTRANSFEROBJ struct{
    handle  uint
    buffer  []byte
    dsName  string
    ckPnt   uint
    file    *os.File
    state   string
}


type ACKC13 byte

const (
	ACKC13OK ACKC13 = 0
	ACKC13RetryableError     ACKC13 = 1
	ACKC13UnknownDataSetName ACKC13 = 2
	ACKC13IllegalCheckpoint  ACKC13 = 3
	ACKC13TooManyOpenDataSet ACKC13 = 4
	ACKC13OpenTooManyTimes   ACKC13 = 5
	ACKC13NoOpenDataSet      ACKC13 = 6
	ACKC13CannotContinue     ACKC13 = 7
	ACKC13EndOfData          ACKC13 = 8
	ACKC13HandleInUse        ACKC13 = 9
	ACKC13PendingTransaction ACKC13 = 10
)

var  RECV_MAP map[uint]*DSTRANSFEROBJ
var  SEND_MAP   map[uint]*DSTRANSFEROBJ

func CreateDSTMODULE( log *logger.Logger) *DSTMODULE {
    o := DSTMODULE{ BaseModule : CreateBaseModule(log) }
    o.wg.Add(1)
    RECV_MAP = make(map[uint]*DSTRANSFEROBJ, 100) 
    SEND_MAP   = make(map[uint]*DSTRANSFEROBJ, 100) 
    go o.stateRun()
    return &o
}

func (dstm * DSTMODULE) PutEvt(e Evt) {
    dstm.iChan <- e
}

func (dstm * DSTMODULE) openSeek(dsName string,ckPnt int64)(*os.File,ACKC13){
    file, err := os.OpenFile( dsName, os.O_RDONLY, 0666)
    if(err != nil){
        dstm.log.Printf("OpenFile failed : %v\n",err)
        return nil,ACKC13UnknownDataSetName
    } else {
        dstm.log.Printf("OpenFile Success\n")
    }

    _ , err = file.Seek( ckPnt , io.SeekStart) // go to beginning
    if err != nil {
        file.Close()
        return nil , ACKC13IllegalCheckpoint
    } else {
        dstm.log.Printf("SeekFile Success\n")
    }
    return file , ACKC13OK
}

func (dstm * DSTMODULE) GenericCB(err error,s *SendCtx,r * RecvCtx)(int){
    if(err != nil ){
        if(errors.Is(err,ErrTimeout)){
            dstm.log.Printf("DSTMODULE Timeout %v\n",s);
        } else {
            dstm.log.Printf("DSTMODULE Unknown Error %v\n",err);
        }
    } else {
        dstm.log.Printf("DSTMODULE get ack %v\n",r);
    }

    return 0
}

func (dstm * DSTMODULE)sendS13F1(dsName string){
    msg :=  sm.CreateDataMessage( 13 , 1 , true , sm.CreateASCIINode(dsName) , -1 , 0 , "ALL" )
    ctx := &SendCtx{ msg : msg , cb : dstm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F1(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "A" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    /////
    out := make([]byte, len(item.Values()))
    for i, v := range item.Values() {
        out[i] = v.(byte)
    }
    dsName := string(out)
    /////
    dstm.log.Printf("dsname : %v\n",dsName);
    ack := ACKC13OK
    // TODO: check if dataset exit or permission granted
    // if(!dsexist){
    //     ack = ACKC13RetryableError
    //     ack = ACKC13UnknownDataSetName
    //
    // }

    if _, err := os.Stat(dsName); err == nil {
        // file exists
        ack = ACKC13OK
    } else {
        ack = ACKC13UnknownDataSetName
    }
    rootNode := sm.CreateListNode( sm.CreateASCIINode(dsName),sm.CreateBinaryNode( byte(ack) ) );
    replyMsg := sm.CreateDataMessage( 13, 2, false, rootNode , -1 , msg.SystemBytes() , msg.SourceHost() )
    ctx := &SendCtx{ msg : replyMsg , cb : dstm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act

    //auto allow now
    if(ack == ACKC13OK){
        dstm.sendS13F3(1 , dsName  , 0)
    }

}

func (dstm * DSTMODULE)handleS13F2(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    dsNameNode , err := item.(*sm.ListNode).Get(0)
    ////
    out := make([]byte, len(dsNameNode.Values()))
    for i, v := range dsNameNode.Values() {
        out[i] = v.(byte)
    }
    dsName := string(out)
    ////
    ackNode , err :=  item.(*sm.ListNode).Get(1)
    ack := ackNode.Values()[0].(byte)
    dstm.log.Printf("handleS13F2 : %s | ack : %d \n",dsName,ack);
    if(ACKC13(ack) != ACKC13OK){
        dstm.log.Printf("handleS13F2 error : %d\n",ack);
        return;
    }

    return
}


/* receiving sysytem */
func (dstm * DSTMODULE)sendS13F3(handle uint , dsName string , ckpnt uint){
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateASCIINode(dsName),sm.CreateUintNode( 4 , ckpnt ) );
    msg :=  sm.CreateDataMessage( 13 , 3 , true , rootNode , -1 , 0 , "ALL" )
    ctx := &SendCtx{ msg : msg , cb : dstm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

/* sending system */
func (dstm * DSTMODULE)handleS13F3(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    handleNode , err := item.(*sm.ListNode).Get(0)
    handle := handleNode.Values()[0].(uint64)
    dsNameNode , err := item.(*sm.ListNode).Get(1)
    ////
    out := make([]byte, len(dsNameNode.Values()))
    for i, v := range dsNameNode.Values() {
        out[i] = v.(byte)
    }
    dsName := string(out)
    ////
    ckPntNode  , err := item.(*sm.ListNode).Get(2)
    ckPnt := ckPntNode.Values()[0].(uint64)
    dstm.log.Printf("handle : %d | dsname : %s | ckPnt : %d\n",handle,dsName,ckPnt);
    RTYPE := 0   //0 : stream | 1 : discrete ,support stream only
    RECLEN := 0  //record length
    ack := ACKC13OK
    if _ , found := SEND_MAP[uint(handle)]; found {
        dstm.log.Printf("Send Handle Already open\n")
        ack = ACKC13HandleInUse
    } else {
        var file *os.File
        file , ack = dstm.openSeek(dsName,int64(ckPnt))
        if(ack == ACKC13OK){
            dst := &DSTRANSFEROBJ{ handle:uint(handle) , buffer : nil , dsName : dsName , ckPnt : uint(ckPnt) , file : file , state : "IDLE" }
            SEND_MAP[uint(handle)] = dst
        }
        dstm.log.Printf("Create Send Handle : %d\n",handle)
    }
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateASCIINode(dsName),sm.CreateBinaryNode( byte(ack) ) , sm.CreateUintNode(1,RTYPE) , sm.CreateUintNode(4,RECLEN) );
    replyMsg := sm.CreateDataMessage( 13, 4, false, rootNode , -1 , msg.SystemBytes() , msg.SourceHost() )
    ctx := &SendCtx{ msg : replyMsg , cb : dstm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F4(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    handleNode , err := item.(*sm.ListNode).Get(0)
    handle := handleNode.Values()[0].(uint64)
    dsNameNode , err := item.(*sm.ListNode).Get(1)
    ////
    out := make([]byte, len(dsNameNode.Values()))
    for i, v := range dsNameNode.Values() {
        out[i] = v.(byte)
    }
    dsName := string(out)
    ////
    ackNode , err :=  item.(*sm.ListNode).Get(2)
    ack := ackNode.Values()[0].(byte)
    rtypeNode , err := item.(*sm.ListNode).Get(3)
    rtype := rtypeNode.Values()[0].(uint64)
    recLenNode , err := item.(*sm.ListNode).Get(4)
    recLen := recLenNode.Values()[0].(uint64)
    dstm.log.Printf("handle : %d | dsname : %s | ack : %d | rtype : %d | recLen : %d\n",handle,dsName,ack,rtype,recLen);
    if(ACKC13(ack) != ACKC13OK){
        dstm.log.Printf("handleS13F4 error : %d\n",ack);
        return;
    }

    if _ , found := RECV_MAP[uint(handle)]; found {
        dstm.log.Printf("RECV Handle Already open\n")
        return
    } else {
        file, _ := os.OpenFile( dsName + "-COPY" , os.O_RDWR|os.O_CREATE|os.O_TRUNC , 0666)
        dst := &DSTRANSFEROBJ{ handle : uint(handle) , buffer : nil , dsName : dsName , ckPnt : 0 , file : file , state : "IDLE"  }
        RECV_MAP[uint(handle)] = dst
        dstm.log.Printf("Create RECV Handle : %d\n",handle)
    }
}

//read request
func (dstm * DSTMODULE)sendS13F5(handle uint,readlen uint){
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateUintNode( 4 , readlen ) );
    msg :=  sm.CreateDataMessage( 13 , 5 , true , rootNode , -1 , 0 , "ALL" )
    ctx := &SendCtx{ msg : msg , cb : dstm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F5(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    handleNode , err := item.(*sm.ListNode).Get(0)
    handle := handleNode.Values()[0].(uint64)
    readLenNode , err := item.(*sm.ListNode).Get(1)
    readLen := readLenNode.Values()[0].(uint64)
    dstm.log.Printf("handle : %d | readLen : %d\n",handle,readLen);
    ack := ACKC13OK
    ckPnt := int64(0)
    buffer := make([]byte,  readLen)
    filDataLstNode := sm.CreateListNode()
    if sendds , found := SEND_MAP[uint(handle)]; found {
        dstm.log.Printf("SEND Handle found\n")
        n , err := sendds.file.Read(buffer)
        buffer = buffer[:n]
        ckPnt , _ = sendds.file.Seek(0, 1) // current file position
        sendds.ckPnt = uint(ckPnt);
        if( err != nil && err == io.EOF ){
            ack = ACKC13EndOfData
            dstm.log.Printf("SEND Handle file reach end\n")
        }
        filDataLstNode = sm.CreateListNode(sm.CreateBinaryNode(buffer...))

    } else {
        dstm.log.Printf("SEND Handle not found %v\n",handle)
        ack = ACKC13NoOpenDataSet
        ckPnt = 0xFFFFFFFF
        buffer = buffer[:0]
    }
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateBinaryNode( byte(ack) ) , sm.CreateUintNode(4,ckPnt) , filDataLstNode );
    replyMsg := sm.CreateDataMessage( 13, 6, false, rootNode , -1 , msg.SystemBytes() , msg.SourceHost() )
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F6(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    handleNode , err := item.(*sm.ListNode).Get(0)
    handle := handleNode.Values()[0].(uint64)
    ackNode , err :=  item.(*sm.ListNode).Get(1)
    ack := ackNode.Values()[0].(byte)
    ckPntNode  , err := item.(*sm.ListNode).Get(2)
    ckPnt := ckPntNode.Values()[0].(uint64)
    if entry, ok := RECV_MAP[uint(handle)]; ok {
        if(ACKC13(ack) == ACKC13OK){
            filDataLstNode , _ := item.(*sm.ListNode).Get(3)
            filDataNode , _ := filDataLstNode.(*sm.ListNode).Get(0)
            ////
            out := make([]byte, len(filDataNode.Values()))
            for i, v := range filDataNode.Values() {
                out[i] = v.(byte)
            }
            filData := out
            ///
            entry.file.Write(filData)
            entry.ckPnt = uint(ckPnt)
            entry.state = "IDLE"
            dstm.log.Printf("handle : %d | ack : %d | ckPnt : %d | filData : %v\n",handle,ack,ckPnt,filData);
        } else if(ACKC13(ack) == ACKC13EndOfData){
            dstm.log.Printf(" HandleS13F6 handle %d | ack : ACKC13EndOfData",handle);
            delete(RECV_MAP,uint(handle))
            dstm.sendS13F7(uint(handle))
        } else {
            //TODO : need recovery
            entry.state = "IDLE"
            delete(RECV_MAP,uint(handle))
            dstm.log.Printf("Err : HandleS13F6 handle %d | ack : %d ",handle,ack);
        }
    }
}

//close request
func (dstm * DSTMODULE)sendS13F7(handle uint){
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) );
    msg :=  sm.CreateDataMessage( 13 , 7 , true , rootNode , -1 , 0 , "ALL" )
    ctx := &SendCtx{ msg : msg , cb : dstm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F7(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    handleNode , err := item.(*sm.ListNode).Get(0)
    handle := handleNode.Values()[0].(uint64)
    ack := ACKC13OK
    if sendds , found := SEND_MAP[uint(handle)]; found {
        dstm.log.Printf("Close Handle found\n")
        sendds.file.Close()
        delete(SEND_MAP, uint(handle))
    } else {
        dstm.log.Printf("Close Handle not found : %d\n",handle)
        ack = ACKC13NoOpenDataSet
    }
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateBinaryNode( byte(ack) ) );
    replyMsg := sm.CreateDataMessage( 13, 8, false, rootNode , -1 , msg.SystemBytes() , msg.SourceHost() )
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F8(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    handleNode , err := item.(*sm.ListNode).Get(0)
    handle := handleNode.Values()[0].(uint64)
    ackNode , err :=  item.(*sm.ListNode).Get(1)
    ack := ackNode.Values()[0].(byte)
    dstm.log.Printf("handle : %d | ack : %d ",handle,ack);
    return
}

//reset

func (dstm * DSTMODULE)sendS13F9(handle uint){
    for k ,v := range RECV_MAP {
        v.file.Close()
        delete(RECV_MAP, k )
    }
    for k ,v := range SEND_MAP {
        v.file.Close()
        delete(SEND_MAP, k )
    }
    msg :=  sm.CreateDataMessage( 13 , 9 , true , sm.CreateEmptyElementType()  , -1 , 0 , "ALL" )
    ctx := &SendCtx{ msg : msg , cb : dstm.GenericCB , timeout : time.Now().Unix() + (T3/1000) }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F9(msg *sm.DataMessage){
    for k ,v := range RECV_MAP {
        v.file.Close()
        delete(RECV_MAP, k )
    }
    for k ,v := range SEND_MAP {
        v.file.Close()
        delete(SEND_MAP, k )
    }
    replyMsg :=  sm.CreateDataMessage( 13 , 10 , true , sm.CreateEmptyElementType()  , -1 , 0 , "ALL" )
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F10(msg *sm.DataMessage){
    return
}



func (dstm * DSTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 13){
        if(msg.FunctionCode() == 1){
            dstm.handleS13F1(msg)
        }
        if(msg.FunctionCode() == 2){
            dstm.handleS13F2(msg)
        }
        if(msg.FunctionCode() == 3){
            dstm.handleS13F3(msg)
        }
        if(msg.FunctionCode() == 4){
            dstm.handleS13F4(msg)
        }
        if(msg.FunctionCode() == 5){
            dstm.handleS13F5(msg)
        }
        if(msg.FunctionCode() == 6){
            dstm.handleS13F6(msg)
        }
        if(msg.FunctionCode() == 7){
            dstm.handleS13F7(msg)
        }
        if(msg.FunctionCode() == 8){
            dstm.handleS13F8(msg)
        }
        if(msg.FunctionCode() == 9){
            dstm.handleS13F9(msg)
        }
        if(msg.FunctionCode() == 10){
            dstm.handleS13F10(msg)
        }
    }
    return true
}

func (dstm * DSTMODULE)processEvt(evt Evt){
    if(evt.ctx == nil){
        return
    }

    if(evt.cmd == "executefn"){
        fn := evt.ctx.(func())
        fn()
        return
    }
    if(evt.cmd == "recv"){
        msg := evt.ctx.(*RecvCtx).msg.(*sm.DataMessage)
        dstm.processMsg(msg)
    }
}

func (dstm * DSTMODULE)processRecvDs(){
    for _,v := range RECV_MAP {
        if(v.state == "IDLE"){
            dstm.sendS13F5(v.handle,4096)
            v.state = "READ_WAIT"
        } else if(v.state == "READ_WAIT"){
            dstm.log.Printf("wait read result\n");
        }
    }
}

func (dstm * DSTMODULE)stateRun(){
    ticker := time.NewTicker( 20 * time.Millisecond)
    defer func(){
        dstm.log.Printf("Exit DSTMODULE \n");
        ticker.Stop()
        dstm.wg.Done()
    }()

    for {
        select {
            case evt := <-dstm.iChan :
                dstm.processEvt(evt)
            case <-ticker.C:
                dstm.processRecvDs();
            case cmd :=<-dstm.ctrlChan:
                if(cmd == "quit"){
                    return
                }

        }
    }
    return
}
