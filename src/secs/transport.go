package secs

import (
    "encoding/binary"
    "encoding/json"
    "io"
    "net"
    "sync"
    "time"

    "secs/logger"
    sm "secs/secs_message"
)

type Transport struct {
    BaseComponent
    Conn      net.Conn
    CloseChan chan struct{}
}


func (t *Transport)ReadFullTimeout(p []byte,ms int)(string){
    errT := t.Conn.SetReadDeadline(time.Now().Add( time.Duration(ms) * time.Millisecond))
    if errT != nil {
        t.log.Println("SetReadDeadline failed:", errT)
        return "ERROR"
    }
    _ , err := io.ReadFull(t.Conn, p ) // recv data
    if err != nil {
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            //t.log.Printf("read timeout n : %d | err :%s\n",n,err)
            return "TIMEOUT"
        } else {
            t.log.Println("read error:", err)
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
        //t.log.Printf("-> %d\n",secLen);
        msg := make([]byte,secLen)
        ret := t.ReadFullTimeout(msg,T8)
        if(ret == "OK"){
            t.log.Printf("-> %v\n",append(msgLen,msg...));
            info , _ := sm.Decode(append(msgLen,msg...))
            t.log.Printf("Get %s @transport\n",info.ToSml() )
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
        t.log.Printf("write error %s\n",err);
        return "WRITEERROR"
    }
    if(msg != nil){
        t.log.Printf("SendAct %s\n",msg.ToSml());
        item := secsObj{ SML : msg.ToSml() , MsgType : "Send" ,TimeStamp : time.Now().Format("15:04:05.000") }
        uievt := &UIEvt{ EvtType : "Packet" , Source : "Transport" , Data : item }
        jsonData, _ := json.Marshal(uievt)
        t.oChan <- Evt{ cmd : "uievent" ,msg : string(jsonData)  }
    }
    return "ACTOK"
}

func NewTransport(Conn net.Conn, log *logger.Logger)(*Transport){
    transport := &Transport{
        BaseComponent : BaseComponent{
             iChan : make(chan Evt,10),
             oChan : make(chan Evt,10 ) ,
             wg : new(sync.WaitGroup),
             run : false,
             log: log,
        },

        Conn:      Conn,
        CloseChan: make(chan struct{}),
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
    t.run = true
    for t.run == true {
        select {
            case act := <-t.iChan:
                if(act.cmd == "send" || act.cmd == "sendforce"){
                    t.log.Printf("Put %s\n", act.msg.(sm.HSMSMessage).ToSml() )
                    ret := t.SendAct(act.msg.(sm.HSMSMessage))
                    if (ret == "WRITEERROR") {
                        t.log.Println("send error:", ret)
                        return
                    }
                }
            case <-t.CloseChan:
                return
        }
    }
    t.run = false
}

func (t *Transport)StateStop() {
    t.run = false
    t.Conn.Close()
    t.wg.Wait()
    t.log.Printf("Transport Exit\n");
}
