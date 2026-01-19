package secs

import (
    "sync"
    "time"
    "fmt"
    "secs/data"
    "secs/logger"
    "os/exec"
    "regexp"
    "strings"
    "strconv"
    "errors"
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

// SyncTime parses input time string then syncs to Linux system time.
// Supported formats:
// 1) "YYMMDDHHMMSS"               (assumes local timezone)
// 2) "YYYYMMDDHHMMSScc"           (cc = centiseconds, assumes local timezone)
// 3) RFC3339/RFC3339Nano: "YYYY-MM-DDTHH:MM:SS.s[s]*{Z|+hh:mm|-hh:mm}"
//
// Notes:
// - Setting system time requires root or CAP_SYS_TIME.
// - If NTP is enabled, it may immediately adjust the time back; we best-effort disable it via timedatectl.
func SyncTime(input string) error {
	t, err := ParseSyncTime(input, time.Local)
	if err != nil {
		return err
	}

	// Most common system setters are 1-second resolution. Drop sub-second.
	t = t.Truncate(time.Second)

	// Prefer timedatectl if available.
	if td, _ := exec.LookPath("timedatectl"); td != "" {
		_ = exec.Command(td, "set-ntp", "false").Run()

		// timedatectl expects local time string: "YYYY-MM-DD HH:MM:SS"
		localStr := t.In(time.Local).Format("2006-01-02 15:04:05")
		cmd := exec.Command(td, "set-time", localStr)
		if out, e := cmd.CombinedOutput(); e == nil {
			return nil
		} else {
			// Fall back to date -s below; include timedatectl error context if date also fails.
			_ = out
		}
	}

	// Fallback: date -s "YYYY-MM-DD HH:MM:SS"
	if d, _ := exec.LookPath("date"); d != "" {
		localStr := t.In(time.Local).Format("2006-01-02 15:04:05")
		cmd := exec.Command(d, "-s", localStr)
		if out, e := cmd.CombinedOutput(); e != nil {
			return fmt.Errorf("failed to set time via date: %v, output: %s", e, strings.TrimSpace(string(out)))
		}
		return nil
	}

	return errors.New("cannot set system time: neither timedatectl nor date found in PATH")
}

// ParseSyncTime parses the three supported formats.
// - For YYMMDDHHMMSS and YYYYMMDDHHMMSScc, it assumes tz (pass time.Local typically).
// - For RFC3339/RFC3339Nano it uses the timezone embedded in the string.
func ParseSyncTime(input string, tz *time.Location) (time.Time, error) {
	s := strings.TrimSpace(input)

	// 3) RFC3339/RFC3339Nano (includes timezone)
	// We try this first if it looks ISO-like.
	if strings.Contains(s, "T") && (strings.HasSuffix(s, "Z") || strings.Contains(s, "+") || strings.LastIndex(s, "-") > strings.Index(s, "T")) {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t, nil
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, nil
		}
		// continue to numeric formats
	}

	// Normalize whitespace (though your formats likely have none)
	s = strings.Join(strings.Fields(s), "")

	// 1) YYMMDDHHMMSS (12 digits)
	re12 := regexp.MustCompile(`^\d{12}$`)
	if re12.MatchString(s) {
		yy, _ := strconv.Atoi(s[0:2])
		mo, _ := strconv.Atoi(s[2:4])
		dd, _ := strconv.Atoi(s[4:6])
		hh, _ := strconv.Atoi(s[6:8])
		mm, _ := strconv.Atoi(s[8:10])
		ss, _ := strconv.Atoi(s[10:12])

		year := twoDigitYearToYear(yy)
		return time.Date(year, time.Month(mo), dd, hh, mm, ss, 0, tz), nil
	}

	// 2) YYYYMMDDHHMMSScc (16 digits; cc=centiseconds)
	re16 := regexp.MustCompile(`^\d{16}$`)
	if re16.MatchString(s) {
		year, _ := strconv.Atoi(s[0:4])
		mo, _ := strconv.Atoi(s[4:6])
		dd, _ := strconv.Atoi(s[6:8])
		hh, _ := strconv.Atoi(s[8:10])
		mm, _ := strconv.Atoi(s[10:12])
		ss, _ := strconv.Atoi(s[12:14])
		cc, _ := strconv.Atoi(s[14:16])

		if cc < 0 || cc > 99 {
			return time.Time{}, fmt.Errorf("invalid centiseconds (cc): %d", cc)
		}
		nsec := cc * 10_000_000 // 1 centisecond = 10ms = 1e7 ns
		return time.Date(year, time.Month(mo), dd, hh, mm, ss, nsec, tz), nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %q", input)
}

// twoDigitYearToYear converts YY into a full year with a pivot.
// 00-69 => 2000-2069, 70-99 => 1970-1999
func twoDigitYearToYear(yy int) int {
	if yy >= 70 {
		return 1900 + yy
	}
	return 2000 + yy
}


// FormatTime formats time t according to mode:
// 0 => "A:12 YYMMDDHHMMSS"
// 1 => "A:16 YYYYMMDDHHMMSScc" (cc = centiseconds, 00-99)
// 2 => "YYYY-MM-DDTHH:MM:SS.s[s]*{Z|+hh:mm|-hh:mm}" (RFC3339Nano-like)
func FormatTime(mode int, t time.Time) (string, error) {
	switch mode {
	case 0:
		// Two-digit year
		return t.Format("060102150405"), nil

	case 1:
		// Centiseconds: truncate to 1/100s (not round)
		cc := (t.Nanosecond() / 1e7) % 100 // 1e7 ns = 10ms
		return fmt.Sprintf("%s%02d", t.Format("20060102150405"), cc), nil

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
    timestr , _ := FormatTime( 1 , t )
    timeNode := sm.CreateASCIINode(timestr)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2,18, false,
                                     timeNode , cm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    cm.oChan <- act
}

func (cm * COMMONMODULE)handleS2F31(msg *sm.DataMessage){
    //_ = SyncTime("260119101112")
    item , err := msg.Get()
    if( item.Type() != "A" || err != nil){
        cm.log.Printf("Error S1F11 format\n")
        cm.sendS9FX(msg, 7)
        return ;
    }
    timestr :=  item.Values().(string)
    _ = SyncTime(timestr)
    act := Evt{ cmd : "send" , msg : sm.CreateDataMessage(2,32, false,
                                     sm.CreateBinaryNode( byte(0) ) , cm.deviceID , msg.SystemBytes() , msg.SourceHost()),ts : time.Now().Unix()}
    cm.oChan <- act

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
