package data

import "sync"

// ----------------------------
// SECS-II JSON Node structure
// ----------------------------

type NodeValue struct {
    Type   string       `json:"type"`              // "U4", "A", "L", "B", "F4", ...
    Value  string       `json:"value,omitempty"`   // A-type
    Values []float64    `json:"values,omitempty"`  // numeric type
    Bools  []bool       `json:"bools,omitempty"`   // BOOLEAN
    Bytes  string       `json:"bytes,omitempty"`   // B-type
    Items  []*NodeValue `json:"items,omitempty"`   // L-type
}

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
    id    uint32
    name  string
    units string
    value interface{}
    /*
       Equipment const variable
       Equipment const variable list format is not allowed
    */

    defv     interface{} // for ec only
    min      interface{} // for ec only
    max      interface{} // for ec only
    limitEvt interface{} // could be nil
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
