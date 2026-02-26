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



//func (n *sm.Node) EncodeSecs() (sm.ElementType, error) {
//return nil,nil
/*
    toInterfaces := func(vals []float64) []interface{} {
        out := make([]interface{}, len(vals))
        for i, v := range vals {
            out[i] = v
        }
        return out
    }

    switch n.Type {

    case "L":
        lst := make([]interface{}, 0, len(n.Items))
        for _, child := range n.Items {
            item, err := child.EncodeSecs()
            if err != nil {
                return nil, err
            }
            lst = append(lst, item)
        }
        return sm.CreateListNode(lst...), nil

    // -----------------------------
    // Integer (signed)
    // -----------------------------
    case "I1":
        return sm.CreateIntNode(1, toInterfaces(n.Values)...), nil

    case "I2":
        return sm.CreateIntNode(2, toInterfaces(n.Values)...), nil

    case "I4":
        return sm.CreateIntNode(4, toInterfaces(n.Values)...), nil

    case "I8":
        return sm.CreateIntNode(8, toInterfaces(n.Values)...), nil

    // -----------------------------
    // Unsigned Integer
    // -----------------------------
    case "U1":
        return sm.CreateUintNode(1, toInterfaces(n.Values)...), nil

    case "U2":
        return sm.CreateUintNode(2, toInterfaces(n.Values)...), nil

    case "U4":
        return sm.CreateUintNode(4, toInterfaces(n.Values)...), nil

    case "U8":
        return sm.CreateUintNode(8, toInterfaces(n.Values)...), nil

    // -----------------------------
    // Float
    // -----------------------------
    case "F4":
        return sm.CreateFloatNode(4, toInterfaces(n.Values)...), nil

    case "F8":
        return sm.CreateFloatNode(8, toInterfaces(n.Values)...), nil

    // -----------------------------
    // ASCII
    // -----------------------------
    case "A":
        return sm.CreateASCIINode(n.Value), nil

    // -----------------------------
    // Binary
    // -----------------------------
    case "B":
        if n.Bytes == "" {
            return sm.CreateBinaryNode(), nil
        }

        // hex string must be even number of characters
        if len(n.Bytes)%2 != 0 {
            return nil, fmt.Errorf("invalid hex length for B: %s", n.Bytes)
        }

        vals := make([]byte, 0, len(n.Bytes)/2)

        for i := 0; i < len(n.Bytes); i += 2 {
            hexByte := n.Bytes[i : i+2]

            v, err := strconv.ParseUint(hexByte, 16, 8)
            if err != nil {
                return nil, fmt.Errorf("invalid hex '%s' in B: %v", hexByte, err)
            }

            vals = append(vals, byte(v))
        }

        return sm.CreateBinaryNode(vals...), nil

    // -----------------------------
    // Boolean
    // -----------------------------
    case "BOOLEAN":
        boolList := make([]interface{}, len(n.Bools))
        for i, v := range n.Bools {
            boolList[i] = v
        }
        return sm.CreateBooleanNode(boolList...), nil
    }

    return nil, fmt.Errorf("invalid data type: %s", n.Type)
*/
//}
