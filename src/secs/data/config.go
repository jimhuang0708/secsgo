package data
import (
    //"fmt"
    "github.com/spf13/viper"
    "reflect"
    //"encoding/json"
    //"encoding/hex"
    "fmt"
    //"os"
    sm "secs/secs_message"
    "strconv"
)

type DEFAULT_STATE struct{
    DEFAULT_CTRLSTATE string
    DEFAULT_CTRLSUBSTATE string
    DEFAULT_REJECT_CTRLSUBSTATE string
    DEFAULT_ACCEPT_CTRLSUBSTATE string
    DEFAULT_COMSTATE string
}


func getType(v interface{})(string){
    t := reflect.TypeOf(v)
    if t == nil {
        return ""
    }
    return t.String()
}

var G_STATE DEFAULT_STATE


func LoadConfig() {
    viper.AddConfigPath("./configs")
    viper.SetConfigName("config") // Register config file name (no extension)
    viper.SetConfigType("json")   // Look for specific type
    viper.ReadInConfig()
    viper.SetConfigName("system")
    viper.MergeInConfig()
    G_STATE.DEFAULT_CTRLSTATE = viper.Get("DEFAULT_CTRLSTATE").(string)
    G_STATE.DEFAULT_CTRLSUBSTATE = viper.Get("DEFAULT_CTRLSUBSTATE").(string)
    G_STATE.DEFAULT_REJECT_CTRLSUBSTATE = viper.Get("DEFAULT_REJECT_CTRLSUBSTATE").(string)
    G_STATE.DEFAULT_ACCEPT_CTRLSUBSTATE = viper.Get("DEFAULT_ACCEPT_CTRLSUBSTATE").(string)
    G_STATE.DEFAULT_COMSTATE = viper.Get("DEFAULT_COMSTATE").(string)
 
}



// ----------------------------
// Shared basic structure
// ----------------------------

type BaseItem struct {
    Name     string      `json:"name"`
    Units    *string     `json:"units"`
    ID       int         `json:"id"`
    Desc     string      `json:"desc,omitempty"`
    Node     *NodeValue  `json:"nodevalue"`            // formerly "sml"
    Max      *NodeValue  `json:"max,omitempty"`
    Min      *NodeValue  `json:"min,omitempty"`
    LimitEvt *int        `json:"limitevt,omitempty"`
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

// ----------------------------
// Whole config
// ----------------------------

type SecsConfig struct {
    SysSV    []BaseItem   `json:"syssv"`
    SysDV    []BaseItem   `json:"sysdv"`
    SysEC    []BaseItem   `json:"sysec"`
    SysEvt   []EventItem  `json:"sysevt"`
    SysAlarm []AlarmItem  `json:"sysalarm"`
    SysRpt   []ReportItem `json:"sysrpt"`
}

/*func LoadConfig2(path string) (*SecsConfig, error) {
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg SecsConfig
    if err := json.Unmarshal(raw, &cfg); err != nil {
        return nil, fmt.Errorf("json unmarshal: %w", err)
    }
    fmt.Printf("%v\n",cfg);
    return &cfg, nil
}*/

func (n *NodeValue) EncodeSecs() (sm.ElementType, error) {

    // helper: convert []float64 → []interface{}
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

        vals := make([]interface{}, 0, len(n.Bytes)/2)

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
}
