package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"xsserve/core"
	"xsserve/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Exported functions
func ServeUI(addr string, port int) (err error) {
	mux := http.NewServeMux()

	//redirectHandler := http.RedirectHandler("http://example.org", 307)
	//mux.Handle("/foo", rh)
	//notFoundHandler := http.NotFoundHandler()
	favicon := http.RedirectHandler("/static/resources/ui/images/favicon.ico", 307)
	index := http.HandlerFunc(indexHandle)
	report := http.HandlerFunc(reportHandle)
	triggers := http.HandlerFunc(triggersHandle)
	payloads := http.HandlerFunc(payloadsHandler)
	getScreenshot := http.HandlerFunc(getScreenshotHandler)

	mux.Handle("/favicon.ico", favicon)
	mux.Handle("/dashboard", index)
	mux.Handle("/triggers", triggers)
	mux.Handle("/report", report)
	mux.Handle("/payloads", payloads)
	mux.Handle("/get/screenshot", getScreenshot)

	//TODO: fix, The script from “http://host/static/resources/ui/js/main.js” was loaded even though its MIME type (“text/plain”) is not a valid JavaScript MIME type.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS))))

	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", addr, port),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	err = server.ListenAndServe()

	return err
}

func loadTemplate(file string) (*template.Template, error) {
	tmpl, err := template.ParseFS(FS, file, "resources/ui/layout.tmpl")

	return tmpl, err
}

// Handles
func indexHandle(w http.ResponseWriter, r *http.Request) {
	tmpl, err := loadTemplate("resources/ui/index.tmpl")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func triggersHandle(w http.ResponseWriter, r *http.Request) {
	tmpl, err := loadTemplate("resources/ui/triggers.tmpl")
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", 500)
		return
	}

	var triggers []core.Trigger
	cursor, err := database.DB.Collection("triggers").Find(database.CTX, bson.M{})
	cursor.All(database.CTX, &triggers)

	err = tmpl.Execute(w, triggers)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func reportHandle(w http.ResponseWriter, r *http.Request) {
	tmpl, err := loadTemplate("resources/ui/report.tmpl")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	objID, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}

	var trigger core.Trigger
	err = database.DB.Collection("triggers").FindOne(database.CTX, bson.M{"_id": objID.Hex()}).Decode(&trigger)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}

	err = tmpl.Execute(w, trigger)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func payloadsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: add more payloads and customize payloads with current HOST/IP address
	tmpl, err := loadTemplate("resources/ui/payloads.tmpl")
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	var payloads []core.Payload
	cursor, err := database.DB.Collection("payloads").Find(database.CTX, bson.M{})
	cursor.All(database.CTX, &payloads)

	err = tmpl.Execute(w, payloads)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func getScreenshotHandler(w http.ResponseWriter, r *http.Request) {
	objID, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}

	var trigger core.Trigger
	err = database.DB.Collection("triggers").FindOne(database.CTX, bson.M{"_id": objID.Hex()}).Decode(&trigger)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}

	w.Header().Add("Content-type", "image/png")
	w.Write(trigger.Screenshot)
}