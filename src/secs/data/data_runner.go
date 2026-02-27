package data

import (
    "fmt"
    "os"
    //"strconv"
    "sync"
    //"github.com/spf13/viper"
    "secs/logger"
    "encoding/json"
    sm "secs/secs_message"
)

var log = logger.New(nil)

// SetLogger configures the logger used by the data package.
func SetLogger(l *logger.Logger) {
    if l != nil {
        log = l
    }
}

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


type VariableRoot struct {
    Variable []*SECSVARIABLE `json:"variable"`
}

func (sd *SECS_DATA)LoadSVConfig() (error) {
    raw, err := os.ReadFile("configs/custom/custom_sv.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    var cfg VariableRoot
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Variable ); i++ {
        log.Printf("customsv : %v \n", cfg.Variable[i])
        v := cfg.Variable[i]
        sd.svs[cfg.Variable[i].Id] = &SECSVARIABLE{Id : v.Id , Name: v.Name, Units: v.Units, Value: v.Value.Clone(), LimitEvt: v.LimitEvt, Max: v.Max.Clone(), Min: v.Min.Clone()}
    }
    raw, err = os.ReadFile("configs/system/system_sv.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Variable ); i++ {
        log.Printf("systemsv : %v\n", cfg.Variable[i])
        v := cfg.Variable[i]
        sd.svs[cfg.Variable[i].Id] = &SECSVARIABLE{Id : v.Id , Name: v.Name, Units: v.Units, Value: v.Value.Clone(), LimitEvt: v.LimitEvt, Max: v.Max.Clone(), Min: v.Min.Clone()}
    }
    return nil
}

func (sd *SECS_DATA)LoadDVConfig() (error) {
    raw, err := os.ReadFile("configs/custom/custom_dv.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    var cfg VariableRoot
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Variable ); i++ {
        log.Printf("customdv : %v\n", cfg.Variable[i])
        v := cfg.Variable[i]
        sd.dvs[cfg.Variable[i].Id] = &SECSVARIABLE{Id : v.Id , Name: v.Name, Units: v.Units, Value: v.Value.Clone(), LimitEvt: v.LimitEvt, Max: v.Max.Clone(), Min: v.Min.Clone()}
    }

    raw, err = os.ReadFile("configs/system/system_dv.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Variable ); i++ {
        log.Printf("systemdv : %v\n", cfg.Variable[i])
        v := cfg.Variable[i]
        sd.dvs[cfg.Variable[i].Id] = &SECSVARIABLE{Id : v.Id , Name: v.Name, Units: v.Units, Value: v.Value.Clone(), LimitEvt: v.LimitEvt, Max: v.Max.Clone(), Min: v.Min.Clone()}
    }
    return nil
}

func (sd *SECS_DATA)LoadECConfig() (error) {
    raw, err := os.ReadFile("configs/custom/custom_ec.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    var cfg VariableRoot
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Variable ); i++ {
        log.Printf("customec : %v\n", cfg.Variable[i])
        v := cfg.Variable[i]
        sd.ecs[cfg.Variable[i].Id] = &SECSVARIABLE{Id : v.Id , Name: v.Name, Units: v.Units, Value: v.Value.Clone(), Defv : v.Value.Clone() ,LimitEvt: v.LimitEvt, Max: v.Max.Clone(), Min: v.Min.Clone()}
    }

    raw, err = os.ReadFile("configs/system/system_ec.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Variable ); i++ {
        log.Printf("systemec : %v\n", cfg.Variable[i])
        v := cfg.Variable[i]
        sd.ecs[cfg.Variable[i].Id] = &SECSVARIABLE{Id : v.Id , Name: v.Name, Units: v.Units, Value: v.Value.Clone(), Defv: v.Value.Clone(), LimitEvt: v.LimitEvt, Max: v.Max.Clone(), Min: v.Min.Clone()}
    }
    return nil
}

type EvtRoot struct {
    Evt []*SECSCE `json:"evt"`
}


func (sd *SECS_DATA)LoadEVTConfig() (error) {
    raw, err := os.ReadFile("configs/custom/custom_evt.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    var cfg EvtRoot
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Evt ); i++ {
        log.Printf("customevt : %v\n", cfg.Evt[i])
        v := cfg.Evt[i]
        sd.evt[cfg.Evt[i].Id] = &SECSCE{Id: v.Id, Name: v.Name, RptLst: v.RptLst, DvLst: make([]uint32, 0), Enable: v.Enable}
    }

    raw, err = os.ReadFile("configs/system/system_evt.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Evt ); i++ {
        log.Printf("systemevt : %v\n", cfg.Evt[i])
        v := cfg.Evt[i]
        sd.evt[cfg.Evt[i].Id] = &SECSCE{Id: v.Id, Name: v.Name, RptLst: v.RptLst, DvLst: make([]uint32, 0), Enable: v.Enable}
    }
    return nil
}

type RptRoot struct {
    Rpt []*SECSRPT `json:"rpt"`
}

func (sd *SECS_DATA)LoadRPTConfig() (error) {
    raw, err := os.ReadFile("configs/custom/custom_rpt.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    var cfg RptRoot
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Rpt ); i++ {
        log.Printf("customrpt : %v\n", cfg.Rpt[i])
        v := cfg.Rpt[i]
        vids := make([]uint32, 0)
        for j := 0; j < len(v.Vids); j++ {
            vids = append(vids, uint32(v.Vids[j]))
        }
        sd.rpt[cfg.Rpt[i].Id] = &SECSRPT{Id: v.Id, Name: v.Name, Vids: vids }


    }

    raw, err = os.ReadFile("configs/system/system_rpt.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Rpt ); i++ {
        log.Printf("systemrpt : %v\n", cfg.Rpt[i])
        v := cfg.Rpt[i]
        vids := make([]uint32, 0)
        for j := 0; j < len(v.Vids); j++ {
            vids = append(vids, uint32(v.Vids[j]))
        }
        sd.rpt[cfg.Rpt[i].Id] = &SECSRPT{Id: v.Id, Name: v.Name, Vids : vids}
    }
    return nil
}

type AlarmRoot struct {
    Alarm []*SECSALARM `json:"alarm"`
}

func (sd *SECS_DATA)LoadALARMConfig() (error) {
    raw, err := os.ReadFile("configs/custom/custom_alarm.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    var cfg AlarmRoot
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Alarm ); i++ {
        log.Printf("customalarm : %v\n", cfg.Alarm[i])
        v := cfg.Alarm[i]
        sd.alarm[cfg.Alarm[i].Id] = &SECSALARM{Id: v.Id, Name: v.Name, Enable: v.Enable, Set: false, Text: v.Text, Evt: v.Evt}


    }

    raw, err = os.ReadFile("configs/system/system_alarm.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }

    for i := 0; i < len( cfg.Alarm ); i++ {
        log.Printf("systemalarm : %v\n", cfg.Alarm[i])
        v := cfg.Alarm[i]
        sd.alarm[cfg.Alarm[i].Id] = &SECSALARM{Id: v.Id, Name: v.Name, Enable: v.Enable, Set: false, Text: v.Text, Evt: v.Evt}
    }
    return nil
}

func (sd *SECS_DATA)LoadConfigBase() (error) {
    raw, err := os.ReadFile("configs/config.json")
    if err != nil {
        return  fmt.Errorf("read config: %w", err)
    }
    if err := json.Unmarshal(raw, &G_STATE); err != nil {
        return  fmt.Errorf("json unmarshal: %v", err)
    }
    log.Printf("Config : %v\n", G_STATE)
    return nil
}


func (sd *SECS_DATA)moduleLoadData() (error) {
    err := sd.LoadConfigBase()
    if(err != nil){
        return err
    }
    err = sd.LoadSVConfig()
    if(err != nil){
        return err
    }
    err = sd.LoadDVConfig()
    if(err != nil){
        return err
    }
    err = sd.LoadECConfig()
    if(err != nil){
        return err
    }
    err = sd.LoadEVTConfig()
    if(err != nil){
        return err
    }
    err = sd.LoadRPTConfig()
    if(err != nil){
        return err
    }
    err = sd.LoadALARMConfig()
    if(err != nil){
        return err
    }

    return nil
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
    log.Printf("Exit SECS_DATA\n")
    return
}
