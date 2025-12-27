package secs
import (
    "fmt"
    "time"
    sm "secs/secs_message"
    "secs/data"
    "sync"
    "encoding/json"
)

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
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    lmtWatch     map[uint32] *LIMITTARGE;
    deviceID int
}

func NewLIMITMONITORMODULE(deviceID int) *LIMITMONITORMODULE {
    o := LIMITMONITORMODULE{
                         run : "stop",
                         iChan : make(chan Evt,10),
                         oChan : make(chan Evt,10 ) ,
                         wg : new(sync.WaitGroup),
                         lmtWatch : make( map[uint32]*LIMITTARGE),
                         deviceID : deviceID,
                  }
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
        lst := n.Values().([]uint64)
        for _ ,v := range  lst{
            ret = append(ret , float64(v))
        }

    }
    if(n.Type() == "I1" || n.Type() == "I2" || n.Type() == "I4" || n.Type() == "I8"){
        lst := n.Values().([]int64)
        for _ ,v := range  lst{
            ret = append(ret , float64(v))
        }
    }
    if(n.Type() == "F4" || n.Type() == "F8"){
        lst := n.Values().([]float64)
        for _ ,v := range  lst{
            ret = append(ret , float64(v))
        }
    }
    if(n.Type() == "BOOLEAN"){
        lst := n.Values().([]bool)
        for _ ,v := range  lst{
            if v == true {
                ret = append(ret , float64(1))
            } else {
                ret = append(ret , float64(0))
            }
        }
    }
    if(n.Type() == "B"){
        lst := n.Values().([]int)
        for _ ,v := range  lst{
            ret = append(ret , float64(v))
        }
    }

    if(n.Type() == "A"){
        return false,nil
        //str := n.Values().(string)
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
    uievt := &UIEvt{ EvtType : "S2F45" , Source : "LIMITMONITORMODULE" , Data : fmt.Sprintf("%d:%d:%f:%f",vid,limitid ,upper,lower) }
    jsonData, _ := json.Marshal(uievt)
    lm.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
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

func (lm * LIMITMONITORMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , lm.deviceID , 0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix()  }
    lm.oChan <- act
    return
}

func (lm * LIMITMONITORMODULE)handleS2F45(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        fmt.Printf("Error S2F45 format\n")
        lm.sendS9FX(msg, 7)
        return ;
    }
    if(item.Size() != 2){
        fmt.Printf("Error S2F45 list size\n")
        lm.sendS9FX(msg, 7)
        return ;
    }
    dataidNode ,err := item.(*sm.ListNode).Get(0);
    if(dataidNode.Type() != "U4" || dataidNode.Size() != 1 || err != nil){
        fmt.Printf("Error S2F45 dataid wrong\n")
        lm.sendS9FX(msg, 7)
        return ;
    }

    attrLst , err := item.(*sm.ListNode).Get(1);
    if(attrLst.Size() == 0 ){
        //clean all limitbound
        fmt.Printf("Clean all limit bounds\n");
        lm.lmtWatch = make( map[uint32]*LIMITTARGE )
        act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 46, false,
                                     sm.CreateListNode( sm.CreateBinaryNode( byte(0)  ) , sm.CreateListNode() ) ,
                                     lm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix() }
        lm.oChan <- act
        return
    }
    if(attrLst.Type() != "L" || err != nil){
        fmt.Printf("Error S2F45 attrlist wrong\n")
        lm.sendS9FX(msg, 7)
        return ;
    }
    vlaack := byte(0)
    rptNodes := make ( []interface{}, 0)
    for k := 0; k < attrLst.Size() ; k++ {
        attrNode , err := attrLst.(*sm.ListNode).Get(k)
        if(attrNode.Type() != "L" || attrNode.Size() != 2 || err != nil){
            fmt.Printf("error S2F45 attrNode type error\n");
            lm.sendS9FX(msg, 7)
            return;
        }
        vidNode ,err := attrNode.(*sm.ListNode).Get(0);
        if(vidNode.Type() != "U4" || vidNode.Size() != 1 || err != nil){
            fmt.Printf("Error S2F45 vid wrong\n")
            lm.sendS9FX(msg, 7)
            return ;
        }

        vid := vidNode.Values().([]uint64)[0]
        ok , _ , maxNode , minNode , _ , _ := data.GetVidElementType( uint32(vid) )
        if(!ok ){
            fmt.Printf("Error | vid : %d not exist\n ",vid);
            rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(1) ) , sm.CreateListNode()  ) //no such vid
            rptNodes = append(rptNodes , rptNode)
            vlaack = 1
            continue
        }

        fmt.Printf("vid : %d\n",vid);
        _ , max := converToFloat64( maxNode.(sm.ElementType) )
        _ , min := converToFloat64( minNode.(sm.ElementType) )
        fmt.Printf("max : %f | min : %f \n",max[0],min[0]);



        limitLst , err :=  attrNode.(*sm.ListNode).Get(1);
        if( limitLst.Size() == 0){
            fmt.Printf("vid : %d clean limitbounds\n",vid)
            delete (lm.lmtWatch , uint32(vid))
            continue;
        }


        if(limitLst.Type() != "L" || err != nil){
            fmt.Printf("Error S2F45 limitlist wrong\n")
            lm.sendS9FX(msg, 7)
            return ;
        }

        for j := 0; j < limitLst.Size() ; j++ {
            lmtNode , err := limitLst.(*sm.ListNode).Get(j)
            if(lmtNode.Type() != "L" || lmtNode.Size() != 2 || err != nil){
                fmt.Printf("error S2F45 lmtNode type error\n");
                lm.sendS9FX(msg, 7)
                return;
            }
            lmtidNode ,err := lmtNode.(*sm.ListNode).Get(0);
            if(lmtidNode.Type() != "B" || lmtidNode.Size() != 1 || err != nil){
                fmt.Printf("Error S2F45 lmtid wrong\n")
                lm.sendS9FX(msg, 7)
                return ;
            }

            lmtid := lmtidNode.Values().([]uint8)[0]
            fmt.Printf("lmtid : %d\n",lmtid);

            boundNode , err := lmtNode.(*sm.ListNode).Get(1);
            if(boundNode.Size() == 0 ){
                fmt.Printf("vid : %d | limitid : %d | clean limitbounds\n",vid,lmtid)
                _ , ok :=  lm.lmtWatch[uint32(vid)]
                if(ok){
                    lm.lmtWatch[uint32(vid)] = &LIMITTARGE{ vid : uint32(vid) }
                    lm.lmtWatch[uint32(vid)].lmtbounds =  make( map[uint32]*LIMITBOUND)
                    delete (lm.lmtWatch[uint32(vid)].lmtbounds , uint32(lmtid))
                }
                continue
            }

            if(boundNode.Type() != "L" || boundNode.Size() != 2 || err != nil){
                fmt.Printf("error S2F45 boundNode type error\n");
                lm.sendS9FX(msg, 7)
                return;
            }
            upperboundNode , err := boundNode.(*sm.ListNode).Get(0);
            lowerboundNode , err := boundNode.(*sm.ListNode).Get(1);

            _ , upperbound := converToFloat64(upperboundNode)
            _ , lowerbound := converToFloat64(lowerboundNode)

            if(  lowerbound[0] > upperbound[0]  ){
                fmt.Printf("Error | lowerbound : %d > upperbound : %d\n ",lowerbound[0] , upperbound[0]);
                lmtErrNode := sm.CreateListNode( lmtidNode , sm.CreateBinaryNode( byte(4)) ) //UPPERDB < LOWERDB
                rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(4) ) , lmtErrNode  ) //limit value error
                rptNodes = append(rptNodes , rptNode)
                vlaack = 1
                break;
            }
            if( upperbound[0] > max[0] ){
                fmt.Printf("Error | upperbound : %d > max : %d\n ",upperbound[0] , max[0]);
                lmtErrNode := sm.CreateListNode( lmtidNode , sm.CreateBinaryNode( byte(2)) )
                rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(4) ) , lmtErrNode  ) //limit value error
                rptNodes = append(rptNodes , rptNode)
                vlaack = 1
                break;
            }

            if( lowerbound[0] < min[0] ){
                fmt.Printf("Error | lowerbound : %d < min : %d\n ",lowerbound[0] , min[0]);
                lmtErrNode := sm.CreateListNode( lmtidNode , sm.CreateBinaryNode( byte(3)) )
                rptNode := sm.CreateListNode( sm.CreateUintNode(4,vid) , sm.CreateBinaryNode( byte(4) ) , lmtErrNode  ) //limit value error
                rptNodes = append(rptNodes , rptNode)
                vlaack = 1
                break;
            }
            lm.setLimits(uint32(vid),uint32(lmtid), upperboundNode.Clone() , lowerboundNode.Clone() )
            lm.TellUI(uint32(vid),uint32(lmtid),upperbound[0] ,lowerbound[0] );
            fmt.Printf("bound : %v | %v\n",upperboundNode,lowerboundNode);
        }
    }
    vlaackNODE :=sm.CreateBinaryNode( vlaack  )
    rptNodes = append( []interface{}{ vlaackNODE  } , rptNodes...  )
    fmt.Printf("%v \n",vlaackNODE);
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 46, false,
                                     sm.CreateListNode(rptNodes...) ,
                                     lm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    lm.oChan <- act

}

func (lm * LIMITMONITORMODULE)handleS2F47(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        fmt.Printf("Error S2F47 format\n")
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
            fmt.Printf("Error S2F47 vid wrong\n")
            lm.sendS9FX(msg, 7)
            return ;
        }
        vid := vidNode.Values().([]uint64)[0]
        fmt.Printf("vid %d\n",vid);
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

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 2, 48, false,
                                     sm.CreateListNode(limitNodes...) ,
                                     lm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    lm.oChan <- act


}

func (lm * LIMITMONITORMODULE)trigEvt(e uint32,dvCtx map[uint32]interface{}){
    p := make( map[string]interface{} )
    p["evtid"] = e
    p["dvctx"] = dvCtx
    lm.oChan <- Evt{ cmd : "TRIG_EVENT" , msg : p ,ts : time.Now().Unix() }
    return
}


func (lm * LIMITMONITORMODULE)doMonitor(){
    vidList := data.GetDvbyName( "LM_LIMITID","LM_TRANSITION","LM_VALUE","LM_UPPER","LM_LOWER" )
    for k, _ := range lm.lmtWatch {
        //fmt.Printf("monitor %d\n",k)
        ok , valueNode , _  , _  , evt , _ := data.GetVidElementType(k)
        if(!ok){
            fmt.Printf("Error | no such vid %d\n",k);
            continue
        }
        _ , value_now := converToFloat64( valueNode.(sm.ElementType) )
        for limitid , v1 := range lm.lmtWatch[k].lmtbounds {
            //fmt.Printf("bound  %d %v %v\n",limitid, v1.upper , v1.lower)
            _ , upperbound := converToFloat64( v1.upper.(sm.ElementType) )
            _ , lowerbound := converToFloat64( v1.lower.(sm.ElementType) )

            if( value_now[0] > upperbound[0] && v1.state != "ABOVELIMIT" ){
                fmt.Printf("Evt ABOVE upperbound vid : %d | limitid : %d | upperdb : %f | lowerdb : %f | value : %f \n",k,limitid,upperbound[0],lowerbound[0],value_now[0]);
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
                fmt.Printf("Evt BELOW lowerbound vid : %d | limitid : %d | upperdb : %f | lowerdb : %f | value : %f \n",k,limitid,upperbound[0],lowerbound[0],value_now[0]);
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
    msg := evt.msg.(*sm.DataMessage)
    lm.processMsg(msg)
}

func (lm * LIMITMONITORMODULE)moduleStop(){
    lm.run = "stop"
    lm.iChan <- Evt{ cmd : "quit"}
    lm.wg.Wait()
}

func (lm * LIMITMONITORMODULE)stateRun(){
    defer lm.wg.Done()
    lm.run = "run"
    monitor_ticker := time.NewTicker(1*time.Second)
    for lm.run == "run" {
        select {
            case evt := <-lm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                lm.processEvt(evt)
            case <-monitor_ticker.C:
                lm.doMonitor()
        }
    }
    lm.run = "stop"
    fmt.Printf("Exit LIMITMONITORMODULE \n");
    return
}
