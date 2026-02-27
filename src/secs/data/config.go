package data

import (
    //"fmt"
    "reflect"
    //"encoding/json"
    //"encoding/hex"
    //"os"
    //"strconv"
    "github.com/spf13/viper"
    //seclog "secs/logger"
    //sm "secs/secs_message"
)

type DEFAULT_STATE struct{
    DEFAULT_CTRLMAINSTATE string
    DEFAULT_CTRLSUBSTATE string
    DEFAULT_REJECT_CTRLSUBSTATE string
    DEFAULT_ACCEPT_CTRLSUBSTATE string
    DEFAULT_COMSTATE int
}


func getType(v interface{})(string){
    t := reflect.TypeOf(v)
    if t == nil {
        return ""
    }
    return t.String()
}

var G_STATE DEFAULT_STATE


func LoadConfigViper() {
    viper.AddConfigPath("./configs")
    viper.SetConfigName("config") // Register config file name (no extension)
    viper.SetConfigType("json")   // Look for specific type
    viper.ReadInConfig()
    viper.AddConfigPath("./configs/system")
//    viper.SetConfigName("system_sv.json")
//    viper.MergeInConfig()
//    viper.SetConfigName("system_dv.json")
//    viper.MergeInConfig()
//    viper.SetConfigName("system_ec.json")
    viper.MergeInConfig()
    viper.SetConfigName("system_evt.json")
    viper.MergeInConfig()
    viper.SetConfigName("system_rpt.json")
    viper.MergeInConfig()
    viper.SetConfigName("system_alarm.json")
    viper.MergeInConfig()

    // Merge optional custom overrides/extensions
    viper.AddConfigPath("./configs/custom")
//    viper.SetConfigName("custom_sv.json")
//    viper.MergeInConfig()
//    viper.SetConfigName("custom_dv.json")
//    viper.MergeInConfig()
//    viper.SetConfigName("custom_ec.json")
    viper.MergeInConfig()
    viper.SetConfigName("custom_evt.json")
    viper.MergeInConfig()
    viper.SetConfigName("custom_rpt.json")
    viper.MergeInConfig()
    viper.SetConfigName("custom_alarm.json")
    viper.MergeInConfig()

    G_STATE.DEFAULT_CTRLMAINSTATE = viper.Get("DEFAULT_CTRLMAINSTATE").(string)
    G_STATE.DEFAULT_CTRLSUBSTATE = viper.Get("DEFAULT_CTRLSUBSTATE").(string)
    G_STATE.DEFAULT_REJECT_CTRLSUBSTATE = viper.Get("DEFAULT_REJECT_CTRLSUBSTATE").(string)
    G_STATE.DEFAULT_ACCEPT_CTRLSUBSTATE = viper.Get("DEFAULT_ACCEPT_CTRLSUBSTATE").(string)
    G_STATE.DEFAULT_COMSTATE = viper.GetInt("DEFAULT_COMSTATE")
 
}

// ----------------------------
// Event and Alarm structures
// ----------------------------

type EventItem struct {
    Name   string `json:"name"`
    ID     int    `json:"id"`
    Rpt    []int  `json:"rpt"`
    Enable bool   `json:"enable"`
}

type AlarmItem struct {
    Name   string `json:"name"`
    ID     int    `json:"id"`
    Text   string `json:"text"`
    Evt    int    `json:"evt"`
    Enable bool   `json:"enable"`
}

type ReportItem struct {
    Name string `json:"name"`
    ID   int    `json:"id"`
    VID  []int  `json:"vid"`
}



