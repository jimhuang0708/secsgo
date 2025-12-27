package data

import (
    "fmt"

    sm "secs/secs_message"
)

// ----------------------------
// Public API (for other goroutine)
// 保持原本函式簽章不變，只是內部改成送 closure
// ----------------------------

func ModuleStop(){
   gData.moduleStop()
}

// setVidValue(id, v) bool

func SetVidValue(id uint32, v sm.ElementType) bool {
    reply := make(chan bool, 1)
    gData.iChan <- cmdSetVidValue(id, v, reply)
    answer := <-reply
    fmt.Printf("setVidValue answer: %v\n", answer)
    return answer
}

func cmdSetVidValue(id uint32, v sm.ElementType, reply chan<- bool) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.setVidValue(id, v)
            reply <- ret
        },
    }
}

// isVidExist(id) bool

func IsVidExist(id uint32) bool {
    reply := make(chan bool, 1)
    gData.iChan <- cmdIsVidExist(id, reply)
    answer := <-reply
    fmt.Printf("isVidExist answer: %v\n", answer)
    return answer
}

func cmdIsVidExist(id uint32, reply chan<- bool) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.isVidExist(id)
            reply <- ret
        },
    }
}

// isEvtExist(id) bool

func IsEvtExist(id uint32) bool {
    reply := make(chan bool, 1)
    gData.iChan <- cmdIsEvtExist(id, reply)
    answer := <-reply
    fmt.Printf("isEvtExist answer: %v\n", answer)
    return answer
}

func cmdIsEvtExist(id uint32, reply chan<- bool) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.isEvtExist(id)
            reply <- ret
        },
    }
}

// createReport(id, v...)

func CreateReport(id uint32, v ...uint32) {
    done := make(chan struct{}, 1)
    gData.iChan <- cmdCreateReport(id, v, done)
    <-done
    return
}

func cmdCreateReport(id uint32, vids []uint32, done chan<- struct{}) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            sd.createReport(id, vids...)
            done <- struct{}{}
        },
    }
}

// deleteAllReport()

func DeleteAllReport() {
    done := make(chan struct{}, 1)
    gData.iChan <- cmdDeleteAllReport(done)
    <-done
    return
}

func cmdDeleteAllReport(done chan<- struct{}) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            sd.deleteAllReport()
            done <- struct{}{}
        },
    }
}

// setEvtRptLink(id, v...) string

func SetEvtRptLink(id uint32, v ...uint32) string {
    reply := make(chan string, 1)
    gData.iChan <- cmdSetEvtRptLink(id, v, reply)
    return <-reply
}

func cmdSetEvtRptLink(id uint32, rpts []uint32, reply chan<- string) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.setEvtRptLink(id, rpts...)
            reply <- ret
        },
    }
}

// enableEvent(act, v...) bool

func EnableEvent(act bool, v ...uint32) bool {
    reply := make(chan bool, 1)
    gData.iChan <- cmdEnableEvent(act, v, reply)
    return <-reply
}

func cmdEnableEvent(act bool, evts []uint32, reply chan<- bool) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.enableEvent(act, evts...)
            reply <- ret
        },
    }
}

// getVidElementType(id) (bool,value,max,min,evt,unit)

func GetVidElementType(id uint32) (bool, interface{}, interface{}, interface{}, interface{}, string) {
    reply := make(chan VidElementResult, 1)
    gData.iChan <- cmdGetVidElementType(id, reply)
    r := <-reply
    return r.Ret, r.Value, r.Max, r.Min, r.Evt, r.Unit
}

func cmdGetVidElementType(id uint32, reply chan<- VidElementResult) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret, value, max, min, evt, unit := sd.getVidElementType(id)
            reply <- VidElementResult{
                Ret:   ret,
                Value: value,
                Max:   max,
                Min:   min,
                Evt:   evt,
                Unit:  unit,
            }
        },
    }
}

// getEventNameList(evtLst) sm.ElementType

func GetEventNameList(evtLst []uint32) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetEventNameList(evtLst, reply)
    return <-reply
}

func cmdGetEventNameList(evtLst []uint32, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getEventNameList(evtLst)
            reply <- ret
        },
    }
}

// getEventReport(evtID, dvCtx) sm.ElementType

func GetEventReport(evtID uint32, dvCtx map[uint32]interface{}) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetEventReport(evtID, dvCtx, reply)
    return <-reply
}

func cmdGetEventReport(evtID uint32, dvCtx map[uint32]interface{}, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getEventReport(evtID, dvCtx)
            reply <- ret
        },
    }
}

// getRptReport(rptID) sm.ElementType

func GetRptReport(rptID uint32) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetRptReport(rptID, reply)
    return <-reply
}

func cmdGetRptReport(rptID uint32, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getRptReport(rptID)
            reply <- ret
        },
    }
}

// getSVElementTypeLst(svidLst) sm.ElementType

func GetSVElementTypeLst(svidLst []uint32) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetSVElementTypeLst(svidLst, reply)
    return <-reply
}

func cmdGetSVElementTypeLst(svidLst []uint32, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getSVElementTypeLst(svidLst)
            reply <- ret
        },
    }
}

// getSVNameLst(svidLst) sm.ElementType

func GetSVNameLst(svidLst []uint32) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetSVNameLst(svidLst, reply)
    return <-reply
}

func cmdGetSVNameLst(svidLst []uint32, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getSVNameLst(svidLst)
            reply <- ret
        },
    }
}

// setEC(ecs) int

func SetEC(ecs map[uint32]interface{}) int {
    reply := make(chan int, 1)
    gData.iChan <- cmdSetEC(ecs, reply)
    return <-reply
}

func cmdSetEC(ecs map[uint32]interface{}, reply chan<- int) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.setEC(ecs)
            reply <- ret
        },
    }
}

// getEC(ecLst) sm.ElementType

func GetEC(ecLst []uint32) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetEC(ecLst, reply)
    return <-reply
}

func cmdGetEC(ecLst []uint32, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getEC(ecLst)
            reply <- ret
        },
    }
}

// getECName(ecLst) sm.ElementType

func GetECName(ecLst []uint32) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetECName(ecLst, reply)
    return <-reply
}

func cmdGetECName(ecLst []uint32, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getECName(ecLst)
            reply <- ret
        },
    }
}

// setAlarmEnable(alid, aled) int

func SetAlarmEnable(alid uint64, aled int) int {
    reply := make(chan int, 1)
    gData.iChan <- cmdSetAlarmEnable(alid, aled, reply)
    return <-reply
}

func cmdSetAlarmEnable(alid uint64, aled int, reply chan<- int) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.setAlarmEnable(alid, aled)
            reply <- ret
        },
    }
}

// getAlarmsLst(alids) sm.ElementType

func GetAlarmsLst(alids []uint64) sm.ElementType {
    reply := make(chan sm.ElementType, 1)
    gData.iChan <- cmdGetAlarmsLst(alids, reply)
    return <-reply
}

func cmdGetAlarmsLst(alids []uint64, reply chan<- sm.ElementType) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getAlarmsLst(alids)
            reply <- ret
        },
    }
}

// setAlarm(alid, alcd) (uint32, bool)

func SetAlarm(alid uint64, alcd int) (uint32, bool) {
    reply := make(chan AlarmSetResult, 1)
    gData.iChan <- cmdSetAlarm(alid, alcd, reply)
    r := <-reply
    return r.Ret, r.Ok
}

func cmdSetAlarm(alid uint64, alcd int, reply chan<- AlarmSetResult) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret, ok := sd.setAlarm(alid, alcd)
            reply <- AlarmSetResult{
                Ret: ret,
                Ok:  ok,
            }
        },
    }
}

// getDvbyName(namelist...) []uint32

func GetDvbyName(namelist ...string) []uint32 {
    reply := make(chan []uint32, 1)
    gData.iChan <- cmdGetDvByName(namelist, reply)
    return <-reply
}

func cmdGetDvByName(namelist []string, reply chan<- []uint32) ACCESS_CMD {
    return ACCESS_CMD{
        fn: func(sd *SECS_DATA) {
            ret := sd.getDvbyName(namelist)
            reply <- ret
        },
    }
}
