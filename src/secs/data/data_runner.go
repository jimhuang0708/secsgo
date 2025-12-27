package data

import (
    "fmt"
    "strconv"
    "sync"

    "github.com/spf13/viper"
    sm "secs/secs_message"
)

// ----------------------------
// Runner / lifecycle
// ----------------------------

func InitSECSData() {
    gData = SECS_DATA{
        run:   "stop",
        iChan: make(chan ACCESS_CMD, 10),
        evt:   make(map[uint32]*SECSCE),
        rpt:   make(map[uint32]*SECSRPT),
        svs:   make(map[uint32]*SECSVARIABLE),
        dvs:   make(map[uint32]*SECSVARIABLE),
        ecs:   make(map[uint32]*SECSVARIABLE),
        alarm: make(map[uint32]*SECSALARM),
        wg:    new(sync.WaitGroup),
    }
    gData.wg.Add(1)
    go gData.moduleRun()
}

func CloneElementType(obj any) sm.ElementType {
    if obj == nil {
        return nil
    }
    itemNode := obj.(sm.ElementType)
    if itemNode == nil {
        return nil
    }
    return itemNode.Clone()
}

func (sd *SECS_DATA) moduleLoadData() {
    rpts := viper.Get("sysrpt")
    for i := 0; i < len(rpts.([]interface{})); i++ {
        idx := "sysrpt." + strconv.Itoa(i)
        id := viper.GetUint32(idx + ".id")
        name := viper.GetString(idx + ".name")
        vid := viper.GetIntSlice(idx + ".vid")
        temp_rpt := &SECSRPT{id: id, name: name, vids: make([]uint32, 0)}
        for j := 0; j < len(vid); j++ {
            temp_rpt.vids = append(temp_rpt.vids, uint32(vid[j]))
        }
        fmt.Printf("sysrpt : %v\n", temp_rpt)
        sd.rpt[temp_rpt.id] = temp_rpt
    }

    evts := viper.Get("sysevt")
    for i := 0; i < len(evts.([]interface{})); i++ {
        idx := "sysevt." + strconv.Itoa(i)
        id := viper.GetUint32(idx + ".id")
        name := viper.GetString(idx + ".name")
        rpt := viper.GetIntSlice(idx + ".rpt")
        vid := viper.GetIntSlice(idx + ".vid")
        enable := viper.GetBool(idx + ".enable")
        temp_ce := &SECSCE{id: id, name: name, rptLst: make([]uint32, 0), dvLst: make([]uint32, 0), enable: enable}
        for j := 0; j < len(rpt); j++ {
            temp_ce.rptLst = append(temp_ce.rptLst, uint32(rpt[j]))
        }
        for j := 0; j < len(vid); j++ {
            temp_ce.dvLst = append(temp_ce.dvLst, uint32(vid[j]))
        }
        fmt.Printf("sysevt : %v\n", temp_ce)
        sd.evt[temp_ce.id] = temp_ce
    }

    vids := viper.Get("syssv")
    for i := 0; i < len(vids.([]interface{})); i++ {
        idx := "syssv." + strconv.Itoa(i)
        id := viper.GetUint32(idx + ".id")
        limitEvt := viper.GetUint32(idx + ".limitevt")
        name := viper.GetString(idx + ".name")
        units := viper.GetString(idx + ".units")
        var valueNode NodeValue
        viper.UnmarshalKey(idx+".nodevalue", &valueNode)
        value, _ := valueNode.EncodeSecs()
        fmt.Printf("value %v\n", value)
        var maxNode NodeValue
        viper.UnmarshalKey(idx+".max", &maxNode)
        max, _ := maxNode.EncodeSecs()
        fmt.Printf("max %v\n", max)
        var minNode NodeValue
        viper.UnmarshalKey(idx+".min", &minNode)
        min, _ := minNode.EncodeSecs()
        fmt.Printf("min %v\n", min)

        if value == nil {
            panic("syssv lack of default value!\n")
        }
        fmt.Printf("id : %d | name : %s | units : %s | limitEvt : %d\n", id, name, units, limitEvt)
        temp_sv := &SECSVARIABLE{id: id, name: name, units: units, value: value, limitEvt: limitEvt, max: max, min: min}
        fmt.Printf("sysesv : %v\n", temp_sv)
        sd.svs[temp_sv.id] = temp_sv
    }

    vids = viper.Get("sysdv")
    for i := 0; i < len(vids.([]interface{})); i++ {
        idx := "sysdv." + strconv.Itoa(i)
        id := viper.GetUint32(idx + ".id")
        limitEvt := viper.GetUint32(idx + ".limitevt")
        name := viper.GetString(idx + ".name")
        units := viper.GetString(idx + ".units")
        var valueNode NodeValue
        viper.UnmarshalKey(idx+".nodevalue", &valueNode)
        value, _ := valueNode.EncodeSecs()
        fmt.Printf("value %v\n", value)
        var maxNode NodeValue
        viper.UnmarshalKey(idx+".max", &maxNode)
        max, _ := maxNode.EncodeSecs()
        fmt.Printf("max %v\n", max)
        var minNode NodeValue
        viper.UnmarshalKey(idx+".min", &minNode)
        min, _ := minNode.EncodeSecs()
        fmt.Printf("min %v\n", min)
        if value == nil {
            panic("sysdv lack of default value!\n")
        }
        fmt.Printf("id : %d | name : %s | units : %s | limitEvt : %d\n", id, name, units, limitEvt)
        temp_dv := &SECSVARIABLE{id: id, name: name, units: units, value: value, limitEvt: limitEvt, max: max, min: min}
        sd.dvs[temp_dv.id] = temp_dv
    }

    vids = viper.Get("sysec")
    for i := 0; i < len(vids.([]interface{})); i++ {
        idx := "sysec." + strconv.Itoa(i)
        id := viper.GetUint32(idx + ".id")
        name := viper.GetString(idx + ".name")
        units := viper.GetString(idx + ".units")
        var valueNode NodeValue
        viper.UnmarshalKey(idx+".nodevalue", &valueNode)
        value, _ := valueNode.EncodeSecs()
        fmt.Printf("value %v\n", value)
        var maxNode NodeValue
        viper.UnmarshalKey(idx+".max", &maxNode)
        max, _ := maxNode.EncodeSecs()
        fmt.Printf("max %v\n", max)
        var minNode NodeValue
        viper.UnmarshalKey(idx+".min", &minNode)
        min, _ := minNode.EncodeSecs()
        fmt.Printf("min %v\n", min)

        if value == nil {
            panic("sysec lack of default value!\n")
        }
        fmt.Printf("id : %d | name : %s | units : %s \n", id, name, units)
        temp_ec := &SECSVARIABLE{id: id, name: name, units: units, min: min, max: max, defv: value, value: value, limitEvt: nil}
        sd.ecs[temp_ec.id] = temp_ec
    }

    alarms := viper.Get("sysalarm")
    for i := 0; i < len(alarms.([]interface{})); i++ {
        idx := "sysalarm." + strconv.Itoa(i)
        id := viper.GetUint32(idx + ".id")
        name := viper.GetString(idx + ".name")
        enable := viper.GetBool(idx + "enable")
        text := viper.GetString(idx + ".text")
        evt := viper.GetUint32(idx + ".evt")
        temp_alarm := &SECSALARM{id: id, name: name, enable: enable, set: false, text: text, evt: evt}
        fmt.Printf("sysalarm : %v\n", temp_alarm)
        sd.alarm[temp_alarm.id] = temp_alarm
    }

    fmt.Printf("%v \n", gData)
}

// 單純呼叫 closure
func (sd *SECS_DATA) handleAccess(e ACCESS_CMD) {
    if e.fn != nil {
        e.fn(sd)
    }
}

func (sd *SECS_DATA) moduleStop() {
    done := make(chan struct{}, 1)
    sd.iChan <- ACCESS_CMD{
        fn: func(d *SECS_DATA) {
            d.run = "stop"
            done <- struct{}{}
        },
    }
    <-done
    sd.wg.Wait()
}

func (sd *SECS_DATA) moduleRun() {
    defer sd.wg.Done()
    sd.moduleLoadData()
    sd.run = "run"
    for sd.run == "run" {
        e := <-sd.iChan
        sd.handleAccess(e)
    }
    sd.run = "stop"
    fmt.Printf("Exit SECS_DATA\n")
    return
}
