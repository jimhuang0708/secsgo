package secs

import (
//    "net"
    "time"
//    "secs/data"
    "secs/logger"
//    "encoding/json"
    "sync"
    sm "secs/secs_message"
)

type BaseComponent struct {
    iChan chan Evt
    oChan chan Evt
    wg       *sync.WaitGroup
    log *logger.Logger
    run      bool
}

func CreateBaseComponent(log *logger.Logger) BaseComponent {
    return BaseComponent{
        iChan: make(chan Evt, 10),
        oChan: make(chan Evt, 10),
        wg:    new(sync.WaitGroup),
        run:   false,
        log:   log,
    }
}


type BaseModule struct {
    BaseComponent
}

func CreateBaseModule(log *logger.Logger) BaseModule {
    return BaseModule { BaseComponent : CreateBaseComponent(log) }
}

func (bm * BaseModule)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]byte, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , -1 ,0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    bm.oChan <- act
    return
}
