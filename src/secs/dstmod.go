// Data Set Transfers module(STREAM13)
package secs

import (
    "sync"
    "time"
//    "secs/data"
    "secs/logger"
    "os"
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
    bin := make([]interface{}, 10)
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
    rootNode := sm.CreateListNode( sm.CreateASCIINode(dsName),sm.CreateBinaryNode( ACKC13OK ) );
    replyMsg := sm.CreateDataMessage( 13, 2, false, rootNode , dstm.deviceID , msg.SystemBytes() , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : replyMsg , ts : time.Now().Unix()  }
    dstm.oChan <- act
}

/* receiving sysytem */
func (dstm * DSTMODULE)sendS13F3(handle uint , dsName string , ckpnt uint){
    if _ , found := RECV_MAP[handle]; found {
        dstm.log.Printf("RECV Handle Already open\n")
        return 
    } else {
        file, _ := os.OpenFile(dsName, os.O_RDONLY, 0666)
        dst := DSTRANSFEROBJ{ handle:handle , buffer : nil , dsName : dsName , ckPnt : 0 , file : file }
        RECV_MAP[handle] = dst
        dstm.log.Printf("Create RECV Handle : %d\n",handle)
    }


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
    handle := handleNode.Values().([]uint)[0]
    dsNameNode , err := item.(*sm.ListNode).Get(1)
    dsName := dsNameNode.Values().(string)
    ckPntNode  , err := item.(*sm.ListNode).Get(2)
    ckPnt := ckPntNode.Values().([]uint)[0]
    dstm.log.Printf("handle : %d | dsname : %s | ckPnt : %d\n",handle,dsName,ckPnt);
    RTYPE := 0   //0 : stream | 1 : discrete ,support stream only
    RECLEN := 0  //record length
    ack := ACKC13OK
    if _ , found := SEND_MAP[handle]; found {
        dstm.log.Printf("Send Handle Already open\n")
        ack = ACKC13HandleInUse
    } else {
        file, _ := os.OpenFile(dsName, os.O_RDONLY, 0666)
        dst := DSTRANSFEROBJ{ handle:handle , buffer : nil , dsName : dsName , ckPnt : ckPnt , file : file }
        SEND_MAP[handle] = dst
        dstm.log.Printf("Create Send Handle : %d\n",handle)
    }
    rootNode := sm.CreateListNode( sm.CreateUintNode(4,handle) , sm.CreateASCIINode(dsName),sm.CreateBinaryNode( ack ) , sm.CreateUintNode(1,RTYPE) , sm.CreateUintNode(4,RECLEN) );
    replyMsg := sm.CreateDataMessage( 13, 4, false, rootNode , dstm.deviceID , msg.SystemBytes() , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : replyMsg , ts : time.Now().Unix()  }
    dstm.oChan <- act
}



func (dstm * DSTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 13){
        if(msg.FunctionCode() == 1){
            dstm.handleS13F1(msg)
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
