package data

import "sync"
//import "encoding/json"
import sm "secs/secs_message"
// ----------------------------
// Command / data structures
// ----------------------------

// 單一 goroutine 專門操作 SECS_DATA，其他 goroutine 丟 closure 進來
type ACCESS_CMD struct {
    fn func(sd *SECS_DATA)
}

type SECS_DATA struct {
    iChan chan ACCESS_CMD
    run   string

    evt   map[uint32]*SECSCE
    rpt   map[uint32]*SECSRPT
    svs   map[uint32]*SECSVARIABLE
    dvs   map[uint32]*SECSVARIABLE
    ecs   map[uint32]*SECSVARIABLE
    alarm map[uint32]*SECSALARM

    wg *sync.WaitGroup
}

type SECSCE struct { // Event
    id     uint32
    name   string
    rptLst []uint32
    dvLst  []uint32
    enable bool
}

type SECSRPT struct { // Report
    id   uint32
    name string
    vids []uint32
}

type SECSALARM struct {
    id     uint32
    name   string
    evt    uint32
    enable bool
    set    bool
    text   string
}

/*
   "JIS" not supported
*/
type SECSVARIABLE struct { // status/data variable
    Id    uint32          `json:"id"`
    Name  string          `json:"name"`
    Units string          `json:"units"`
    Value sm.ElementWrapper `json:"nodevalue"`
    Defv     sm.ElementWrapper
    Min      sm.ElementWrapper  `json:"min"`// for ec only
    Max      sm.ElementWrapper  `json:"max"`// for ec only
    LimitEvt uint32  `json:"limitevt"`// could be nil
}

type VidElementResult struct {
    Ret   bool
    Value interface{}
    Max   interface{}
    Min   interface{}
    Evt   interface{}
    Unit  string
}

// setAlarm 的回傳封裝
type AlarmSetResult struct {
    Ret uint32
    Ok  bool
}

var gData SECS_DATA
