package data

import (
    "fmt"
    "sort"

    sm "secs/secs_message"
)

// ----------------------------
// Internal methods (行為同原本)
// ----------------------------

func (sd *SECS_DATA) createReport(id uint32, v ...uint32) {
    // renew one report
    sd.rpt[id] = &SECSRPT{id: id, vids: make([]uint32, 0)}
    for _, value := range v {
        sd.rpt[id].vids = append(sd.rpt[id].vids, value)
    }
}

func (sd *SECS_DATA) deleteAllReport() {
    fmt.Printf("Delete all report \n")
    sd.rpt = make(map[uint32]*SECSRPT)
}

func (sd *SECS_DATA) isEvtExist(id uint32) bool {
    _, ok := sd.evt[id]
    return ok
}

func (sd *SECS_DATA) isRptExtist(id uint32) bool {
    _, ok := sd.rpt[id]
    return ok
}

func (sd *SECS_DATA) isVidExist(id uint32) bool {
    _, ok := sd.svs[id]
    if ok {
        return true
    }
    _, ok = sd.dvs[id]
    if ok {
        return true
    }
    _, ok = sd.ecs[id]
    if ok {
        return true
    }
    return false
}

func (sd *SECS_DATA) setVidValue(id uint32, v sm.ElementType) bool {
    _, ok := sd.svs[id]
    if ok {
        sd.svs[id].value = v
        return true
    }
    _, ok = sd.dvs[id]
    if ok {
        sd.dvs[id].value = v
        return true
    }
    _, ok = sd.ecs[id]
    if ok {
        sd.ecs[id].value = v
        return true
    }
    return false
}

func (sd *SECS_DATA) getVidElementType(id uint32) (bool, sm.ElementType, sm.ElementType, sm.ElementType, interface{}, string) {
    variable, ok := sd.svs[id]
    if ok {
        return true, CloneElementType(variable.value), CloneElementType(variable.max), CloneElementType(variable.min), variable.limitEvt, variable.units
    }
    variable, ok = sd.dvs[id]
    if ok {
        return true, CloneElementType(variable.value), CloneElementType(variable.max), CloneElementType(variable.min), variable.limitEvt, variable.units
    }
    variable, ok = sd.ecs[id]
    if ok {
        return true, CloneElementType(variable.value), CloneElementType(variable.max), CloneElementType(variable.min), variable.limitEvt, variable.units
    }
    return false, nil, nil, nil, nil, ""
}

func (sd *SECS_DATA) getVidVariable(id uint32) (bool, *SECSVARIABLE) {
    variable, ok := sd.svs[id]
    if ok {
        return true, variable
    }
    variable, ok = sd.dvs[id]
    if ok {
        return true, variable
    }
    variable, ok = sd.ecs[id]
    if ok {
        return true, variable
    }
    return false, nil
}

func (sd *SECS_DATA) setEvtRptLink(id uint32, v ...uint32) string {
    // renew one event/rpt link
    _, ok := sd.evt[id]
    if ok {
        sd.evt[id].rptLst = make([]uint32, 0)
        for _, value := range v {
            if !sd.isRptExtist(value) {
                fmt.Printf("Error , rpt %d not exist\n", id)
                return "norpt"
            }
            sd.evt[id].rptLst = append(sd.evt[id].rptLst, value)
        }
        return "ok"
    } else {
        fmt.Printf("Error , event %d not exist\n", id)
        return "noevt"
    }
}

func (sd *SECS_DATA) enableEvent(act bool, v ...uint32) bool {
    if len(v) == 0 {
        for k, e := range sd.evt {
            if act {
                e.enable = true
                fmt.Printf("Enable All Event -> %d\n", k)
            } else {
                e.enable = false
                fmt.Printf("Disable All Event -> %d\n", k)
            }
        }
        return true
    }
    for _, value := range v {
        if act {
            val, ok := sd.evt[uint32(value)]
            if ok {
                val.enable = true
                fmt.Printf("Enable Event %d Accept\n", value)
            } else {
                fmt.Printf("Enable Event %d Reject\n", value)
                return false
            }
        } else {
            val, ok := sd.evt[uint32(value)]
            if ok {
                val.enable = false
                fmt.Printf("Disable Event %d Accept\n", value)
            } else {
                fmt.Printf("Disable Event %d Reject\n", value)
                return false
            }
        }
    }
    return true
}

func (sd *SECS_DATA) getEventNameList(evtLst []uint32) sm.ElementType {
    if len(evtLst) == 0 { // select all
        for k := range sd.evt {
            evtLst = append(evtLst, k)
        }
        sort.Slice(evtLst, func(i, j int) bool { return evtLst[i] < evtLst[j] })
    }
    evtNodes := make([]interface{}, 0)
    for k := 0; k < len(evtLst); k++ {
        if v, ok := sd.evt[evtLst[k]]; ok {
            dvLst := make([]interface{}, len(v.dvLst))
            for i := range v.dvLst {
                dvLst[i] = sm.CreateUintNode(4, uint32(v.dvLst[i]))
            }
            n := sm.CreateListNode(sm.CreateUintNode(4, uint32(evtLst[k])), sm.CreateASCIINode(v.name), sm.CreateListNode(dvLst...))
            evtNodes = append(evtNodes, n)
        } else {
            n := sm.CreateListNode(sm.CreateUintNode(4, uint32(evtLst[k])), sm.CreateASCIINode(""), sm.CreateListNode())
            evtNodes = append(evtNodes, n)
        }
    }
    node := sm.CreateListNode(evtNodes...)
    fmt.Printf("getEventNameList : %v \n", node)
    return node
}

func (sd *SECS_DATA) getEventReport(evtID uint32, dvCtx map[uint32]interface{}) sm.ElementType {
    /*dataid is 流水號 for multiblock or 0*/
    evt_entry, ok := sd.evt[evtID]
    if !ok {
        fmt.Printf("event ID not found\n", evtID)
        return nil
    }
    if !evt_entry.enable {
        fmt.Printf("event ID %d disable\n", evtID)
        return nil
    }

    dataID_Node := sm.CreateUintNode(4, []interface{}{uint32(0)}...)
    evtID_Node := sm.CreateUintNode(4, []interface{}{uint32(evtID)}...)

    rptLst := make([]interface{}, 0)
    for i := 0; i < len(evt_entry.rptLst); i++ {
        rptId := evt_entry.rptLst[i]
        rptID_Node := sm.CreateUintNode(4, uint32(rptId))
        rpt_entry, ok := sd.rpt[rptId]
        if !ok {
            fmt.Printf("RPT ID not found\n", rptId)
            return nil
        }
        vidLst := make([]interface{}, 0)
        for j := 0; j < len(rpt_entry.vids); j++ {
            vid := rpt_entry.vids[j]
            value, ok := dvCtx[vid]
            if !ok {
                ok, value, _, _, _, _ := sd.getVidElementType(vid)
                if !ok || value == nil {
                    fmt.Printf("VID not found\n", vid)
                    return nil
                }
                vidLst = append(vidLst, value)
            } else {
                vidLst = append(vidLst, value)
            }
        }
        rptLst = append(rptLst, sm.CreateListNode(rptID_Node, sm.CreateListNode(vidLst...)))
    }
    rootNode := sm.CreateListNode(dataID_Node, evtID_Node, sm.CreateListNode(rptLst...))
    fmt.Printf("getEventReport : %v \n", rootNode)
    return rootNode
}

func (sd *SECS_DATA) getRptReport(rptID uint32) sm.ElementType {
    rpt_entry, ok := sd.rpt[rptID]
    if !ok {
        fmt.Printf("RPT ID not found\n", rptID)
        return nil
    }
    vidLst := make([]interface{}, 0)
    for j := 0; j < len(rpt_entry.vids); j++ {
        vid := rpt_entry.vids[j]
        ok, value, _, _, _, _ := sd.getVidElementType(vid)
        if !ok || value == nil {
            fmt.Printf("VID not found\n", vid)
            return nil
        }
        vidLst = append(vidLst, value)
    }
    rootNode := sm.CreateListNode(vidLst...)
    fmt.Printf("getRptReport : %v \n", rootNode)
    return rootNode
}

func (sd *SECS_DATA) getSVElementTypeLst(svidLst []uint32) sm.ElementType {
    if len(svidLst) == 0 { // select all
        for k := range sd.svs {
            svidLst = append(svidLst, k)
        }
        sort.Slice(svidLst, func(i, j int) bool { return svidLst[i] < svidLst[j] })
    }

    svNodeLst := make([]interface{}, 0)
    for k := 0; k < len(svidLst); k++ {
        svID := svidLst[k]
        if ok, node, _, _, _, _ := sd.getVidElementType(uint32(svID)); ok {
            svNodeLst = append(svNodeLst, node)
        } else {
            svNodeLst = append(svNodeLst, sm.CreateListNode())
        }
    }
    rootNode := sm.CreateListNode(svNodeLst...)
    fmt.Printf("getSVElementTypeLst : %v \n", rootNode)
    return rootNode
}

func (sd *SECS_DATA) getSVNameLst(svidLst []uint32) sm.ElementType {
    if len(svidLst) == 0 { // select all
        for k := range sd.svs {
            svidLst = append(svidLst, k)
        }
        sort.Slice(svidLst, func(i, j int) bool { return svidLst[i] < svidLst[j] })
    }
    svNodeLst := make([]interface{}, 0)
    for k := 0; k < len(svidLst); k++ {
        svID := svidLst[k]
        id := sm.CreateUintNode(4, svID)
        if ok, v := sd.getVidVariable(uint32(svID)); ok {
            name := sm.CreateASCIINode(v.name)
            units := sm.CreateASCIINode(v.units)
            node := sm.CreateListNode(id, name, units)
            svNodeLst = append(svNodeLst, node)
        } else {
            name := sm.CreateASCIINode("")
            units := sm.CreateASCIINode("")
            node := sm.CreateListNode(id, name, units)
            svNodeLst = append(svNodeLst, node)
        }
    }
    rootNode := sm.CreateListNode(svNodeLst...)
    fmt.Printf("getSVNameLst : %v \n", rootNode)
    return rootNode
}

func (sd *SECS_DATA) setEC(ecs map[uint32]interface{}) int {
    for k, v := range ecs {
        ec, ok := sd.ecs[k]
        if !ok {
            fmt.Printf("ECID : %d not exist\n", k)
            return 1 // one or more constants does not exist
        }
        if ec.value.(sm.ElementType).Type() != v.(sm.ElementType).Type() {
            fmt.Printf("ECID : %d Type mismatch\n", k)
            return 3 // one or more values out of range
        }
        if (ec.value.(sm.ElementType).Type() != "A") && (ec.value.(sm.ElementType).Size() != v.(sm.ElementType).Size()) {
            fmt.Printf("ECID : %d Size mismatch\n", k)
            return 3 // one or more values out of range
        }
    }

    for k, v := range ecs {
        sd.ecs[k].value = v
        fmt.Printf("setup%d %v\n", k, v)
    }
    return 0 // ok
}

func (sd *SECS_DATA) getEC(ecLst []uint32) sm.ElementType {
    if len(ecLst) == 0 { // select all
        for k := range sd.ecs {
            ecLst = append(ecLst, k)
        }
        sort.Slice(ecLst, func(i, j int) bool { return ecLst[i] < ecLst[j] })
    }

    ecNodeLst := make([]interface{}, 0)
    for k := 0; k < len(ecLst); k++ {
        ecID := ecLst[k]
        ec, ok := sd.ecs[ecID]
        if ok {
            ecNodeLst = append(ecNodeLst, ec.value)
        } else {
            ecNodeLst = append(ecNodeLst, sm.CreateListNode())
        }
    }
    rootNode := sm.CreateListNode(ecNodeLst...)
    fmt.Printf("getEC : %v\n", rootNode)
    return rootNode
}

func (sd *SECS_DATA) getECName(ecLst []uint32) sm.ElementType {
    if len(ecLst) == 0 { // select all
        for k := range sd.ecs {
            ecLst = append(ecLst, k)
        }
        sort.Slice(ecLst, func(i, j int) bool { return ecLst[i] < ecLst[j] })
    }

    ecNodeLst := make([]interface{}, 0)
    for k := 0; k < len(ecLst); k++ {
        ecID := ecLst[k]
        ec, ok := sd.ecs[ecID]
        if ok {
            var ecNode sm.ElementType
            ecNode = sm.CreateListNode(
                sm.CreateUintNode(4, ecID),
                sm.CreateASCIINode(ec.name),
                ec.max, ec.min,
                ec.defv, sm.CreateASCIINode(ec.units),
            )
            ecNodeLst = append(ecNodeLst, ecNode)
        } else {
            ecNode := sm.CreateListNode(
                sm.CreateUintNode(4, ecID),
                sm.CreateASCIINode(""),
                sm.CreateASCIINode(""),
                sm.CreateASCIINode(""),
                sm.CreateASCIINode(""),
                sm.CreateASCIINode(""),
            )
            ecNodeLst = append(ecNodeLst, ecNode)
        }
    }
    rootNode := sm.CreateListNode(ecNodeLst...)
    fmt.Printf("getECName : %v \n", rootNode)
    return rootNode
}

func (sd *SECS_DATA) setAlarmEnable(alid uint64, aled int) int {
    ret := 1
    for k, alarm := range sd.alarm {
        if k == uint32(alid) || alid == uint64(0xFFFFFFFFFFFFFFFF) {
            if aled == 128 {
                alarm.enable = true
                fmt.Printf("set %v %v enable\n", alid, aled)
            } else if aled == 0 {
                alarm.enable = false
                fmt.Printf("set %v %v disable\n", alid, aled)
            }
            if k == uint32(alid) {
                ret = 0
                break
            }
        }
    }
    if alid == uint64(0xFFFFFFFFFFFFFFFF) {
        ret = 0
    }
    return ret
}

func (sd *SECS_DATA) getAlarmsLst(alids []uint64) sm.ElementType {
    if len(alids) == 0 {
        for k := range sd.alarm {
            alids = append(alids, uint64(k))
        }
    }
    nodeLst := make([]interface{}, 0)
    for i := 0; i < len(alids); i++ {
        alarmid := uint32(alids[i])
        alarm, ok := sd.alarm[alarmid]
        alidNode := sm.CreateUintNode(4, alarmid)
        var alcdNode sm.ElementType
        var textNode sm.ElementType
        if ok {
            if alarm.set == true {
                alcdNode = sm.CreateBinaryNode([]interface{}{byte(128)}...)
            } else {
                alcdNode = sm.CreateBinaryNode([]interface{}{byte(0)}...)
            }
            textNode = sm.CreateASCIINode(alarm.text)
        } else {
            alcdNode = sm.CreateBinaryNode()
            textNode = sm.CreateASCIINode("")
        }
        obj := sm.CreateListNode(alcdNode, alidNode, textNode)
        nodeLst = append(nodeLst, obj)
    }
    rootNode := sm.CreateListNode(nodeLst...)
    return rootNode
}

func (sd *SECS_DATA) setAlarm(alid uint64, alcd int) (uint32, bool) {
    alarm, ok := sd.alarm[uint32(alid)]
    if ok {
        if alcd >= 128 {
            alarm.set = true
        } else {
            alarm.set = false
        }
        return alarm.evt, true
    }
    return 0, false
}

func (sd *SECS_DATA) getDvbyName(namelist []string) []uint32 {
    vidList := make([]uint32, 0)
    for _, name := range namelist {
        for id, dv := range sd.dvs {
            if dv.name == name {
                vidList = append(vidList, id)
            }
        }
    }
    return vidList
}
