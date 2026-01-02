const MIN = 0;
const MAX = 100;
const params = {
    temperature: { current:50, l1Lower:10, l1Upper:15, l2Lower:85, l2Upper:90 },
    rpm:  { current:50, l1Lower:10, l1Upper:15, l2Lower:85, l2Upper:90 },
    psi:  { current:50, l1Lower:10, l1Upper:15, l2Lower:85, l2Upper:90 }
};
let ws = null;

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


function sendControlEvent(type, payload) {
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

function setLimitUI(p){
    let tokens = p.split(":")
    let lowerEl = null
    let upperEl = null
    let name = ""
    if( tokens[0] == 6) { //temperature
        name = "temperature"
        if(tokens[1] == 0){ //
            lowerEl = document.querySelector(
                '.slider[data-key="temperature"] [data-point="l1Lower"]'
            );
            upperEl = document.querySelector(
                '.slider[data-key="temperature"] [data-point="l1Upper"]'
            );
            params[name]['l1Upper'] = tokens[2]
            params[name]['l1Lower'] = tokens[3]
        }
        if(tokens[1] == 1){ //
            lowerEl = document.querySelector(
                '.slider[data-key="temperature"] [data-point="l2Lower"]'
            );
            upperEl = document.querySelector(
                '.slider[data-key="temperature"] [data-point="l2Upper"]'
            );
            params[name]['l2Upper'] = tokens[2]
            params[name]['l2Lower'] = tokens[3]

        }
    }

    if( tokens[0] == 7) { //rpm
        name = "rpm"
        if(tokens[1] == 2){ //
            lowerEl = document.querySelector(
                '.slider[data-key="rpm"] [data-point="l1Lower"]'
            );
            upperEl = document.querySelector(
                '.slider[data-key="rpm"] [data-point="l1Upper"]'
            );
            params[name]['l1Upper'] = tokens[2]
            params[name]['l1Lower'] = tokens[3]
        }
        if(tokens[1] == 3){ //
            lowerEl = document.querySelector(
                '.slider[data-key="rpm"] [data-point="l2Lower"]'
            );
            upperEl = document.querySelector(
                '.slider[data-key="rpm"] [data-point="l2Upper"]'
            );
            params[name]['l2Upper'] = tokens[2]
            params[name]['l2Lower'] = tokens[3]
        }
    }
    if( tokens[0] == 8) { //temperature
        name = "psi"
        if(tokens[1] == 4){ //
            lowerEl = document.querySelector(
                '.slider[data-key="psi"] [data-point="l1Lower"]'
            );
            upperEl = document.querySelector(
                '.slider[data-key="psi"] [data-point="l1Upper"]'
            );
            params[name]['l1Upper'] = tokens[2]
            params[name]['l1Lower'] = tokens[3]
        }
        if(tokens[1] == 5){ //
            lowerEl = document.querySelector(
                '.slider[data-key="psi"] [data-point="l2Lower"]'
            );
            upperEl = document.querySelector(
                '.slider[data-key="psi"] [data-point="l2Upper"]'
            );
            params[name]['l2Upper'] = tokens[2]
            params[name]['l2Lower'] = tokens[3]

        }
    }

    const upperpercent = tokens[2]*100/(MAX-MIN)
    const lowerpercent = tokens[3]*100/(MAX-MIN)


    upperEl.style.left =upperpercent +'%'
    lowerEl.style.left = lowerpercent + '%'

}

function initWebSocket() {
    ws = new WebSocket("ws://" + location.hostname  + ":" + location.port + "/api/equipment");
    ws.onopen = function () {
        console.log("WebSocket connected");
        updateStatus("Connected");
    };

    ws.onclose = function () {
        console.log("WebSocket closed");
        updateStatus("Closed");
    };

    ws.onerror = function (err) {
        console.error("WebSocket error:", err);
        updateStatus("Error");
    };

    ws.onmessage = function (event) {
        console.log("Received from server:", event.data);
        obj = JSON.parse(event.data)
        console.log(obj)
        if(obj["evttype"] == "CtrlChange"){
            const [substate,state] = obj["data"].split("@");
            document.getElementById("ctrlstate").textContent = obj["data"]
            if(state == "ONLINE"){
                document.getElementById("btnOnlineRequest").disabled = true
                document.getElementById("btnOffline").disabled = false
                if(substate == "LOCAL"){
                    document.getElementById("btnOnlineRemote").disabled = false
                    document.getElementById("btnOnlineLocal").disabled = true
                }
                if(substate == "REMOTE"){
                    document.getElementById("btnOnlineRemote").disabled = true
                    document.getElementById("btnOnlineLocal").disabled = false
                }
            }
            if(state == "OFFLINE"){
                 if(substate == "ATTEMPTONLINE"){ //wait host reply or timeout
                     document.getElementById("btnOnlineRequest").disabled = true
                     document.getElementById("btnOffline").disabled = true
                     document.getElementById("btnOnlineRemote").disabled = true
                     document.getElementById("btnOnlineLocal").disabled = true
                 }
                 if(substate == "EQUIPMENT"){
                     document.getElementById("btnOnlineRequest").disabled = false
                     document.getElementById("btnOffline").disabled = true
                     document.getElementById("btnOnlineRemote").disabled = true
                     document.getElementById("btnOnlineLocal").disabled = true
                 }
                 if(substate == "HOST"){
                     document.getElementById("btnOnlineRequest").disabled = true
                     document.getElementById("btnOffline").disabled = false
                     document.getElementById("btnOnlineRemote").disabled = true
                     document.getElementById("btnOnlineLocal").disabled = true
                 }
            }
        }
        if(obj["evttype"] == "S10F3"){
            document.getElementById("chatbox").innerHTML = obj["data"] + ' <button type="button" id="btnRecognize">Recognize</button>'
            document.getElementById("btnRecognize").addEventListener("click", function () {
                sendControlEvent("recognize", { });
                document.getElementById("btnRecognize").remove()
            });

        }
        if(obj["evttype"] == "S2F41"){
            //document.getElementById("chatbox").innerText = obj["data"]
            alert("Receive : Host Call " + obj["data"] );
        }
        if(obj["evttype"] == "S2F45"){
            //document.getElementById("chatbox").innerText = obj["data"]
            setLimitUI(obj["data"])
        }

    };
}

// 綁定 UI 事件
function bindEvents() {
    // 第一組：radio
    document.getElementById("commEnable").addEventListener("change", function (e) {
        if (e.target.checked) {
            sendControlEvent("communication", { value: "enable" });
        }
    });

    document.getElementById("commDisable").addEventListener("change", function (e) {
        if (e.target.checked) {
            sendControlEvent("communication", { value: "disable" });
        }
     });

    // 第二組：四個按鈕
    document.getElementById("btnOnlineRequest").addEventListener("click", function () {
        sendControlEvent("mode", { value: "online_request" });
    });

    document.getElementById("btnOffline").addEventListener("click", function () {
        sendControlEvent("mode", { value: "offline" });
    });

    document.getElementById("btnOnlineRemote").addEventListener("click", function () {
        sendControlEvent("mode", { value: "online_remote" });
    });

    document.getElementById("btnOnlineLocal").addEventListener("click", function () {
        sendControlEvent("mode", { value: "online_local" });
    });

    document.getElementById("btnSendText").addEventListener("click", function () {
        textcontent = document.getElementById("sendtextcontent").value
        sendControlEvent("sendtext", { value : textcontent });
    });

    document.getElementById("btnZombieAttack").addEventListener("click", function () {
        alid = 1
        alcd =128
        sendControlEvent("setalarm", { alid : alid , alcd : alcd });
    });

    document.getElementById("btnRobotRebellion").addEventListener("click", function () {
        alid = 2
        alcd =128
        sendControlEvent("setalarm", { alid : alid , alcd : alcd });
    });

    document.getElementById("btnSetEC").addEventListener("click", function () {
        dataitem = { "type": "L", "items": [] }
        let pairList = document.getElementById("setec_param").value.split(",")
        for(let i = 0 ; i < pairList.length ; i++){
            if( pairList[i] == "" ){
                alert("empty not allowed")
                return;
            }
            let pair = pairList[i].split("=")
            dataitem.items.push( { "type" : "L", "items" :[ { "type" : "U4" , "values" : [   parseInt(pair[0],10) ] } , { "type" : "U4" , "values" : [   parseInt(pair[1],10) ] } ] } )
        }
        sendControlEvent("setec", {  dataitem : JSON.stringify(dataitem) } );
    });

}

function clamp(v){ return Math.max(MIN, Math.min(MAX, v)); }

function syncUI(name) {
    const p = params[name];
    const slider = document.querySelector(`.slider[data-key='${name}']`);

    slider.querySelectorAll(".slider-handle").forEach(h => {
        const key = h.dataset.point;
        const v = p[key];
        h.style.left = ((v - MIN)/(MAX-MIN)*100) + "%";
    });

    for (const key in p) {
        const inp = document.querySelector(`input[data-input='${name}-${key}']`);
        if (inp) inp.value = p[key].toFixed(1);
    }
}

function applyConstraint(p) {
    p.l1Lower = Math.min(p.l1Lower, p.l1Upper);
    p.l1Upper = Math.max(p.l1Upper, p.l1Lower);

    p.l2Lower = Math.min(p.l2Lower, p.l2Upper);
    p.l2Upper = Math.max(p.l2Upper, p.l2Lower);

    if(p.l1Lower > p.l1Upper){
        p.l1Upper = p.l1Lower + 1
    }
    if(p.l2Lower > p.l2Upper){
        p.l2Lower =  p.l2Upper - 1
    }


}

document.querySelectorAll(".slider").forEach(slider => {
    const name = slider.dataset.key;

    let active = null;

    slider.addEventListener("mousedown", e => {
        if (e.target.classList.contains("slider-handle")) {
            active = e.target;
            document.addEventListener("mousemove", move);
            document.addEventListener("mouseup", up);
        }
    });

    function move(e) {
        if (!active) return;
        const rect = slider.getBoundingClientRect();
        const percent = (e.clientX - rect.left) / rect.width * 100;
        const value = clamp(MIN + (percent/100)*(MAX-MIN));

        const key = active.dataset.point;
        params[name][key] = value;
        applyConstraint(params[name]);
        syncUI(name);
    }

    function up(e) {

        const key = active.dataset.point;
        if(key == "l1Lower" || key == "l1Upper"){
            if(name == "temperature"){
                sendControlEvent(name + "_limit" , { upperdb : Number(params[name]["l1Upper"]) , lowerdb : Number(params[name]["l1Lower"]) , limitid : 0 });
            }
            if(name == "rpm"){
                sendControlEvent(name + "_limit" , { upperdb : Number(params[name]["l1Upper"]) , lowerdb : Number(params[name]["l1Lower"]) , limitid : 2 });
            }
            if(name == "psi"){
                sendControlEvent(name + "_limit" , { upperdb : Number(params[name]["l1Upper"]) , lowerdb : Number(params[name]["l1Lower"]) , limitid : 4 });
            }


        }
        if(key == "current"){
            sendControlEvent(name, { value: Number(params[name][key]) });
        }
        if(key == "l2Lower" || key == "l2Upper"){
            if(name == "temperature"){
                sendControlEvent(name + "_limit" , { upperdb : Number(params[name]["l2Upper"]) , lowerdb : Number(params[name]["l2Lower"]) , limitid : 1  });
            }
            if(name == "rpm"){
                sendControlEvent(name + "_limit" , { upperdb : Number(params[name]["l2Upper"]) , lowerdb : Number(params[name]["l2Lower"]) , limitid : 3 });
            }
            if(name == "psi"){
                sendControlEvent(name + "_limit" , { upperdb : Number(params[name]["l2Upper"]) , lowerdb : Number(params[name]["l2Lower"]) , limitid : 5 });
            }

        }
        active = null;
        document.removeEventListener("mousemove", move);
        document.removeEventListener("mouseup", up);
    }
});

document.querySelectorAll("input[data-input]").forEach(inp => {
    inp.addEventListener("change", () => {
        const [name, key] = inp.dataset.input.split("-");
        params[name][key] = clamp(parseFloat(inp.value));
        applyConstraint(params[name]);
        syncUI(name);
    });
});

window.addEventListener("load", function () {
        updateStatus("Connecting…");
        initWebSocket();
        bindEvents();
        // INIT
        syncUI("temperature");
        syncUI("rpm");
        syncUI("psi");
    setTimeout( function(){
        sendControlEvent("temperature_limit" , { upperdb : Number(params["temperature"]["l1Upper"]) , lowerdb : Number(params["temperature"]["l1Lower"]) , limitid : 0 });
        sendControlEvent("temperature_limit" , { upperdb : Number(params["temperature"]["l2Upper"]) , lowerdb : Number(params["temperature"]["l2Lower"]) , limitid : 1  });
        sendControlEvent("rpm_limit" , { upperdb : Number(params["rpm"]["l1Upper"]) , lowerdb : Number(params["rpm"]["l1Lower"]) , limitid : 2 });
        sendControlEvent("rpm_limit" , { upperdb : Number(params["rpm"]["l2Upper"]) , lowerdb : Number(params["rpm"]["l2Lower"]) , limitid : 3 });
        sendControlEvent("psi_limit" , { upperdb : Number(params["psi"]["l1Upper"]) , lowerdb : Number(params["psi"]["l1Lower"]) , limitid : 4 });
        sendControlEvent("psi_limit" , { upperdb : Number(params["psi"]["l2Upper"]) , lowerdb : Number(params["psi"]["l2Lower"]) , limitid : 5 });
        },2000);
});

