package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "secs/data"
    "sync"
)

type ALARMMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    deviceID int
}

func NewALARMMODULE(deviceID int) *ALARMMODULE {
    o := ALARMMODULE{
                         run : "stop",
                         iChan : make(chan Evt,10),
                         oChan : make(chan Evt,10 ) ,
                         wg : new(sync.WaitGroup),
                         deviceID : deviceID,
                  }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (am * ALARMMODULE) PutEvt(e Evt) {
    am.iChan <- e
}


func (am * ALARMMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , am.deviceID ,0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    am.oChan <- act
    return
}

func (am * ALARMMODULE)sendS5F1(id uint64){
    alids := make([]uint64,1)
    alids[0] = id
    rootNode := data.GetAlarmsLst(alids)
    node , _ := rootNode.(*sm.ListNode).Get(0)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 5, 1, true,  node ,am.deviceID , 0 , "ALL"),
                ts : time.Now().Unix() }
    fmt.Printf("send report\n")
    am.oChan <- act
}

func (am * ALARMMODULE)trigEvt(e uint32){
    p := make(map[string]interface{})
    p["evtid"] = e
    p["dvctx"] = make(map[uint32]interface{})
    am.oChan <- Evt{ cmd : "TRIG_EVENT" , msg : p ,ts : time.Now().Unix()  }
    return
}

func (am * ALARMMODULE)setAlarm(id uint64,v int){
    evt , ok := data.SetAlarm(id,v)
    if(ok){
        am.sendS5F1(id)
        am.trigEvt(evt);
    }
}

func (am * ALARMMODULE)handleS5F2(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "B" || item.Size() != 1 || err != nil){
        fmt.Printf("Error S5F23 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
}

func (am * ALARMMODULE)handleS5F3(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || item.Size() != 2 || err != nil){
        fmt.Printf("Error S5F3 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
    aledNode , err := item.(*sm.ListNode).Get(0);
    if(aledNode.Type() != "B" || aledNode.Size() != 1 || err != nil){
        fmt.Printf("Error S5F3 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
    alidNode , err := item.(*sm.ListNode).Get(1);
    if(alidNode.Type() != "U4" || err != nil){
        fmt.Printf("Error S5F3 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }

    aled := aledNode.Values().([]int)[0]
    alid := uint64(0xFFFFFFFFFFFFFFFF)
    if(alidNode.Size() > 0){
        alid = alidNode.Values().([]uint64)[0]
    }
    ret := data.SetAlarmEnable(alid,aled)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 5, 4, false, 
                                     sm.CreateBinaryNode( []interface{}{byte(ret)}... ) ,
                                     am.deviceID , msg.SystemBytes() , msg.SourceHost() ),ts : time.Now().Unix()}
    am.oChan <- act
}

func (am * ALARMMODULE)handleS5F5(msg *sm.DataMessage){
    alidNode , err := msg.Get()
    if( alidNode.Type() != "U4" || err != nil){
        fmt.Printf("Error S5F5 format\n")
        am.sendS9FX(msg, 7)
        return ;
    }
    alids := alidNode.Values().([]uint64)
    rootNode := data.GetAlarmsLst(alids)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 5, 6, false, 
                                     rootNode ,
                                     am.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    am.oChan <- act
}



func (am * ALARMMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 5){
        if(msg.FunctionCode() == 2){
            am.handleS5F2(msg)
        }

        if(msg.FunctionCode() == 3){
            am.handleS5F3(msg)
        }
        if(msg.FunctionCode() == 5){
            am.handleS5F5(msg)
        }
    }
    return true
}


func (am * ALARMMODULE)processEvt(evt Evt){
    am.processMsg(evt.msg.(*sm.DataMessage))
}

func (am * ALARMMODULE)moduleStop(){
    am.run = "stop"
    am.iChan <- Evt{ cmd : "quit"}
    am.wg.Wait()
}

func (am * ALARMMODULE)stateRun(){
    defer am.wg.Done()
    am.run = "run"

    for am.run == "run" {
        select {
            case evt := <-am.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                am.processEvt(evt)
        }
    }
    am.run = "stop"
    fmt.Printf("Exit ALARMMODULE \n");
    return
}
