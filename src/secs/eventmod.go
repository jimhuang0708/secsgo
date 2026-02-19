package secs

import (
    "time"
    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

const GEM_CTRL_STATE_LOCAL  = 300

const RPT_GEM_CTRL_STATE = 400

const SV_GEM_CTRL_STATE = 3

type EVENTMODULE struct{
    BaseModule
}

func CreateEVENTMODULE( log *logger.Logger) *EVENTMODULE {
    o := EVENTMODULE{ BaseModule : CreateBaseModule(log) }
    o.wg.Add(1)
    go o.moduleRun()
    return &o
}

func (em * EVENTMODULE) PutEvt(e Evt) {
    em.iChan <- e
}

/* send by Equipment only */
func (em * EVENTMODULE)sendS1F24(msg *sm.DataMessage,evtLst []uint32){

    node := data.GetEventNameList(evtLst)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 24, false, node , -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    em.oChan <- act
}

/* send by Equipment only */
func (em * EVENTMODULE)sendS2F34(msg *sm.DataMessage,result string){
    if(result == "ok"){
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 34, false,
                    sm.CreateBinaryNode( byte(0) ) , -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }

    if(result == "duprpt"){
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 34, false,
                    sm.CreateBinaryNode( byte(3) ) , -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }

    if(result == "novid"){
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 34, false,
                    sm.CreateBinaryNode( byte(4) ), -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }

    return
}

/* send by Equipment only */
func (em * EVENTMODULE)sendS2F36(msg *sm.DataMessage,result string){
    if(result == "ok"){
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2, 36, false,
                    sm.CreateBinaryNode( byte(0) ), -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }

    if(result == "dupevt"){//duplicate evt
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 36, false,
                    sm.CreateBinaryNode( byte(3) ), -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }

    if(result == "noevt"){//invalid evnt id
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 36, false,
                    sm.CreateBinaryNode( byte(4) ), -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }

    if(result == "norpt"){//invalid rpt id
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2, 36, false,
                    sm.CreateBinaryNode( byte(5) ), -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }

    return
}

/* send by Equipment only */
func (em * EVENTMODULE)sendS2F38(msg *sm.DataMessage,result string){
    if(result == "accept"){
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 38, false,
                    sm.CreateBinaryNode( byte(0) ), -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }
    if(result == "reject"){
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2, 38, false,
                    sm.CreateBinaryNode( byte(1) ), -1 , msg.SystemBytes(), msg.SourceHost()),ts : time.Now().Unix()}
        em.oChan <- act
    }
    return
}

func (em * EVENTMODULE)sendS6F11(node sm.ElementType){
    cmd :=  "send"
    act := Evt{ cmd : cmd , msg : sm.CreateDataMessage( 6, 11, true, node ,-1 , 0 , "ALL"), ts : time.Now().Unix()   }
    em.log.Printf("send report\n")
    em.oChan <- act
}

func (em * EVENTMODULE)handleS1F23(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        em.log.Printf("Error S1F23 format\n")
        em.sendS9FX(msg,7)
        return ;
    }
    evtQueryLst := make( []uint32 , 0  )
    for k := 0; k < item.Size() ; k++ {
        child , err := item.(*sm.ListNode).Get(k)
        if(child.Type() != "U4" || err != nil){
            em.log.Printf("Error S1F23 format error\n");
            em.sendS9FX(msg,7)
            return;
        }
        evtID := uint32(child.Values().([]uint64)[0])
        if(!data.IsEvtExist(evtID)){
            em.log.Printf("Error S1F23 Event %d not exist\n",evtID);
        } else {
            em.log.Printf("S1F23 query Event %d\n",evtID);
        }
        evtQueryLst = append( evtQueryLst , evtID)
    }
    if(item.Size() == 0){
        em.log.Printf("S1F23 query all Event\n");
    }
    em.log.Printf("S1F23 %#v\n",evtQueryLst)
    em.sendS1F24(msg,evtQueryLst)
    return;
}

func (em * EVENTMODULE)handleS2F33(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type()  != "L" || item.Size() != 2 || err != nil){
        em.log.Printf("Error S2F33 format\n")
        em.sendS9FX(msg,7)
        return ;
    }
    //item.Get(0) is DATA ID ,not used now
    node , err :=  item.(*sm.ListNode).Get(1)
    if( node.Type()  != "L" || err != nil){
        em.log.Printf("node[1] should be list\n")
        em.sendS9FX(msg,7)
        return ;
    }
    markProcessRpt := make(map[uint32]bool)
    for k := 0; k < node.Size() ; k++ {
        child , err := node.(*sm.ListNode).Get(k)
        if(child.Type() != "L" || err != nil){
            em.log.Printf("child should be list but %s\n",child.Type())
            em.sendS9FX(msg,7)
            return
        }
        if( child.Size() != 2 || err != nil){
            em.log.Printf("Error S2F33 format\n")
            em.sendS9FX(msg,7)
            return
        }
        grandChild1 , _  := child.(*sm.ListNode).Get(0)
        if(grandChild1.Type() != "U4"){
            em.log.Printf("Error S2F33 format\n")
            em.sendS9FX(msg,7)
            return
        }
        rptID := uint32(grandChild1.Values().([]uint64)[0]);
        if _ , ok := markProcessRpt[rptID]; ok  {//duplicate rpt id
            em.sendS2F34(msg, "duprpt");
            return
        }

        em.log.Printf("S2F33 Define RPT ID : %d\n",rptID)
        grandChild2,_ := child.(*sm.ListNode).Get(1)
        if( grandChild2.Type() != "L" || err != nil ){
            em.log.Printf("grandchild should be list but %s\n",grandChild2.Type())
            em.sendS9FX(msg,7)
            return
        }
        if( grandChild2.Size() == 0){
            em.log.Printf("Delete  Report ID : %d\n",rptID)
            data.DeleteReport(rptID)
            continue
        } else {
            vids := make( []uint32 , 0)
            for l := 0; l < grandChild2.Size() ; l++ {
                n , err := grandChild2.(*sm.ListNode).Get(l)
                if(n.Type() != "U4" || err != nil ){
                    em.log.Printf("vid should be u4 but %s\n",n.Type())
                    em.sendS9FX(msg,7)
                    return
                }
                if( !data.IsVidExist( uint32(n.Values().([]uint64)[0]) )){
                    em.log.Printf("vid %d not exit\n", uint32(n.Values().([]uint64)[0]) )
                    em.sendS2F34(msg, "novid");
                    return
                }
                vids = append(vids,uint32(n.Values().([]uint64)[0]) )
                em.log.Printf("\tVID : %v \n", uint32(n.Values().([]uint64)[0]))
            }
            data.CreateReport( rptID ,vids...)
            markProcessRpt[rptID] = true
        }
    }

    if(node.Size() == 0){
        data.DeleteAllReport();
    }
    em.sendS2F34(msg, "ok");
}

func (em * EVENTMODULE)handleS2F35(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type()  != "L" || item.Size() != 2 || err != nil){
        em.log.Printf("Error S2F33 format\n")
        em.sendS9FX(msg,7)
        return ;
    }
    //item.Get(0) is DATA ID ,not used now
    node , err :=  item.(*sm.ListNode).Get(1)
    if( node.Type()  != "L" || err != nil){
        em.log.Printf("node[1] should be list\n")
        em.sendS9FX(msg,7)
        return ;
    }
    markProcessEvt := make(map[uint32]bool)
    for k := 0; k < node.Size() ; k++ {
        child , err := node.(*sm.ListNode).Get(k)
        if(child.Type() != "L" || err != nil){
            em.log.Printf("child should be list but %s\n",child.Type())
            em.sendS9FX(msg,7)
            return
        }
        if( child.Size() != 2 || err != nil){
            em.log.Printf("Error S2F33 format\n")
            em.sendS9FX(msg,7)
            return
        }
        grandChild1 , _  := child.(*sm.ListNode).Get(0)
        if(grandChild1.Type() != "U4"){
            em.log.Printf("Error S2F33 format\n")
            em.sendS9FX(msg,7)
            return
        }
        ceID := uint32(grandChild1.Values().([]uint64)[0]);
        if _ , ok := markProcessEvt[ceID]; ok  {//duplicate event id
            em.sendS2F36(msg,"dupevt")
            return
        }


        em.log.Printf("S2F33 Link ceID ID : %d\n",ceID)
        grandChild2,_ := child.(*sm.ListNode).Get(1)
        if( grandChild2.Type() != "L" || err != nil ){
            em.log.Printf("grandchild should be list but %s\n",grandChild2.Type())
            em.sendS9FX(msg,7)
            return
        }

        rids := make( []uint32 , 0)
        for l := 0; l < grandChild2.Size() ; l++ {
            n , err := grandChild2.(*sm.ListNode).Get(l)
            if(n.Type() != "U4" || err != nil){
                em.log.Printf("rptID  should be u4 but %s\n",n.Type())
                em.sendS9FX(msg,7)
                return
            }
            rids = append(rids,uint32(n.Values().([]uint64)[0]))
            em.log.Printf("\trptID : %v \n", uint32(n.Values().([]uint64)[0]))
        }
        ret := data.SetEvtRptLink( ceID ,rids...)
        if(ret != "ok"){
            em.sendS2F36(msg,ret)
            return
        }
        markProcessEvt[ceID] = true
    }
    em.sendS2F36(msg,"ok")


}

func (em * EVENTMODULE)handleS2F37(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type()  != "L" || item.Size() != 2 || err != nil){
        em.log.Printf("Error S2F37 format\n")
        em.sendS9FX(msg,7)
        return ;
    }
    node , err := item.(*sm.ListNode).Get(0)
    if( node.Type() != "BOOLEAN" || node.Size() !=  1 || err != nil){
        em.log.Printf("node[0] should be boolean\n")
        em.sendS9FX(msg,7)
        return ;
    }
    act := false
    if( node.Values().([]bool)[0] == true) {
        act = true
    } else {
        act = false
    }

    node , err =  item.(*sm.ListNode).Get(1)
    if( node.Type()  != "L" || err != nil){
        em.log.Printf("node[1] should be list\n")
        em.sendS9FX(msg,7)
        return ;
    }
    accept := true
    for k := 0; k < node.Size() ; k++ {
        child , err := node.(*sm.ListNode).Get(k)
        if(child.Type() != "U4" || err != nil){
            em.log.Printf("child should be U4 but %s\n",child.Type())
            em.sendS9FX(msg,7)
            return
        }
        if(!data.EnableEvent(act, uint32(child.Values().([]uint64)[0])) ){
            accept = false
            break
        }
    }
    if(node.Size() == 0){
        if(!data.EnableEvent(act)) {
            accept = false
        }
    }
    if(accept){
        em.sendS2F38(msg,"accept")
    } else {
        em.sendS2F38(msg,"reject")
    }
}

func (em * EVENTMODULE)handleS6F12(msg *sm.DataMessage){
    node , err := msg.Get()
    if( node.Type()  != "B" || err != nil || node.Size() != 1){
        em.log.Printf("handleS6F12 event id should be one u4\n")
        em.sendS9FX(msg,7)
        return ;
    }
    return
}


func (em * EVENTMODULE)handleS6F15(msg *sm.DataMessage){
    node , err := msg.Get()
    if( node.Type()  != "U4" || err != nil || node.Size() != 1){
        em.log.Printf("handleS6F15 event id should be one u4\n")
        em.sendS9FX(msg,7)
        return ;
    }
    evtID := uint32(node.Values().([]uint64)[0])
    em.log.Printf("evtID %d\n",evtID);
    rootNode := data.GetEventReport(evtID , nil )
    //em.log.Printf("rootNode : %v\n",rootNode);
    var act Evt
    if(rootNode != nil){
        act = Evt{ cmd : "send" , msg : sm.CreateDataMessage( 6, 16, false,
                rootNode , -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    } else {
        em.log.Printf("evtID %d not found\n",evtID);
        act = Evt{ cmd : "send" , msg : sm.CreateDataMessage( 6, 16, false,
                sm.CreateListNode() , -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    }
    em.oChan <- act
}

func (em * EVENTMODULE)handleS6F19(msg *sm.DataMessage){
    node , err := msg.Get()
    if( node.Type()  != "U4" || err != nil || node.Size() != 1){
        em.log.Printf("handleS6F19 event id should be one u4\n")
        em.sendS9FX(msg,7)
        return ;
    }
    rptID := uint32(node.Values().([]uint64)[0])
    em.log.Printf("rptID %d\n",rptID);
    rootNode := data.GetRptReport( rptID )
    //em.log.Printf("rootNode : %v\n",rootNode);
    var act Evt
    if(rootNode != nil){
        act = Evt{ cmd : "send" , msg : sm.CreateDataMessage( 6, 20, false,
                rootNode , -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    } else {
        em.log.Printf("rptID %d not found\n",rptID);
        act = Evt{ cmd : "send" , msg : sm.CreateDataMessage( 6, 20, false,
                sm.CreateListNode() , -1 , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    }
    em.oChan <- act
}

func (em * EVENTMODULE)processMsg(msg *sm.DataMessage)(bool){
    if( msg.MsgType() == sm.TypeDataMessage){

        if( msg.StreamCode() == 1 && msg.FunctionCode() == 23 ){
            em.handleS1F23(msg)
            return false
        }

        if( msg.StreamCode() == 2 && msg.FunctionCode() == 33 ){
            em.handleS2F33(msg)
            return false
        }

        if( msg.StreamCode() == 2 && msg.FunctionCode() == 35 ){
            em.handleS2F35(msg)
            return false
        }

        if( msg.StreamCode() == 2 && msg.FunctionCode() == 37 ){
            em.handleS2F37(msg)
            return false
        }

        if( msg.StreamCode() == 6 && msg.FunctionCode() == 12 ){
            em.handleS6F12(msg)
            return false
        }

        if( msg.StreamCode() == 6 && msg.FunctionCode() == 15 ){
            em.handleS6F15(msg)
            return false
        }

        if( msg.StreamCode() == 6 && msg.FunctionCode() == 19 ){
            em.handleS6F19(msg)
            return false
        }
    }
    return true
}

func (em * EVENTMODULE)buildEventReport(evt Evt){
    paraemeter := evt.msg.(map[string]interface{})
    evtID := paraemeter["evtid"].(uint32)
    dvCtx := paraemeter["dvctx"].(map[uint32]interface{})
    rootNode := data.GetEventReport(evtID ,dvCtx )
    //em.log.Printf("rootNode : %v\n",rootNode);
    if(rootNode != nil){
        em.sendS6F11(rootNode)
    }
    return;
}



func (em * EVENTMODULE)processEvt(evt Evt){

    if(evt.cmd == "TRIG_EVENT"){
        em.buildEventReport(evt)
        return;
    }

    msg := evt.msg.(*sm.DataMessage)
    em.processMsg(msg)
}

func (em *EVENTMODULE)moduleRun(){
    defer func(){
        em.log.Printf("Exit EVENTMODULE \n");
        em.wg.Done()
    }()
    for {
        select {
            case evt := <-em.iChan:
                em.processEvt(evt)
            case cmd :=<-em.ctrlChan:
                if(cmd == "quit"){
                    return
                }

        }
    }
    return
}
