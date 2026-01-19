package secs

import (
    "sync"
    "time"
    "fmt"
    "secs/data"
    "secs/logger"
    sm "secs/secs_message"
)

type COMMONMODULE struct{
    iChan chan Evt
    oChan chan Evt
    run      string
    wg *sync.WaitGroup
    deviceID int
    log *logger.Logger
}

// FormatTime formats time t according to mode:
// 0 => "A:12 YYMMDDHHMMSS"
// 1 => "A:16 YYYYMMDDHHMMSScc" (cc = centiseconds, 00-99)
// 2 => "YYYY-MM-DDTHH:MM:SS.s[s]*{Z|+hh:mm|-hh:mm}" (RFC3339Nano-like)
func FormatTime(mode int, t time.Time) (string, error) {
	switch mode {
	case 0:
		// Two-digit year
		return "A:12 " + t.Format("060102150405"), nil

	case 1:
		// Centiseconds: truncate to 1/100s (not round)
		cc := (t.Nanosecond() / 1e7) % 100 // 1e7 ns = 10ms
		return fmt.Sprintf("A:16 %s%02d", t.Format("20060102150405"), cc), nil

	case 2:
		// RFC3339Nano produces variable fractional seconds and timezone like Z or +hh:mm
		// Example: 2026-01-19T12:34:56.123456789+08:00 or ...Z
		return t.Format(time.RFC3339Nano), nil

	default:
		return "", fmt.Errorf("unsupported mode: %d (expect 0,1,2)", mode)
	}
}

func NewCOMMONMODULE(deviceID int, log *logger.Logger) *COMMONMODULE {
    o := COMMONMODULE{
                         run : "stop",
                         iChan : make(chan Evt,10),
                         oChan : make(chan Evt,10 ) ,
                         wg : new(sync.WaitGroup),
                         deviceID : deviceID,
                         log: log,
                  }
    o.wg.Add(1)
    go o.stateRun()
    return &o
}

func (cm * COMMONMODULE) PutEvt(e Evt) {
    cm.iChan <- e
}

func (cm * COMMONMODULE)sendS9FX(msg *sm.DataMessage,f int){
    bin := make([]interface{}, 10)
    raw := msg.EncodeBytes();
    for i := 0 ; i < 10; i++ {
        bin[i] = raw[i+4]
    }
    errmsg := sm.CreateDataMessage( 9, f ,false, sm.CreateBinaryNode( bin... ) , cm.deviceID , 0 , msg.SourceHost() )
    act := Evt{ cmd : "send" , msg : errmsg ,ts : time.Now().Unix() }
    cm.oChan <- act
    return
}

func (cm * COMMONMODULE)handleS1F3(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        cm.log.Printf("Error S1F3 format\n")
        cm.sendS9FX(msg, 7)
        return ;
    }
    svidLst := make( []uint32 , 0  )
    for k := 0; k < item.Size() ; k++ {
        svNode , err := item.(*sm.ListNode).Get(k);
        if(svNode.Type() != "U4" || err != nil){
            cm.log.Printf("error S1F3 format\n");
            cm.sendS9FX(msg, 7)
            return;
        }
        svID := uint32(svNode.Values().([]uint64)[0]);
        svidLst = append(svidLst , svID)
    }
    rootNode := data.GetSVElementTypeLst(svidLst)
    cm.log.Printf("svLst : %v\n",rootNode);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 4, false, rootNode , cm.deviceID , msg.SystemBytes() , msg.SourceHost() ) , ts : time.Now().Unix()  }
    cm.oChan <- act
}

func (cm * COMMONMODULE)handleS1F11(msg *sm.DataMessage){
    item , err := msg.Get()
    if( item.Type() != "L" || err != nil){
        cm.log.Printf("Error S1F11 format\n")
        cm.sendS9FX(msg, 7)
        return ;
    }
    svidLst := make( []uint32 , 0  )
    for k := 0; k < item.Size() ; k++ {
        svNode , err := item.(*sm.ListNode).Get(k);
        if(svNode.Type() != "U4" || err != nil){
            cm.log.Printf("error S1F11 format\n");
            cm.sendS9FX(msg, 7)
            return;
        }
        svID := uint32(svNode.Values().([]uint64)[0]);
        svidLst = append(svidLst , svID)
    }
    rootNode := data.GetSVNameLst(svidLst)
    cm.log.Printf("svLst : %v\n",rootNode);

    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(1, 12, false, rootNode , cm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix() }
    cm.oChan <- act
}

func (cm * COMMONMODULE)handleS2F17(msg *sm.DataMessage){
    //header only
    t:= time.Now()
    timestr , _ := FormatTime( 2 , t )
    timeNode := sm.CreateASCIINode(timestr)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2,18, false,
                                     timeNode , cm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    cm.oChan <- act
}

func (cm * COMMONMODULE)handleS2F31(msg *sm.DataMessage){
    // NTP ?
}


func (cm * COMMONMODULE)processMsg(msg *sm.DataMessage)(bool){
    if(msg.StreamCode() == 1){
        if(msg.FunctionCode() == 1){
            item , err := msg.Get()
            if(err != nil || item.Type()!= "empty" ){
                cm.log.Printf("error S1F1 format\n");
                cm.sendS9FX(msg, 7)
                return true;
            }

            var node sm.ElementType
            node = sm.CreateListNode( sm.CreateASCIINode("HMITaker") ,sm.CreateASCIINode("1.0") )
            act := Evt{ cmd : "send" , msg : sm.CreateDataMessage( 1, 2, false, node , cm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
            cm.log.Printf("do On-Line Identification\n")
            cm.oChan <- act
        }

        if(msg.FunctionCode() == 3){
            cm.handleS1F3(msg)
        }
        if(msg.FunctionCode() == 11){
            cm.handleS1F11(msg)
        }
    }
    if(msg.StreamCode() == 2){
         if(msg.FunctionCode() == 17){
             cm.handleS2F17(msg)
         }
         if(msg.FunctionCode() == 31){
             cm.handleS2F31(msg)
         }

    }
    return true
}

func (cm * COMMONMODULE)processEvt(evt Evt){
    msg := evt.msg.(*sm.DataMessage)
    cm.processMsg(msg)
}

func (cm * COMMONMODULE)moduleStop(){
    cm.run = "stop"
    cm.iChan <- Evt{ cmd : "quit"}
    cm.wg.Wait()
}

func (cm * COMMONMODULE)stateRun(){
    defer cm.wg.Done()
    cm.run = "run"

    for cm.run == "run" {
        select {
            case evt := <-cm.iChan:
                if(evt.cmd == "quit"){
                    break
                }
                cm.processEvt(evt)
        }
    }
    cm.run = "stop"
    cm.log.Printf("Exit COMMONMODULE \n");
    return
}
