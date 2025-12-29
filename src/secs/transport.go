package secs

import (
    "fmt"
    "net"
    "encoding/binary"
    "encoding/json"
    sm "secs/secs_message"
    "time"
    "io"
    "sync"
)

type Transport struct {
    Conn      net.Conn
    iChan chan Evt
    oChan chan Evt
    CloseChan chan struct{}
    wg    *sync.WaitGroup
}


func (t *Transport)ReadFullTimeout(p []byte,ms int)(string){
    errT := t.Conn.SetReadDeadline(time.Now().Add( time.Duration(ms) * time.Millisecond))
    if errT != nil {
        fmt.Println("SetReadDeadline failed:", errT)
        return "ERROR"
    }
    _ , err := io.ReadFull(t.Conn, p ) // recv data
    if err != nil {
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            //fmt.Printf("read timeout n : %d | err :%s\n",n,err)
            return "TIMEOUT"
        } else {
            fmt.Println("read error:", err)
            // some error else, do something else, for example create new conn
            return "ERROR"
        }
    }
    return "OK"
}


func (t *Transport)ReadMsg()(string,sm.HSMSMessage){
    msgLen := make([]byte,4)
    ret := t.ReadFullTimeout(msgLen,100)//wait anything
    if(ret == "OK"){
        secLen := binary.BigEndian.Uint32(msgLen)
        //fmt.Printf("-> %d\n",secLen);
        msg := make([]byte,secLen)
        ret := t.ReadFullTimeout(msg,T8)
        if(ret == "OK"){
            fmt.Printf("-> %v\n",append(msgLen,msg...));
            info , _ := sm.Decode(append(msgLen,msg...))
            fmt.Printf("Get %s @transport\n",info.ToSml() )
            if(info != nil){
                item := &secsObj{ SML : info.ToSml() , MsgType : "Receive" , TimeStamp : time.Now().Format("15:04:05.000") }
                uievt := &UIEvt{ EvtType : "Packet" , Source : "Transport" , Data : item }
                jsonData, _ := json.Marshal(uievt)
                t.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
            }

            return "READOK" , info

        } else if(ret == "TIMEOUT"){
            return "T8_TIMEOUT",nil
        } else {
            return "READERROR",nil
        }
    } else {
        if(ret == "TIMEOUT"){//EMPTY is ok
            return "READOK",nil
        } else {
            return "READERROR",nil
        }
    }
}

func (t *Transport)SendAct( msg sm.HSMSMessage)(string){
    _ , err := t.Conn.Write(msg.EncodeBytes());
    if(err != nil){
        fmt.Printf("write error %s\n",err);
        return "WRITEERROR"
    }
    if(msg != nil){
        fmt.Printf("SendAct %s\n",msg.ToSml());
        item := secsObj{ SML : msg.ToSml() , MsgType : "Send" ,TimeStamp : time.Now().Format("15:04:05.000") }
        uievt := &UIEvt{ EvtType : "Packet" , Source : "Transport" , Data : item }
        jsonData, _ := json.Marshal(uievt)
        t.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
    }


    return "ACTOK"
}

func NewTransport(Conn net.Conn)(*Transport){
    transport := &Transport{
        Conn:      Conn,
        iChan:  make(chan Evt, 64),
        oChan:  make(chan Evt, 64),
        CloseChan: make(chan struct{}),
        wg    : new(sync.WaitGroup),
    }
    transport.wg.Add(1)
    go transport.handleRead()
    transport.wg.Add(1)
    go transport.handleSend()
    return transport
}

func (t *Transport)handleRead() {
    defer func() {
        t.wg.Done()
    }()

    for {
        ret , msg := t.ReadMsg()
        if(ret == "READOK"){
            if(msg != nil){
                t.oChan <- Evt{ cmd : "recv" ,msg : msg}
            }
        } else {
            close(t.CloseChan)
            return
        }
    }
}

func (t *Transport)handleSend() {
    defer func() {
        t.wg.Done()
        t.oChan <- Evt{ cmd : "disconnect" ,msg : nil  }
    }()
    for {
        select {
            case act := <-t.iChan:
                if(act.cmd == "send" || act.cmd == "sendforce"){
                    fmt.Printf("Put %s\n", act.msg.(sm.HSMSMessage).ToSml() )
                    ret := t.SendAct(act.msg.(sm.HSMSMessage))
                    if (ret == "WRITEERROR") {
                        fmt.Println("send error:", ret)
                        return
                    }
                }
            case <-t.CloseChan:
                return
        }
    }
}

func (t *Transport)StateStop() {
    t.Conn.Close()
    t.wg.Wait()
    fmt.Printf("Transport Exit\n");
}

