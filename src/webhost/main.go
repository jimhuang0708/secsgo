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
)

type WsConn struct {
        id         string
        addr       string
        ws         *websocket.Conn
        recvBuf *bytes.Buffer
        run        bool
}

var hostLog *logger.Logger = nil

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
        conn.run = false
        conn.recvBuf = &bytes.Buffer{}
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


func wsHost(c *gin.Context) {
    wsConn , err := UpgradeGinWsConn(c)
    if err != nil {
        hostLog.Printf("WebSocket error: %v\n", err)
        return
    }
    //browser refresh,1 second delay to wait previous listen socket close
    time.Sleep(1000 * time.Millisecond)
    ctxLog := hostLog.With("IP", wsConn.addr)
    hc := secs.NewHostContext( 0 , "ACTIVE" , ":5000" ,ctxLog )

    defer func(){
        wsConn.run = false
        wsConn.ws.Close()
    }()

    go wsConn.readHost(hc)
    wsConn.readWebSocket(c,hc)
    hc.StateStop()
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


func (conn *WsConn) readHost(hc *secs.HostContext){
    conn.run = true
    for conn.run {
        if s, ok := hc.GetUIEvt(); ok {
            err := conn.ws.WriteMessage(websocket.TextMessage, []byte(s))
            if err != nil {
                hostLog.Printf("ws write error: %v\n", err)
                return
            }
        } else {
            time.Sleep(100 * time.Millisecond)
        }
    }
}



func (conn *WsConn) readWebSocket(ctx context.Context,hc *secs.HostContext) {
    defer func() {
        //errCh <- "readWebSocket Exit"
        hostLog.Printf("readWebSocket exit\n")
    }()

    conn.ws.SetReadLimit(1024 * 1024)

    for {
        // Read next WebSocket frame and append to buffer
        if err := conn.FillBuffer(); err != nil {
            hostLog.Println("WebSocket read error:", err)
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
            hostLog.Println("ReadWS error:", err)
            break
        }

        // Successfully got a full message
        var genericData map[string]interface{}
	err = json.Unmarshal([]byte(msg[4:]), &genericData)
	if err != nil {
            hostLog.Println("Error unmarshalling JSON to map:", err)
	    return
	}
        hostLog.Printf("%v\n",genericData);
        TypeStr := genericData["type"].(string)
        if( TypeStr == "sxfy"){
            data := genericData["data"].(string)
            hc.PutUICmd(string(data))
        }
        if( TypeStr == "readeq"){
            dsName := genericData["data"].(string)
            hc.ReadEq(dsName)

        }
        if( TypeStr == "writeeq"){
            dsName := genericData["data"].(string)
            hc.WriteEq(dsName)
        }
        hostLog.Printf("%s\n",string(msg));
    }
}

func main() {
    h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ Level: slog.LevelDebug, }) // h is slog.Handler
    l := slog.New(h).With("module", "HostMain") // *slog.Logger
    hostLog = logger.New(l)
    router := gin.Default()
    router.Static("/site", "/srv/secs/")
    router.GET("/api/host", wsHost);
    router.Run(":8090")
}
