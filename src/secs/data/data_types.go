package data

import "sync"
//import "encoding/json"
import sm "secs/secs_message"
// ----------------------------
// Command / data structures
// ----------------------------

type DEFAULT_STATE struct{
    DEFAULT_CTRLMAINSTATE string         `json:"DEFAULT_CTRLMAINSTATE"`
    DEFAULT_CTRLSUBSTATE string          `json:"DEFAULT_CTRLSUBSTATE"`
    DEFAULT_REJECT_CTRLSUBSTATE string   `json:"DEFAULT_REJECT_CTRLSUBSTATE"`
    DEFAULT_ACCEPT_CTRLSUBSTATE string   `json:"DEFAULT_ACCEPT_CTRLSUBSTATE"`
    DEFAULT_COMSTATE int                 `json:"DEFAULT_COMSTATE"`
}

/*func getType(v interface{})(string){
    t := reflect.TypeOf(v)
    if t == nil {
        return ""
    }
    return t.String()
}*/

var G_STATE DEFAULT_STATE


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
    Id     uint32   `json:"id"`
    Name   string   `json:"name"`
    RptLst []uint32 `json:"rptlst"`
    DvLst  []uint32
    Enable bool     `json:"enable"`
}

type SECSRPT struct { // Report
    Id   uint32    `json:"id"`
    Name string    `json:"name"`
    Vids []uint32  `json:"vids"`
}

type SECSALARM struct {
    Id     uint32  `json:"id"`
    Name   string  `json:"name"`
    Evt    uint32  `json:"evt"`
    Enable bool    `json:"enable"`
    Set    bool
    Text   string  `json:"text"`
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
