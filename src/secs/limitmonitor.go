package secs

import (
    "fmt"
    "time"
    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

type VLAACK byte
const (
    VLAACK_OK VLAACK = iota
    VLAACK_LIMIT_ATTRIBUTE_DEFINE_ERROR
    VLAACK_CANNOT_PERFORM
)

type LVACK byte
const(
    LVACK_RESERVE LVACK = iota
    LVACK_VID_NOT_EXIST
    LVACK_LIMIT_NOT_SUPPORT_VID
    LVACK_VARIABLE_REPEAT
    LVACK_LIMIT_VALUE_ERROR
)

type LIMITACK byte
const(
    LIMITACK_RESERVE LIMITACK = iota
    LIMITACK_ID_NOT_EXIST
    LIMITACK_UPPER_OUTRANGE
    LIMITACK_LOWER_OUTRANGE
    LIMITACK_UPPER_SMALLER_THAN_LOWER
    LIMITACK_ILLEGAL_FORMAT
    LIMITACK_NOT_NUMERIC_ASCII
    LIMITACK_DUPLICATE)

type LIMITBOUND struct{
    upper interface{}
    lower interface{}
    state string
}

type LIMITTARGE struct{
    vid uint32
    lmtbounds map[uint32] *LIMITBOUND
}

type LIMITMONITORMODULE struct{
    BaseModule
    lmtWatch     map[uint32] *LIMITTARGE;
}

func CreateLIMITMONITORMODULE(log *logger.Logger) *LIMITMONITORMODULE {
    o := LIMITMONITORMODULE{   BaseModule : CreateBaseModule(log),
                               lmtWatch : make( map[uint32]*LIMITTARGE)  }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (lm * LIMITMONITORMODULE) PutEvt(e Evt) {
    lm.iChan <- e
}


func converToFloat64(n sm.ElementType)(bool,[]float64){
    ret := make ([]float64,0)
    if(n.Type() == "U1" || n.Type() == "U2" || n.Type() == "U4" || n.Type() == "U8"){
        lst := n.Values()
        for _ ,v := range  lst{
            ret = append(ret , float64(v.(uint64)))
        }

    }
    if(n.Type() == "I1" || n.Type() == "I2" || n.Type() == "I4" || n.Type() == "I8"){
        lst := n.Values()
        for _ ,v := range  lst{
            ret = append(ret , float64(v.(int64)))
        }
    }
    if(n.Type() == "F4" || n.Type() == "F8"){
        lst := n.Values()
        for _ ,v := range  lst{
            ret = append(ret , float64(v.(float64)))
        }
    }
    if(n.Type() == "BOOLEAN"){
        lst := n.Values()
        for _ ,v := range  lst{
            if v.(bool) == true {
                ret = append(ret , float64(1))
            } else {
                ret = append(ret , float64(0))
            }
        }
    }
    if(n.Type() == "B"){
        lst := n.Values()
        for _ ,v := range  lst{
            ret = append(ret , float64(v.(int)))
        }
    }

    if(n.Type() == "A"){
        return false,nil
        //str := string(n.Values().([]byte))
        //for _ ,c := range  str{
        //    ret = append(ret, byte(c))
        //}
    }
    if(n.Type() == "L"){
        return false,nil
    }
    return true,ret
}

func (lm * LIMITMONITORMODULE)TellUI(vid uint32,limitid uint32, upper float64,lower float64){
    ctx := &UIEvtCtx{ Datatype : "S2F45" , Data : fmt.Sprintf("%d:%d:%f:%f",vid,limitid ,upper,lower) }
    lm.oChan <- Evt{ cmd : "uievent" ,ctx : ctx  }
}


func (lm * LIMITMONITORMODULE)setLimits(vid uint32,lmtid uint32,upper interface{},lower interface{})(bool){
    _ , ok :=  lm.lmtWatch[vid]
    if(!ok){
        lm.lmtWatch[vid] = &LIMITTARGE{ vid : vid }
        lm.lmtWatch[vid].lmtbounds =  make( map[uint32]*LIMITBOUND)
    }
    _ , ok = lm.lmtWatch[vid].lmtbounds[lmtid]
    if(!ok){
        lm.lmtWatch[vid].lmtbounds[lmtid] = &LIMITBOUND{ upper : upper , lower : lower , state : "NOZONE" }
    } else {
        lm.lmtWatch[vid].lmtbounds[lmtid].upper = upper
        lm.lmtWatch[vid].lmtbounds[lmtid].lower = lower
    }
    return true
}

func (lm * LIMITMONITORMODULE)handleS2F45(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        lm.log.Printf("Error S2F45 format\n")
        lm.sendS9FX(msg, 7)
        return ;
    }
    if(item.Size() != 2){
        lm.log.Printf("Error S2F45 list size\n")
        lm.sendS9FX(msg, 7)
        return ;
    }
    dataidNode ,err := item.(*sm.ListNode).Get(0);
    if(dataidNode.Type() != "U4" || dataidNode.Size() != 1 || err != nil){
        lm.log.Printf("Error S2F45 dataid wrong\n")
        lm.sendS9FX(msg, 7)
        return ;
    }

    attrLst , err := item.(*sm.ListNode).Get(1);
    if(attrLst.Size() == 0 ){
        //clean all limitbound
        lm.log.Printf("Clean all limit bounds\n");
        lm.lmtWatch = make( map[uint32]*LIMITTARGE )
        replyMsg := sm.CreateDataMessage( 2, 46, false, sm.CreateListNode( sm.CreateBinaryNode( byte(VLAACK_OK) ) , sm.CreateListNode() ) , -1 , msg.SystemBytes() , msg.SourceHost())
        ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
        act := Evt{ cmd : "send" , ctx : ctx }
        lm.oChan <- act
        return
    }
    if(attrLst.Type() != "L" || err != nil){
        lm.log.Printf("Error S2F45 attrlist wrong\n")
        lm.sendS9FX(msg, 7)
        return ;
    }
    vlaack := VLAACK_OK
    rptNodes := make ( []interface{}, 0)
    for k := 0; k < attrLst.Size() ; k++ {
        attrNode , err := attrLst.(*sm.ListNode).Get(k)
        if(attrNode.Type() != "L" || attrNode.Size() != 2 || err != nil){
            lm.log.Printf("error S2F45 attrNode type error\n");
            lm.sendS9FX(msg, 7)
            return;
        }
        vidNode ,err := attrNode.(*sm.ListNode).Get(0);
        if(vidNode.Type() != "U4" || vidNode.Size() != 1 || err != nil){
            lm.log.Printf("Error S2F45 vid wrong\n")
            lm.sendS9FX(msg, 7)
            return ;
        }

        vid := vidNode.Values()[0].(uint64)
        ok , _ , maxNode , minNode , _ , _ := data.GetVidElementType( uint32(vid) )
        if(!ok ){
            lm.log.Printf("Error | vid : %d not exist\n ",vid);
            rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(LVACK_VID_NOT_EXIST) ) , sm.CreateListNode()  ) //no such vid
            rptNodes = append(rptNodes , rptNode)
            vlaack = VLAACK_LIMIT_ATTRIBUTE_DEFINE_ERROR
            continue
        }

        lm.log.Printf("vid : %d\n",vid);
        _ , max := converToFloat64( maxNode.(sm.ElementType) )
        _ , min := converToFloat64( minNode.(sm.ElementType) )
        lm.log.Printf("max : %f | min : %f \n",max[0],min[0]);



        limitLst , err :=  attrNode.(*sm.ListNode).Get(1);
        if( limitLst.Size() == 0){
            lm.log.Printf("vid : %d clean limitbounds\n",vid)
            delete (lm.lmtWatch , uint32(vid))
            continue;
        }


        if(limitLst.Type() != "L" || err != nil){
            lm.log.Printf("Error S2F45 limitlist wrong\n")
            lm.sendS9FX(msg, 7)
            return ;
        }

        for j := 0; j < limitLst.Size() ; j++ {
            lmtNode , err := limitLst.(*sm.ListNode).Get(j)
            if(lmtNode.Type() != "L" || lmtNode.Size() != 2 || err != nil){
                lm.log.Printf("error S2F45 lmtNode type error\n");
                lm.sendS9FX(msg, 7)
                return;
            }
            lmtidNode ,err := lmtNode.(*sm.ListNode).Get(0);
            if(lmtidNode.Type() != "B" || lmtidNode.Size() != 1 || err != nil){
                lm.log.Printf("Error S2F45 lmtid wrong\n")
                lm.sendS9FX(msg, 7)
                return ;
            }

            lmtid := lmtidNode.Values()[0].(uint8)
            lm.log.Printf("lmtid : %d\n",lmtid);

            boundNode , err := lmtNode.(*sm.ListNode).Get(1);
            if(boundNode.Size() == 0 ){
                lm.log.Printf("vid : %d | limitid : %d | clean limitbounds\n",vid,lmtid)
                _ , ok :=  lm.lmtWatch[uint32(vid)]
                if(ok){
                    lm.lmtWatch[uint32(vid)] = &LIMITTARGE{ vid : uint32(vid) }
                    lm.lmtWatch[uint32(vid)].lmtbounds =  make( map[uint32]*LIMITBOUND)
                    delete (lm.lmtWatch[uint32(vid)].lmtbounds , uint32(lmtid))
                }
                continue
            }

            if(boundNode.Type() != "L" || boundNode.Size() != 2 || err != nil){
                lm.log.Printf("error S2F45 boundNode type error\n");
                lm.sendS9FX(msg, 7)
                return;
            }
            upperboundNode , err := boundNode.(*sm.ListNode).Get(0);
            lowerboundNode , err := boundNode.(*sm.ListNode).Get(1);

            _ , upperbound := converToFloat64(upperboundNode)
            _ , lowerbound := converToFloat64(lowerboundNode)

            if(  lowerbound[0] > upperbound[0]  ){
                lm.log.Printf("Error | lowerbound : %d > upperbound : %d\n ",lowerbound[0] , upperbound[0]);
                lmtErrNode := sm.CreateListNode( lmtidNode , sm.CreateBinaryNode( byte(LIMITACK_UPPER_SMALLER_THAN_LOWER)) ) //UPPERDB < LOWERDB
                rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(LVACK_LIMIT_VALUE_ERROR) ) , lmtErrNode  ) //limit value error
                rptNodes = append(rptNodes , rptNode)
                vlaack = VLAACK_LIMIT_ATTRIBUTE_DEFINE_ERROR
                break;
            }
            if( upperbound[0] > max[0] ){
                lm.log.Printf("Error | upperbound : %d > max : %d\n ",upperbound[0] , max[0]);
                lmtErrNode := sm.CreateListNode( lmtidNode , sm.CreateBinaryNode( byte(LIMITACK_UPPER_OUTRANGE)) )
                rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(LVACK_LIMIT_VALUE_ERROR) ) , lmtErrNode  ) //limit value error
                rptNodes = append(rptNodes , rptNode)
                vlaack = VLAACK_LIMIT_ATTRIBUTE_DEFINE_ERROR
                break;
            }

            if( lowerbound[0] < min[0] ){
                lm.log.Printf("Error | lowerbound : %d < min : %d\n ",lowerbound[0] , min[0]);
                lmtErrNode := sm.CreateListNode( lmtidNode , sm.CreateBinaryNode( byte(LIMITACK_LOWER_OUTRANGE)) )
                rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(LVACK_LIMIT_VALUE_ERROR) ) , lmtErrNode  ) //limit value error
                rptNodes = append(rptNodes , rptNode)
                vlaack = VLAACK_LIMIT_ATTRIBUTE_DEFINE_ERROR
                break;
            }
            lm.setLimits(uint32(vid),uint32(lmtid), upperboundNode.Clone() , lowerboundNode.Clone() )
            lm.TellUI(uint32(vid),uint32(lmtid),upperbound[0] ,lowerbound[0] );
            lm.log.Printf("bound : %v | %v\n",upperboundNode,lowerboundNode);
        }
    }
    vlaackNODE :=sm.CreateBinaryNode( byte(vlaack)  )
    rptNodes = append( []interface{}{ vlaackNODE  } , rptNodes...  )
    lm.log.Printf("%v \n",vlaackNODE);
    replyMsg := sm.CreateDataMessage( 2, 46, false, sm.CreateListNode(rptNodes...) , -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    lm.oChan <- act

}

func (lm * LIMITMONITORMODULE)handleS2F47(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        lm.log.Printf("Error S2F47 format\n")
        lm.sendS9FX(msg, 7)
        return ;
    }

    if(item.Size() == 0){
        //query all
        vids := make( []interface{},0)
        for vid , _ := range lm.lmtWatch {
            vids = append(vids,sm.CreateUintNode(4,vid))
        }
        item = sm.CreateListNode( vids... )
    }

    limitNodes := make ( []interface{}, 0)
    for k := 0; k < item.Size() ; k++ {
        vidNode , err := item.(*sm.ListNode).Get(k)
        if(vidNode.Type() != "U4" || vidNode.Size() != 1 || err != nil){
            lm.log.Printf("Error S2F47 vid wrong\n")
            lm.sendS9FX(msg, 7)
            return ;
        }
        vid := vidNode.Values()[0].(uint64)
        lm.log.Printf("vid %d\n",vid);
        ok , _ , maxNode , minNode , _  , unit:= data.GetVidElementType( uint32(vid) )
        if(!ok ){
            limitNodes = append(limitNodes ,sm.CreateListNode(vidNode,sm.CreateListNode()))
            continue
        }
        _ , ok =  lm.lmtWatch[uint32(vid)]
        if(!ok){
            limitNodes = append(limitNodes ,sm.CreateListNode(vidNode,sm.CreateListNode()))
            continue
        }
        boundNodes := make ( []interface{}, 0)
        for limitid,limitbound := range lm.lmtWatch[uint32(vid)].lmtbounds {
            boudNode := sm.CreateListNode(  sm.CreateUintNode(4,limitid) , limitbound.upper ,limitbound.lower   )
            boundNodes = append(boundNodes , boudNode)
        }
        limitNode := sm.CreateListNode( sm.CreateASCIINode(unit) , maxNode , minNode , sm.CreateListNode(boundNodes...) )
        limitNodes = append( limitNodes , sm.CreateListNode( vidNode , limitNode) )
    }
    replyMsg := sm.CreateDataMessage( 2, 48, false, sm.CreateListNode(limitNodes...) ,  -1 , msg.SystemBytes() , msg.SourceHost())
    ctx := &SendCtx{ msg : replyMsg , cb : nil , timeout : 0 }
    act := Evt{ cmd : "send" , ctx : ctx }
    lm.oChan <- act
}

func (lm * LIMITMONITORMODULE)trigEvt(e uint32,dvCtx map[uint32]interface{}){
    p := &TrigerEvtCtx{ evtid : e , dvctx : dvCtx  }
    lm.oChan <- Evt{ cmd : "TRIG_EVENT" , ctx : p }
    return
}


func (lm * LIMITMONITORMODULE)doMonitor(){
    vidList := data.GetDvByName( "LM_LIMITID","LM_TRANSITION","LM_VALUE","LM_UPPER","LM_LOWER" )
    for k, _ := range lm.lmtWatch {
        //lm.log.Printf("monitor %d\n",k)
        ok , valueNode , _  , _  , evt , _ := data.GetVidElementType(k)
        if(!ok){
            lm.log.Printf("Error | no such vid %d\n",k);
            continue
        }
        _ , value_now := converToFloat64( valueNode.(sm.ElementType) )
        for limitid , v1 := range lm.lmtWatch[k].lmtbounds {
            //lm.log.Printf("bound  %d %v %v\n",limitid, v1.upper , v1.lower)
            _ , upperbound := converToFloat64( v1.upper.(sm.ElementType) )
            _ , lowerbound := converToFloat64( v1.lower.(sm.ElementType) )
            if( value_now[0] > upperbound[0] && v1.state != "ABOVELIMIT" ){
                lm.log.Printf("Evt ABOVE upperbound vid : %d | limitid : %d | upperdb : %f | lowerdb : %f | value : %f \n",k,limitid,upperbound[0],lowerbound[0],value_now[0]);
                v1.state = "ABOVELIMIT"
                dvContext := make(map[uint32]interface{})
                dvContext[ vidList[0] ] = sm.CreateUintNode( 4, limitid )
                dvContext[ vidList[1] ] = sm.CreateUintNode( 4,1) //up
                dvContext[ vidList[2] ] = sm.CreateUintNode( 4, uint32(value_now[0]) )
                dvContext[ vidList[3] ] = sm.CreateUintNode( 4, uint32(upperbound[0]))
                dvContext[ vidList[4] ] = sm.CreateUintNode( 4, uint32(lowerbound[0]))
                lm.trigEvt(evt.(uint32),dvContext)
            }

            if( value_now[0] < lowerbound[0] && v1.state != "BELOWLIMIT"){
                lm.log.Printf("Evt BELOW lowerbound vid : %d | limitid : %d | upperdb : %f | lowerdb : %f | value : %f \n",k,limitid,upperbound[0],lowerbound[0],value_now[0]);
                v1.state = "BELOWLIMIT"
                dvContext := make(map[uint32]interface{})
                dvContext[ vidList[0] ] = sm.CreateUintNode( 4, limitid )
                dvContext[ vidList[1] ] = sm.CreateUintNode( 4, 2 ) //down
                dvContext[ vidList[2] ] = sm.CreateUintNode( 4, uint32(value_now[0]))
                dvContext[ vidList[3] ] = sm.CreateUintNode( 4, uint32(upperbound[0]))
                dvContext[ vidList[4] ] = sm.CreateUintNode( 4, uint32(lowerbound[0]))
                lm.trigEvt(evt.(uint32),dvContext)
            }
        }
    }

}

func (lm * LIMITMONITORMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 2){
        if(msg.FunctionCode() == 45){
            lm.handleS2F45(msg)
        }
        if(msg.FunctionCode() == 47){
            lm.handleS2F47(msg)
        }
    }
    return true
}

func (lm * LIMITMONITORMODULE)processEvt(evt Evt){
    if(evt.cmd == "executefn"){
        fn := evt.ctx.(func())
        fn()
        return
    }
    if(evt.cmd == "recv"){
        msg := evt.ctx.(*RecvCtx).msg.(*sm.DataMessage)
        lm.processMsg(msg)
    }
}

func (lm * LIMITMONITORMODULE)stateRun(){
    defer func() {
        lm.log.Printf("Exit LIMITMONITORMODULE \n");
        lm.wg.Done()
    }()
    monitor_ticker := time.NewTicker(1*time.Second)
    for {
        select {
            case evt := <-lm.iChan:
                lm.processEvt(evt)
            case <-monitor_ticker.C:
                lm.doMonitor()
            case cmd := <-lm.ctrlChan:
                if(cmd == "quit"){
                    return
                }
        }
    }
    return
}
