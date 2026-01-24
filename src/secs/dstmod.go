// Data Set Transfers module(STREAM13)
package secs

import (
    "sync"
    "time"
//    "secs/data"
    "secs/logger"
    "os"
    "io"
    sm "secs/secs_message"
)

type DSTMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run   bool
    wg *sync.WaitGroup
    deviceID int
    log *logger.Logger
}

type DSTRANSFEROBJ struct{
    handle  uint
    buffer  []byte
    dsName  string
    ckPnt   uint
    file    *os.File
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

var  RECV_MAP map[uint]DSTRANSFEROBJ
var  SEND_MAP   map[uint]DSTRANSFEROBJ

func NewDSTMODULE(deviceID int, log *logger.Logger) *DSTMODULE {
    o := DSTMODULE{
                         run : false,
                         iChan : make(chan Evt,10),
                         oChan : make(chan Evt,10 ) ,
                         wg : new(sync.WaitGroup),
                         deviceID : deviceID,
                         log: log,
                  }
    o.wg.Add(1)
    RECV_MAP = make(map[uint]DSTRANSFEROBJ, 100) 
    SEND_MAP   = make(map[uint]DSTRANSFEROBJ, 100) 
    go o.stateRun()
    return &o
}

func (dstm * DSTMODULE) PutEvt(e Evt) {
    dstm.iChan <- e
}

func (dstm * DSTMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , dstm.deviceID , 0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    dstm.oChan <- act
    return
}

func (dstm * DSTMODULE)sendS13F1(dsName string){
    msg :=  sm.CreateDataMessage( 13 , 1 , true , sm.CreateASCIINode(dsName) , dstm.deviceID , 0 , "ALL" )
    act := Evt{ cmd : "send" , msg : msg ,ts : time.Now().Unix()}
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F1(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "A" || err != nil){
        dstm.log.Printf("Error S13F1 format\n")
        dstm.sendS9FX(msg, 7)
        return ;
    }
    dsName := item.Values().(string)
    dstm.log.Printf("dsname : %v\n",dsName);
    rootNode := sm.CreateListNode( sm.CreateASCIINode(dsName),sm.CreateBinaryNode( byte(ACKC13OK) ) );
    replyMsg := sm.CreateDataMessage( 13, 2, false, rootNode , dstm.deviceID , msg.SystemBytes() , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : replyMsg , ts : time.Now().Unix()  }
    dstm.oChan <- act
}

func (dstm * DSTMODULE)handleS13F2(msg *sm.DataMessage){
}


/* receiving sysytem */
func (dstm * DSTMODULE)sendS13F3(handle uint , dsName string , ckpnt uint){
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateASCIINode(dsName),sm.CreateUintNode( 4 , ckpnt ) );
    msg :=  sm.CreateDataMessage( 13 , 3 , true , rootNode , dstm.deviceID , 0 , "ALL" )
    act := Evt{ cmd : "send" , msg : msg ,ts : time.Now().Unix()}
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
    handle := handleNode.Values().([]uint64)[0]
    dsNameNode , err := item.(*sm.ListNode).Get(1)
    dsName := dsNameNode.Values().(string)
    ckPntNode  , err := item.(*sm.ListNode).Get(2)
    ckPnt := ckPntNode.Values().([]uint64)[0]
    dstm.log.Printf("handle : %d | dsname : %s | ckPnt : %d\n",handle,dsName,ckPnt);
    RTYPE := 0   //0 : stream | 1 : discrete ,support stream only
    RECLEN := 0  //record length
    ack := ACKC13OK
    if _ , found := SEND_MAP[uint(handle)]; found {
        dstm.log.Printf("Send Handle Already open\n")
        ack = ACKC13HandleInUse
    } else {
        file, err := os.OpenFile("storage/equipment/" + dsName, os.O_RDONLY, 0666)
        if(err != nil){
            dstm.log.Printf("OpenFile failed : %v\n",err)
            ack = ACKC13UnknownDataSetName
            ckPnt = 0
        } else {
            dstm.log.Printf("OpenFile Success\n")
        }
        dst := DSTRANSFEROBJ{ handle:uint(handle) , buffer : nil , dsName : dsName , ckPnt : uint(ckPnt) , file : file }
        SEND_MAP[uint(handle)] = dst
        dstm.log.Printf("Create Send Handle : %d\n",handle)
    }
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateASCIINode(dsName),sm.CreateBinaryNode( byte(ack) ) , sm.CreateUintNode(1,RTYPE) , sm.CreateUintNode(4,RECLEN) );
    replyMsg := sm.CreateDataMessage( 13, 4, false, rootNode , dstm.deviceID , msg.SystemBytes() , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : replyMsg , ts : time.Now().Unix()  }
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
    handle := handleNode.Values().([]uint64)[0]
    dsNameNode , err := item.(*sm.ListNode).Get(1)
    dsName := dsNameNode.Values().(string)
    ackNode , err :=  item.(*sm.ListNode).Get(2)
    ack := ackNode.Values().([]byte)[0]
    rtypeNode , err := item.(*sm.ListNode).Get(3)
    rtype := rtypeNode.Values().([]uint64)[0]
    recLenNode , err := item.(*sm.ListNode).Get(4)
    recLen := recLenNode.Values().([]uint64)[0]
    dstm.log.Printf("handle : %d | dsname : %s | ack : %d | rtype : %d | recLen : %d\n",handle,dsName,ack,rtype,recLen);
    if(ACKC13(ack) != ACKC13OK){
        dstm.log.Printf("handleS13F4 error : %d\n",ack);
        return;
    }

    if _ , found := RECV_MAP[uint(handle)]; found {
        dstm.log.Printf("RECV Handle Already open\n")
        return
    } else {
        file, _ := os.OpenFile(dsName, os.O_RDWR|os.O_CREATE|os.O_TRUNC , 0666)
        dst := DSTRANSFEROBJ{ handle : uint(handle) , buffer : nil , dsName : dsName , ckPnt : 0 , file : file }
        RECV_MAP[uint(handle)] = dst
        dstm.log.Printf("Create RECV Handle : %d\n",handle)
    }
}

//read request
func (dstm * DSTMODULE)sebdS13F5(handle uint,readlen uint){
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateUintNode( 4 , readlen ) );
    msg :=  sm.CreateDataMessage( 13 , 5 , true , rootNode , dstm.deviceID , 0 , "ALL" )
    act := Evt{ cmd : "send" , msg : msg ,ts : time.Now().Unix()}
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
    handle := handleNode.Values().([]uint64)[0]
    readLenNode , err := item.(*sm.ListNode).Get(1)
    readLen := readLenNode.Values().([]uint64)[0]
    dstm.log.Printf("handle : %d | readLen : %d\n",handle,readLen);
    ack := ACKC13OK
    ckPnt := int64(0)
    buffer := make([]byte,  readLen)
    if sendds , found := SEND_MAP[uint(handle)]; found {
        dstm.log.Printf("SEND Handle found\n")
        n , err := sendds.file.Read(buffer)
        buffer = buffer[:n]
        ckPnt , _ = sendds.file.Seek(0, 1) // current file position
        if( err != nil && err == io.EOF ){
            ack = ACKC13EndOfData
            dstm.log.Printf("SEND Handle file reach end\n")
        }

    } else {
        dstm.log.Printf("SEND Handle not found\n",handle)
        ack = ACKC13NoOpenDataSet
    }

    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateBinaryNode( byte(ack) ) , sm.CreateUintNode(4,ckPnt) , sm.CreateBinaryNode(buffer...) );
    replyMsg := sm.CreateDataMessage( 13, 6, false, rootNode , dstm.deviceID , msg.SystemBytes() , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : replyMsg , ts : time.Now().Unix()  }
    dstm.oChan <- act
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
        }


    }
    return true
}

func (dstm * DSTMODULE)processEvt(evt Evt){
    msg := evt.msg.(*sm.DataMessage)
    dstm.processMsg(msg)
}

func (dstm * DSTMODULE)moduleStop(){
    dstm.run = false
    dstm.iChan <- Evt{ cmd : "quit"}
    dstm.wg.Wait()
}

func (dstm * DSTMODULE)stateRun(){
    defer dstm.wg.Done()
    dstm.run = true

    for dstm.run == true {
        select {
            case evt := <-dstm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                dstm.processEvt(evt)
        }
    }
    dstm.run = false
    dstm.log.Printf("Exit DSTMODULE \n");
    return
}
