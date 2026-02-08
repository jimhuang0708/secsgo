const MIN = 0;
const MAX = 100;
const params = {
    temperature: { current:50, l1Lower:10, l1Upper:15, l2Lower:85, l2Upper:90 },
    rpm:  { current:50, l1Lower:10, l1Upper:15, l2Lower:85, l2Upper:90 },
    psi:  { current:50, l1Lower:10, l1Upper:15, l2Lower:85, l2Upper:90 }
};
let ws = null;

function isValidIPPort(s) {
  const parts = s.split(":");
  if (parts.length !== 2) return false;

  const [ip, portStr] = parts;

  // IPv4 check
  const ipParts = ip.split(".");
  if (ipParts.length !== 4) return false;

  for (const p of ipParts) {
    if (!/^\d+$/.test(p)) return false;
    const n = Number(p);
    if (n < 0 || n > 255) return false;
  }

  // Port check
  if (!/^\d+$/.test(portStr)) return false;
  const port = Number(portStr);
  if (port < 1 || port > 65535) return false;

  return true;
}


function updateStatus(text) {
    document.getElementById('wsState').textContent = text;
}

function wsSend(str) {
    //  Convert string to UTF-8 bytes
    const encoder = new TextEncoder();
    const strBytes = encoder.encode(str);
    const length = strBytes.length;

    // Create buffer: 4 bytes for length + string bytes
    const buffer = new ArrayBuffer(4 + length);
    const view = new DataView(buffer);

    // Write length as 32-bit unsigned integer (big endian)
    view.setUint32(0, length, false); // false for big-endian

    // Copy string bytes after length
    new Uint8Array(buffer, 4).set(strBytes);

    // Send the buffer
    ws.send(buffer);
}

function sendHostEvent(type , payload) {
    const message = {
        type: type,
        data: payload,
        timestamp: new Date().toISOString()
    };
    if (ws && ws.readyState === WebSocket.OPEN) {
        wsSend(JSON.stringify(message))
        console.log("Sent:", message);
    } else {
        console.warn("WebSocket not open, cannot send:", message);
    }
}


function initWebSocket() {
    const mode = document.querySelector('input[name="mode"]:checked').value;
    const remoteip = document.getElementById("remoteip").value
    if( !isValidIPPort(remoteip)){
        alert("invalid ip");
        return
    }


    ws = new WebSocket("ws://" + location.hostname  + ":" + location.port + "/api/host?mode=" + mode + "&remoteip=" + remoteip);
    ws.onopen = function () {
        console.log("WebSocket connected");
        updateStatus("Connected");
        document.getElementById("btnConnect").innerText= "Disconnect"
    };

    ws.onclose = function () {
        console.log("WebSocket closed");
        updateStatus("Closed");
        document.getElementById("btnConnect").innerText= "Connect"
    };

    ws.onerror = function (err) {
        console.error("WebSocket error:", err);
        updateStatus("Error");
        document.getElementById("btnConnect").innerText= "Disconnect"
    };

    ws.onmessage = function (event) {
        console.log("Received from server:", event.data);
        obj = JSON.parse(event.data)
        console.log(obj)
        if(obj["evttype"] == "Packet"){
            obj = obj["data"]
            document.getElementById("sv-textarea").value = document.getElementById("sv-textarea").value + '\n\n' + obj.timestamp + " " + obj.msgtype + " : "  + obj.sml
        }
        if(obj["evttype"] == "disconnect"){
             ws.close()
             alert("HOST closed");
        }

        if(obj["evttype"] == "S10F1"){
            document.getElementById("chatbox").innerText = obj["data"]
        }

        if(obj["evttype"] == "Disconnect"){
            ws.close()
            alert("Remote equipment not exist(active) or local listen socket create failed(passive)!");
        }


    };
}

// 綁定 UI 事件
function bindEvents() {
    // 第二組：四個按鈕
    document.getElementById("btnOnlineRequest").addEventListener("click", function () {
        dataitem = {}
        cmd = { "stream" : 1 , "function" : 17 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))

    });

    document.getElementById("btnOfflineRequest").addEventListener("click", function () {
        dataitem = {}
        cmd = { "stream" : 1 , "function" : 15 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });

    //
    document.getElementById("cleartext").addEventListener("click", function () {
        document.getElementById("sv-textarea").value = ""
    });

    document.getElementById("s1f3").addEventListener("click", function () {
        dataitem = { "type": "L", "items": [] }
        let vidList = document.getElementById("s1f3_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.items.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }
        cmd = { "stream" : 1 , "function" : 3 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });

    document.getElementById("s1f11").addEventListener("click", function () {
        dataitem = { "type": "L", "items": [] }
        let vidList = document.getElementById("s1f11_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.items.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }

        cmd = { "stream" : 1 , "function" : 11 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });


    document.getElementById("s2f13").addEventListener("click", function () {
        dataitem = { "type": "L", "items": [] }
        let vidList = document.getElementById("s2f13_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.items.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }

        cmd = { "stream" : 2 , "function" : 13 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });

    document.getElementById("s2f15").addEventListener("click", function () {
        dataitem = { "type": "L", "items": [] }
        let pairList = document.getElementById("s2f15_param").value.split(",")
        for(let i = 0 ; i < pairList.length ; i++){
            if( pairList[i] == "" ){
                alert("empty not allowed")
                return;
            }
            let pair = pairList[i].split("=")
            dataitem.items.push( { "type" : "L", "items" :[ { "type" : "U4" , "values" : [   parseInt(pair[0],10) ] } , { "type" : "U4" , "values" : [   parseInt(pair[1],10) ] } ] } )
        }
        cmd = { "stream" : 2 , "function" : 15 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });

    document.getElementById("s2f29").addEventListener("click", function () {
        dataitem = { "type": "L", "items": [ ]  }
        let vidList = document.getElementById("s2f29_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.items.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }

        cmd = { "stream" : 2 , "function" : 29 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });



    document.getElementById("s10f3").addEventListener("click", function () {
        textcontent = document.getElementById("sendtextcontent").value
        dataitem = { "type": "L", "items": [  { "type": "B", "bytes": "00" }, { "type": "A", "value": textcontent } ] }
        cmd = { "stream" : 10 , "function" : 3 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });



    document.getElementById("s2f33").addEventListener("click", function () {


        let defrptid = parseInt( document.getElementById("defrptid").value  ,10)
        let defrptvidlst = document.getElementById("defrptvidlst").value.split(",")

        dataitem =  {
            "type": "L",
            "items": [
                {
                    "type": "U4",
                    "values": [
                        0
                    ]
                },
                {
                    "type": "L",
                    "items": [
                        {
                            "type": "L",
                            "items": [
                                {
                                    "type": "U4",
                                    "values": [
                                        defrptid
                                    ]
                                },
                                {
                                    "type": "L",
                                    "items": []
                                }
                            ]
                        }
                    ]
                }
            ]
        }
        for (let j = 0 ; j < defrptvidlst.length;j++){
            dataitem.items[1].items[0].items[1].items.push( { "type" : "U4" , "values" : [ parseInt(defrptvidlst[j],10) ] })
        }
        cmd = { "stream" : 2 , "function" : 33 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))

    });
    document.getElementById("s2f35").addEventListener("click", function () {
        let linkevtid = parseInt( document.getElementById("linkevtid").value  ,10)
        let linkrptid = parseInt( document.getElementById("linkrptid").value  ,10)
        dataitem = {
            "type": "L",
            "items": [
                {
                    "type": "U4",
                    "values": [
                        0
                    ]
                },
                {
                    "type": "L",
                    "items": [
                        {
                            "type": "L",
                            "items": [
                                {
                                    "type": "U4",
                                    "values": [
                                        linkevtid
                                    ]
                                },
                                {
                                    "type": "L",
                                    "items": [
                                        {
                                            "type": "U4",
                                            "values": [
                                                linkrptid
                                            ]
                                        }
                                    ]
                                }
                            ]
                        }
                    ]
                }
            ]
        }
        cmd = { "stream" : 2 , "function" : 35 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });



    document.getElementById("s2f41").addEventListener("click", function () {
        dataitem = {
            "type": "L",
            "items": [    {   "type": "A", "value": "HostCallFunctionA" },
                          {
                              "type": "L",
                              "items":    [
                                              {
                                                   "type": "L",
                                                   "items": [
                                                               {  "type": "A", "value": "Parameter1" },
                                                               {  "type": "A", "value": "Value1" } ]
                                              },
                                              {
                                                  "type": "L",
                                                  "items": [
                                                               { "type": "A", "value": "Parameter2" },
                                                               { "type": "A", "value": "Value2" } ]
                                              },
                                              {
                                                  "type": "L",
                                                  "items": [
                                                                { "type": "A", "value": "Parameter3" },
                                                                { "type": "A", "value": "Value3" }  ]
                                              }

                                          ]
                          }
                     ]
        }
        cmd = { "stream" : 2 , "function" : 41 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });

    document.getElementById("s2f45").addEventListener("click", function () {
        dataitem = {
                "type": "L",
                "items": [
                    {
                        "type": "U4",
                        "values": [
                            0
                        ]
                    },
                    {
                        "type": "L",
                        "items": [
                            {
                                "type": "L",
                                "items": [
                                    {
                                        "type": "U4",
                                        "values": [
                                            6
                                        ]
                                    },
                                    {
                                        "type": "L",
                                        "items": [
                                            {
                                                "type": "L",
                                                "items": [
                                                    {
                                                        "type": "B",
                                                        "bytes": "00"
                                                    },
                                                    {
                                                        "type": "L",
                                                        "items": [
                                                            {
                                                                "type": "U4",
                                                                "values": [
                                                                    55
                                                                ]
                                                            },
                                                            {
                                                                "type": "U4",
                                                                "values": [
                                                                    44
                                                                ]
                                                            }
                                                        ]
                                                    }
                                                ]
                                            }
                                        ]
                                    }
                                ]
                            }
                        ]
                    }
                ]
        }
        cmd = { "stream" : 2 , "function" : 45 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });


    document.getElementById("s2f47").addEventListener("click", function () {
        dataitem = { "type": "L", "items": [
                                               { "type" : "U4" , "values" : [ 1006 ]  },
                                               { "type" : "U4" , "values" : [ 1007 ]  },
                                           ]
                   }
        cmd = { "stream" : 2 , "function" : 47 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });



    document.getElementById("s2f23").addEventListener("click", function () {
        dataitem = {
            "type": "L",
            "items": [ ]
        }
        let trid = document.getElementById("s2f23_trid").value
        dataitem.items.push( { "type" : "U4" ,  "values" : [ parseInt(trid,10) ] } )
        let dsper = document.getElementById("s2f23_dsper").value
        dataitem.items.push( { "type" : "A" ,  "value" : dsper } )
        let totsmp = document.getElementById("s2f23_totsmp").value
        dataitem.items.push( { "type" : "U4" ,  "values" : [ parseInt(totsmp,10) ] } )
        let repgsz = document.getElementById("s2f23_repgsz").value
        dataitem.items.push( { "type" : "U4" ,  "values" : [ parseInt(repgsz,10) ] } )
        let vidList = document.getElementById("s2f23_vids").value.split(",")
        jvidLst = { "type": "L" , "items": [] }
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            jvidLst.items.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }
         dataitem.items.push( jvidLst )

        cmd = { "stream" : 2 , "function" : 23 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd))
    });
    document.getElementById("readeq").addEventListener("click", function () {
        sendHostEvent("readeq" , "storage/equipment.ds")
    });
    document.getElementById("writeeq").addEventListener("click", function () {
        sendHostEvent("writeeq" , "storage/host.ds")
    });

    document.getElementById("btnConnect").addEventListener("click", function () {
        if(this.innerText == "Connect"){
            initWebSocket()
        }
        if(this.innerText == "Disconnect"){
            ws.close()
        }
    });

}


window.addEventListener("load", function () {
    updateStatus("Connecting…");
    document.getElementById("remoteip").value = window.location.hostname + ":5000" ;
    initWebSocket();
    bindEvents();
});

