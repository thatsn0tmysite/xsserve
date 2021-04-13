package core

import (
	"fmt"
	"net/http"
	"time"
)

type Payload struct {
	ID          string `bson:"_id"`
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
	UID         int
	ID          string `bson:"_id"`
	Payload     Payload
	Date        time.Time
	Screenshot  []byte
	DOM         string
	Host        string
	URI         string
	Origin      string
	BrowserDate time.Time
	UserAgent   string
	Referrer    string
	Cookies     []*http.Cookie
	//Live      bool //if online now allow hook injection to perform man in the browser?
}

func (trigger Trigger) String() string {
	return fmt.Sprintf("%v : %v :: %v", trigger.ID, trigger.Date.String(), trigger.Host)
}
