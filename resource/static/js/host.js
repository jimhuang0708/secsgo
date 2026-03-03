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
    if(type == "sxfy"){
        $("#jsonInput").val(payload)
        $("#output").text(payload)
        $("#loadBtn").click();
    }
    if (ws && ws.readyState === WebSocket.OPEN) {
        wsSend(JSON.stringify(message,null,4))
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
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))

    });

    document.getElementById("btnOfflineRequest").addEventListener("click", function () {
        dataitem = {}
        cmd = { "stream" : 1 , "function" : 15 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });

    //
    document.getElementById("cleartext").addEventListener("click", function () {
        document.getElementById("sv-textarea").value = ""
    });

    document.getElementById("s1f3").addEventListener("click", function () {
        dataitem = { "type": "L", "values": [] }
        let vidList = document.getElementById("s1f3_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }
        cmd = { "stream" : 1 , "function" : 3 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });

    document.getElementById("s1f11").addEventListener("click", function () {
        dataitem = { "type": "L", "values": [] }
        let vidList = document.getElementById("s1f11_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }

        cmd = { "stream" : 1 , "function" : 11 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });


    document.getElementById("s2f13").addEventListener("click", function () {
        dataitem = { "type": "L", "values": [] }
        let vidList = document.getElementById("s2f13_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }

        cmd = { "stream" : 2 , "function" : 13 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });

    document.getElementById("s2f15").addEventListener("click", function () {
        dataitem = { "type": "L", "values": [] }
        let pairList = document.getElementById("s2f15_param").value.split(",")
        for(let i = 0 ; i < pairList.length ; i++){
            if( pairList[i] == "" ){
                alert("empty not allowed")
                return;
            }
            let pair = pairList[i].split("=")
            dataitem.values.push( { "type" : "L", "values" :[ { "type" : "U4" , "values" : [   parseInt(pair[0],10) ] } , { "type" : "U4" , "values" : [   parseInt(pair[1],10) ] } ] } )
        }
        cmd = { "stream" : 2 , "function" : 15 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });

    document.getElementById("s2f29").addEventListener("click", function () {
        dataitem = { "type": "L", "values": [ ]  }
        let vidList = document.getElementById("s2f29_param").value.split(",")
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }

        cmd = { "stream" : 2 , "function" : 29 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });



    document.getElementById("s10f3").addEventListener("click", function () {
        textcontent = document.getElementById("sendtextcontent").value
        dataitem = { "type": "L", "values": [  { "type": "B", "values": [0] }, { "type": "A", "values": [...textcontent] } ] }
        cmd = { "stream" : 10 , "function" : 3 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });



    document.getElementById("s2f33").addEventListener("click", function () {


        let defrptid = parseInt( document.getElementById("defrptid").value  ,10)
        let defrptvidlst = document.getElementById("defrptvidlst").value.split(",")

        dataitem =  {
            "type": "L",
            "values": [
                {
                    "type": "U4",
                    "values": [
                        0
                    ]
                },
                {
                    "type": "L",
                    "values": [
                        {
                            "type": "L",
                            "values": [
                                {
                                    "type": "U4",
                                    "values": [
                                        defrptid
                                    ]
                                },
                                {
                                    "type": "L",
                                    "values": []
                                }
                            ]
                        }
                    ]
                }
            ]
        }
        for (let j = 0 ; j < defrptvidlst.length;j++){
            dataitem.values[1].values[0].values[1].values.push( { "type" : "U4" , "values" : [ parseInt(defrptvidlst[j],10) ] })
        }
        cmd = { "stream" : 2 , "function" : 33 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))

    });
    document.getElementById("s2f35").addEventListener("click", function () {
        let linkevtid = parseInt( document.getElementById("linkevtid").value  ,10)
        let linkrptid = parseInt( document.getElementById("linkrptid").value  ,10)
        dataitem = {
            "type": "L",
            "values": [
                {
                    "type": "U4",
                    "values": [
                        0
                    ]
                },
                {
                    "type": "L",
                    "values": [
                        {
                            "type": "L",
                            "values": [
                                {
                                    "type": "U4",
                                    "values": [
                                        linkevtid
                                    ]
                                },
                                {
                                    "type": "L",
                                    "values": [
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
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });



    document.getElementById("s2f41").addEventListener("click", function () {
        dataitem = {
            "type": "L",
            "values": [    {   "type": "A", "values": [..."HostCallFunctionA"] },
                          {
                              "type": "L",
                              "values":    [
                                              {
                                                   "type": "L",
                                                   "values": [
                                                               {  "type": "A", "values": [..."Parameter1"] },
                                                               {  "type": "A", "values": [..."Value1"] } ]
                                              },
                                              {
                                                  "type": "L",
                                                  "values": [
                                                               { "type": "A", "values": [..."Parameter2"] },
                                                               { "type": "A", "values": [..."Value2"] } ]
                                              },
                                              {
                                                  "type": "L",
                                                  "values": [
                                                                { "type": "A", "values": [..."Parameter3"] },
                                                                { "type": "A", "values": [..."Value3"] }  ]
                                              }

                                          ]
                          }
                     ]
        }
        cmd = { "stream" : 2 , "function" : 41 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });

    document.getElementById("s2f45").addEventListener("click", function () {
        dataitem = {
                "type": "L",
                "values": [
                    {
                        "type": "U4",
                        "values": [
                            0
                        ]
                    },
                    {
                        "type": "L",
                        "values": [
                            {
                                "type": "L",
                                "values": [
                                    {
                                        "type": "U4",
                                        "values": [
                                            1006
                                        ]
                                    },
                                    {
                                        "type": "L",
                                        "values": [
                                            {
                                                "type": "L",
                                                "values": [
                                                    {
                                                        "type": "B",
                                                        "values": [0]
                                                    },
                                                    {
                                                        "type": "L",
                                                        "values": [
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
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });


    document.getElementById("s2f47").addEventListener("click", function () {
        dataitem = { "type": "L", "values": [
                                               { "type" : "U4" , "values" : [ 1006 ]  },
                                               { "type" : "U4" , "values" : [ 1007 ]  },
                                           ]
                   }
        cmd = { "stream" : 2 , "function" : 47 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
    });



    document.getElementById("s2f23").addEventListener("click", function () {
        dataitem = {
            "type": "L",
            "values": [ ]
        }
        let trid = document.getElementById("s2f23_trid").value
        dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(trid,10) ] } )
        let dsper = document.getElementById("s2f23_dsper").value
        dataitem.values.push( { "type" : "A" ,  "values" : [...dsper] } )
        let totsmp = document.getElementById("s2f23_totsmp").value
        dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(totsmp,10) ] } )
        let repgsz = document.getElementById("s2f23_repgsz").value
        dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(repgsz,10) ] } )
        let vidList = document.getElementById("s2f23_vids").value.split(",")
        jvidLst = { "type": "L" , "values": [] }
        for(let i = 0 ; i < vidList.length ; i++){
            if( vidList[i] == "" ){
                break;
            }
            jvidLst.values.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
        }
         dataitem.values.push( jvidLst )

        cmd = { "stream" : 2 , "function" : 23 , "dataitem" : dataitem }
        sendHostEvent("sxfy",JSON.stringify(cmd,null,4))
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

/////////////////////
const DATA_TYPES = [
    "U1","U2","U4","U8",
    "I1","I2","I4","I8",
    "A","BOOLEAN","B",
    "F4","F8",
    "L"
];

/* -----------------------------
   Node UI Builder
----------------------------- */
function createNode() {
    return $(`
        <div class="node-box">

            <div class="node-row">

                <select class="form-select type-select" style="width:150px;">
                    <option value="">--Type--</option>
                    ${DATA_TYPES.map(t=>`<option value="${t}">${t}</option>`).join("")}
                </select>

                <div class="input-area flex-grow-1"></div>

                <button class="btn btn-secondary btn-sm add-child" style="display:none;">+</button>

                <button class="btn btn-danger btn-sm remove-node">X</button>

            </div>

            <div class="children"></div>
        </div>
    `);
}

/* Init root */
$("#rootContainer").html("").append(createNode());

/* -----------------------------
   Handle type change
----------------------------- */
$(document).on("change", ".type-select", function () {
    const box = $(this).closest(".node-box");
    const area = box.find(".input-area");
    const children = box.find(".children");
    const type = $(this).val();

    area.empty();
    children.empty();
    box.find(".add-child").hide();

    if (["U1","U2","U4","U8","I1","I2","I4","I8","F4","F8"].includes(type)) {
        area.append(`<input class="form-control value-numbers" placeholder="1 2 3">`);
    }
    else if (type === "A") {
        area.append(`<input class="form-control value-ascii" placeholder="some text">`);
    }
    else if (type === "BOOLEAN") {
        area.append(`<input class="form-control value-bools" placeholder="true false">`);
    }
    else if (type === "B") {
        area.append(`<input class="form-control value-bytes" placeholder="0 128 255">`);
    }
    else if (type === "L") {
        box.find(".add-child").show();
    }
});

/* Add Child */
$(document).on("click", ".add-child", function () {
    $(this).closest(".node-box").find("> .children").append(createNode());
});

/* Remove Node */
$(document).on("click", ".remove-node", function () {
    $(this).closest(".node-box").remove();
});

/* -----------------------------
   Build JSON Recursively
----------------------------- */
function buildNode($node) {
    const type = $node.find("> .node-row .type-select").val();
    if (!type) return null;

    let obj = { type };

    if (["U1","U2","U4","U8","I1","I2","I4","I8","F4","F8"].includes(type)) {
        obj.values = $node.find(".value-numbers").val().trim().split(/\s+/).map(Number);
    }
    else if (type === "A") {
        obj.values = [...$node.find(".value-ascii").val()]
    }
    else if (type === "BOOLEAN") {
        obj.values = $node.find(".value-bools").val().trim().split(/\s+/).map(v => v==="true");
    }
    else if (type === "B") {
        //obj.values = $node.find(".value-bytes").val();
        obj.values = $node.find(".value-bytes").val().trim().split(/\s+/).map(Number);
    }
    else if (type === "L") {
        obj.values = [];
        $node.find("> .children > .node-box").each(function () {
            obj.values.push(buildNode($(this)));
        });
    }

    return obj;
}

$("#generateBtn").click(function () {
    const root = $("#rootContainer .node-box").first();
    const dataItem = buildNode(root);
    jsonStr = JSON.stringify({
        stream: parseInt($("#streamInput").val()),
        function: parseInt($("#functionInput").val()),
        dataitem: dataItem
    }, null, 4);
    $("#output").text(jsonStr);
    sendHostEvent("sxfy",jsonStr)

});

/* -----------------------------
   JSON → UI Loader
----------------------------- */
function loadNodeToUI(nodeData, container) {
    const node = createNode();
    container.append(node);

    // set type
    const typeSelect = node.find("> .node-row .type-select");
    typeSelect.val(nodeData.type).trigger("change");

    const area = node.find(".input-area");

    if (nodeData.type === "A") {
        area.find(".value-ascii").val(nodeData.values.join(""));
    }
    else if (["U1","U2","U4","U8","I1","I2","I4","I8","F4","F8"].includes(nodeData.type)) {
        area.find(".value-numbers").val((nodeData.values || []).join(" "));
    }
    else if (nodeData.type === "BOOLEAN") {
        area.find(".value-bools").val((nodeData.values || []).map(v=>v?"true":"false").join(" "));
    }
    else if (nodeData.type === "B") {
        area.find(".value-bytes").val((nodeData.values || []).join(" "));
    }
    else if (nodeData.type === "L") {
        let childContainer = node.find("> .children");
        (nodeData.values || []).forEach(c => loadNodeToUI(c, childContainer));
    }
}

$("#loadBtn").click(function () {
    try {
        let json = JSON.parse($("#jsonInput").val().trim());

        $("#streamInput").val(json.stream);
        $("#functionInput").val(json.function);

        $("#rootContainer").html("");

        loadNodeToUI(json.dataitem, $("#rootContainer"));
        //alert("JSON Loaded Successfully!");
    }
    catch (e) {
        alert("Invalid JSON!");
        console.error(e);
    }
});


function initLoadJsonExample(){
    dataitem = { "type": "L", "values": [] }
    let vidList = document.getElementById("s1f3_param").value.split(",")
    for(let i = 0 ; i < vidList.length ; i++){
        if( vidList[i] == "" ){
            break;
        }
        dataitem.values.push( { "type" : "U4" ,  "values" : [ parseInt(vidList[i],10) ]  })
    }
    cmd = { "stream" : 1 , "function" : 3 , "dataitem" : dataitem }
    $("#jsonInput").val(JSON.stringify(cmd,null,4))
    $("#loadBtn").click();
}

initLoadJsonExample()
