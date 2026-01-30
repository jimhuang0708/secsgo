package main

import (
    "bytes"
    "context"
    "encoding/binary"
    "encoding/json"
    "errors"
    "log/slog"
    "secs/logger"
    "net"
    "net/http"
    "os"
    "strings"
    "time"
    "fmt"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "secs"
    "secs/data"
)



type WsConn struct {
    id         string
    addr       string
    ws         *websocket.Conn
    recvBuf *bytes.Buffer
    run bool 
}

var eqLog *logger.Logger = nil

var gWsUpgrader = &websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
                return true
        },
        Subprotocols: []string{
                "binary",
        },
}

func NewWsConn(ws *websocket.Conn, addr string) *WsConn {
        conn := &WsConn{}
        conn.addr = addr
        conn.ws = ws
        conn.recvBuf = &bytes.Buffer{}
        conn.run = false
        return conn
}


func UpgradeGinWsConn(c *gin.Context) (*WsConn, error) {
        return UpgradeWsConn(c.Writer, c.Request)
}

func UpgradeWsConn(w http.ResponseWriter, r *http.Request) (*WsConn, error) {
        addr := GetHttpRemoteAddr(r)
        ws, err := gWsUpgrader.Upgrade(w, r, nil)
        if err != nil {
                return nil, fmt.Errorf("upgrade %s error: %v", addr, err)
        }

        return NewWsConn(ws, addr), nil
}

func GetHttpRemoteAddr(r *http.Request) string {
        addr := func() string {
                addr := r.Header.Get("X-Real-IP")
                if addr != "" {
                        return addr
                }

                addr = r.Header.Get("X-Forwarded-For")
                if addr != "" {
                        return addr
                }

                return r.RemoteAddr
        }()

        if strings.Contains(addr, ":") {
                host, _, err := net.SplitHostPort(addr)
                if err == nil {
                        return host
                }
        }

        return addr
}


func wsEquipment(c *gin.Context) {
    wsConn , err := UpgradeGinWsConn(c)
    if err != nil {
        eqLog.Printf("WebSocket error: %v\n", err)
        return
    }
    ctxLog := eqLog.With("IP", wsConn.addr)
    ec := secs.NewEquipmentContext(0, ctxLog);
    evtChan := make(chan string,10)
    cmdChan := make(chan string,10)
    ec.AttachUIEvtChan(&evtChan)
    ec.AttachUICmdChan(&cmdChan)
    quit := make(chan struct{})
    go wsConn.readFromServer(&evtChan)
    go StartEquipmentPassive(ec,quit)
    //go StartEquipmentActive(ec)
    wsConn.readWebSocket(c,ec)
    wsConn.run = false
    ec.StateStop()
    close(quit)
    eqLog.Printf("wsEquipment Exit\n");
}

var ErrShortBuffer = errors.New("not enough data in buffer to read full message")

func (conn *WsConn) FillBuffer() error {
    if conn.recvBuf == nil {
        conn.recvBuf = &bytes.Buffer{}
    }
    messageType, data, err := conn.ws.ReadMessage()
    if err != nil {
        return err
    }

    if messageType != websocket.BinaryMessage {
        return nil // ignore non-binary frames
    }
    conn.recvBuf.Write(data)
    return nil
}


func (conn *WsConn) ReadWS() ([]byte, error) {
    if conn.recvBuf == nil {
        return nil, ErrShortBuffer
    }

    buf := conn.recvBuf

    if buf.Len() < 4 {
        // Not enough bytes for length prefix
        return nil, ErrShortBuffer
    }

    // Peek first 4 bytes to get length
    length := binary.BigEndian.Uint32(buf.Bytes()[:4])
    totalLen := int(4 + length)
    if buf.Len() < totalLen {
        // Not enough data yet for full message
        return nil, ErrShortBuffer
    }

    // Extract the full message (length prefix + payload)
    msg := buf.Bytes()[:totalLen]

    // Remove extracted bytes from buffer
    buf.Next(totalLen)

    return msg, nil
}

func (conn *WsConn) readWebSocket(ctx context.Context, ec *secs.EquipmentContext) {
    defer func() {
        //errCh <- "readWebSocket Exit"
        eqLog.Printf("readWebSocket exit\n")
    }()

    conn.ws.SetReadLimit(1024 * 1024)

    for {
        // Read next WebSocket frame and append to buffer
        if err := conn.FillBuffer(); err != nil {
            eqLog.Printf("WebSocket read error : %v\n", err)
            time.Sleep(10 * time.Millisecond)
            return
        }
        // Try to read a complete message from buffer
        msg, err := conn.ReadWS()
        if err == ErrShortBuffer {
            // Not enough data yet, wait for more frames
            time.Sleep(50 * time.Millisecond)
            continue
        } else if err != nil {
            eqLog.Printf("ReadWS error: %v\n", err)
            break
        }

        // Successfully got a full message
        var genericData map[string]interface{}
	err = json.Unmarshal([]byte(msg[4:]), &genericData)
	if err != nil {
		eqLog.Printf("Error unmarshalling JSON to map: %v\n", err)
		return
	}
        eqLog.Printf("%v\n",genericData);
        TypeStr := genericData["type"].(string)
        if( TypeStr == "mode"){
            data := genericData["data"].(map[string]interface{})["value"].(string)
            if(data == "online_request"){
                ec.Operate_Ctrl(0)
            }
            if(data == "offline"){
                ec.Operate_Ctrl(1)
            }
            if(data == "online_local"){
                ec.Operate_Ctrl(2)
            }
            if(data == "online_remote"){
                ec.Operate_Ctrl(3)
            }
        }
        if( TypeStr == "temperature"){
            v := genericData["data"].(map[string]interface{})["value"].(float64)
            //v,_ := strconv.Atoi(data);
            ec.SetVidUint(6,uint32(v))
        }
        if( TypeStr == "rpm"){
            v := genericData["data"].(map[string]interface{})["value"].(float64)
            //v,_ := strconv.Atoi(data);
            ec.SetVidUint(7,uint32(v))
        }
        if( TypeStr == "psi"){
            v := genericData["data"].(map[string]interface{})["value"].(float64)
            //v,_ := strconv.Atoi(data);
            ec.SetVidUint(8,uint32(v))
        }
        if( TypeStr == "temperature_limit"){
            limitid := genericData["data"].(map[string]interface{})["limitid"].(float64)
            upperDB := genericData["data"].(map[string]interface{})["upperdb"].(float64)
            lowerDB := genericData["data"].(map[string]interface{})["lowerdb"].(float64)
            ec.SetVidLimit( 6 ,uint32(limitid),uint32(upperDB),uint32(lowerDB))
        }
        if( TypeStr == "rpm_limit"){
            limitid := genericData["data"].(map[string]interface{})["limitid"].(float64)
            upperDB := genericData["data"].(map[string]interface{})["upperdb"].(float64)
            lowerDB := genericData["data"].(map[string]interface{})["lowerdb"].(float64)
            ec.SetVidLimit( 7 ,uint32(limitid),uint32(upperDB),uint32(lowerDB))
        }
        if( TypeStr == "psi_limit"){
            limitid := genericData["data"].(map[string]interface{})["limitid"].(float64)
            upperDB := genericData["data"].(map[string]interface{})["upperdb"].(float64)
            lowerDB := genericData["data"].(map[string]interface{})["lowerdb"].(float64)
            ec.SetVidLimit( 8 ,uint32(limitid),uint32(upperDB),uint32(lowerDB))
        }
        if( TypeStr == "communication" ) {
            v := genericData["data"].(map[string]interface{})["value"].(string)
            if( v == "disable"){
                ec.SetCommunicate(false)
            } else {
                ec.SetCommunicate(true)
            }
        }
        if( TypeStr == "sendtext" ) {
            data := genericData["data"].(map[string]interface{})["value"].(string)
            ec.SendText(data)
        }
        if( TypeStr == "recognize" ) {
            ec.SendRecognizeEvent()
        }

        if( TypeStr == "setalarm" ){
            alid := genericData["data"].(map[string]interface{})["alid"].(float64)
            alcd := genericData["data"].(map[string]interface{})["alcd"].(float64)
            ec.SetAlarm(uint64(alid),int(alcd))
        }
        if( TypeStr == "setec" ){
            s := genericData["data"].(map[string]interface{})["dataitem"].(string)
            ec.SetEC(s)
        }

    }
}

func (conn *WsConn) readFromServer(evtChan *chan string){
    conn.run = true
    for conn.run {
        select {
            case s := <- *evtChan :
                err := conn.ws.WriteMessage(websocket.TextMessage, []byte(s))
                if err != nil {
                    eqLog.Printf("ws write error: %v\n", err)
                }
            default:
        }
        time.Sleep(100 * time.Millisecond)
    }
}

func StartEquipmentActive(ec *secs.EquipmentContext){
    conn, err := net.Dial("tcp", ":5000")
    if err != nil {
        eqLog.Printf("Error dialing : %v\n", err)
        return
    }

    ec.AttachSession(conn,"ACTIVE")
    for ec.GetRun() == true {
        time.Sleep(1000 * time.Millisecond)
    }
    conn.Close()
    eqLog.Printf("Exit StartEquipmentActive\n");
    return
}

func StartEquipmentPassive(ec *secs.EquipmentContext, quit <-chan struct{}) {
    time.Sleep(1000 * time.Millisecond) //wait previous listen socket close
    ln, err := net.Listen("tcp", ":5000")
    if err != nil {
        eqLog.Printf("Error net.Listen: %v\n", err)
        return
    }
    defer ln.Close()

    go func() {
        <-quit
        eqLog.Printf("StartEquipmentPassive quit\n")
        ln.Close()
    }()

    for {
        conn, err := ln.Accept()
        if err != nil {
            eqLog.Printf("Exit StartEquipmentPassive: %v\n", err)
            return
        }
        ec.AttachSession(conn, "PASSIVE")
    }
}


func main() {
    h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ Level: slog.LevelDebug, }) // h is slog.Handler
    l := slog.New(h).With("module", "EquipmentMain") // *slog.Logger
    eqLog = logger.New(l)
    /* init data module */
    data.SetLogger(eqLog.With("module", "DATA"));
    data.LoadConfig();
    data.InitSECSData();
    //data.ModuleStop()
    router := gin.Default()
    router.Static("/site", "/srv/secs/")
    router.GET("/api/equipment", wsEquipment);
    router.Run(":8080")
}
