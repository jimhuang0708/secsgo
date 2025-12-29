package main

import (
    "net/http"
    "github.com/gorilla/websocket"
    "github.com/gin-gonic/gin"
    "strings"
    "net"
//    "os"
    "fmt"
//    "os/exec"
    "time"
    "bytes"
//    "io/ioutil"
//    "strconv"
//    "io"
//    "encoding/json"
    "errors"
    "encoding/binary"
    "context"
    "secs"
//    "secs/data"
//    "strconv"
)

type WsConn struct {
        id         string
        addr       string
        ws         *websocket.Conn
        recvBuf *bytes.Buffer
        run        bool
}

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
        fmt.Printf("WebSocket error: %v\n", err)
        return
    }
    hc := secs.NewHostContext( 0 )
    evtChan := make(chan string,10)
    cmdChan := make(chan string,10)
    hc.AttachUIEvtChan(&evtChan)
    hc.AttachUICmdChan(&cmdChan)

    defer func(){
        close(evtChan)
        close(cmdChan)
        wsConn.run = false
        wsConn.ws.Close()
    }()

    go StartHostActive(hc)
    //go StartHostPassive(hc)
    go wsConn.readFromHost(&evtChan)
    wsConn.readWebSocket(c,&cmdChan)
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

func (conn *WsConn) readFromHost(evtChan *chan string){
    conn.run = true
    for conn.run {
        select {
            case s := <- *evtChan :
                err := conn.ws.WriteMessage(websocket.TextMessage, []byte(s))
                if err != nil {
                    fmt.Println("ws write error:", err)
                }
            default:
        }
        time.Sleep(100 * time.Millisecond)
    }
    fmt.Printf("Exit readFromHost()");
}

func (conn *WsConn) readWebSocket(ctx context.Context,cmdChan *chan string) {
    defer func() {
        //errCh <- "readWebSocket Exit"
        fmt.Printf("readWebSocket exit\n")
    }()

    conn.ws.SetReadLimit(1024 * 1024)

    for {
        // Read next WebSocket frame and append to buffer
        if err := conn.FillBuffer(); err != nil {
            fmt.Println("WebSocket read error:", err)
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
            fmt.Println("ReadWS error:", err)
            break
        }

        // Successfully got a full message
        /*var genericData map[string]interface{}
	err = json.Unmarshal([]byte(msg[4:]), &genericData)
	if err != nil {
		fmt.Println("Error unmarshalling JSON to map:", err)
		return
	}
        fmt.Printf("%v\n",genericData);*/
        *cmdChan <- string(msg)
        fmt.Printf("%s\n",string(msg));
    }
}

func StartHostActive(hc *secs.HostContext){
    //conn, err := net.Dial("tcp", "192.168.51.118:5000")
    conn, err := net.Dial("tcp", ":5000")
    if err != nil {
        fmt.Println("Error dialing:", err)
        return
    }

    hc.AttachSession(conn,"ACTIVE")
    for hc.GetRun() == true {
        time.Sleep(1000 * time.Millisecond)
    }
    conn.Close()
    fmt.Printf("Exit StartHostActive\n");
    return
}

func StartHostPassive(hc *secs.HostContext) {
    ln, err := net.Listen("tcp", ":5000" )
    if err != nil {
        // handle error
    }
    var conn net.Conn
    for {
        conn, err = ln.Accept()
        if err != nil {
                // handle error
            fmt.Printf("Exit StartEquipmentPassive\n");
            return
        }
        hc.AttachSession(conn,"PASSIVE")
        break //accept one connection only
    }
    ln.Close();

    for hc.GetRun() == true {
        time.Sleep(1000 * time.Millisecond)
    }
    conn.Close()
    fmt.Printf("Exit StartHostPassive\n");
    return
}


func main() {
    router := gin.Default()
    router.Static("/site", "/srv/secs/")
    router.GET("/api/host", wsHost);
    router.Run(":8090")
}
