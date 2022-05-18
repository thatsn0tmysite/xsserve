package core

import (
	"fmt"
	"net/http"
	"time"
)

// Command line flags
type Flags struct {
	DatabaseURI     string // Database URI to use
	Domain          string // Domain name to use
	IsHTTPS         bool   // Serve XSS over HTTPS?
	HTTPSCert       string // Certificate path
	HTTPSKey        string // Key path
	UIAddress       string // Address to host the UI on (defaults to 127.0.0.1)
	UIPort          int    // Port to bind for the UI to (defaults to 7331)
	XSSAddress      string // Address to serve the XSS files on (defaults to 0.0.0.0)
	XSSPort         int    // Port to bind for the XSS server to (defaults to 8443 if IsHTTPS, 8080 otherwise)
	ConfigFile      string // Viper configuration file to use
	Verbosity       int    // Verbosity level (defaults to 0, >4 is debug)
	BasicAuth       bool   // Enable Basic Authentication
	BasicAuthUser   string // Basic authentication username
	BasicAuthPass   string // Basic authentication password
	SeleniumURL     string // Selenium node to use
	SeleniumBrowser string // Selenium driver to use
	HookPluginsPath string // Path to hook.js plugins
}

type Payload struct {
	ID          int
	Code        string
	Description string
	//HasWhiteSpaces bool
	//HasTags bool
	//...
	//IsDOM bool
}

func (payload Payload) String() string {
	return fmt.Sprintf("%v : %v :: %v", payload.ID, payload.Code, payload.Description)
}

type Trigger struct {
	UID         string
	ID          int
	Payload     Payload
	Date        time.Time
	Screenshot  []byte
	DOM         string
	Host        string
	RemoteAddr  string
	URI         string
	Origin      string
	BrowserDate time.Time
	UserAgent   string
	Referrer    string
	Cookies     []*http.Cookie
	Status 		string
	Commands    []string //[]TriggerCommand
	Online      bool
}

/*
type TriggerCommand {
	ID	int
	TriggerId int
	QueuePosition int
	IssuedAt string
	RepliedAt string
	Code string
	Result string
}
*/


func (trigger Trigger) String() string {
	return fmt.Sprintf("%v : %v :: %v", trigger.ID, trigger.Date.String(), trigger.Host)
}
