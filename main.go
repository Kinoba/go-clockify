/*

Package clockify provides an API for interacting with the Clockify time tracking service.

See https://clockify.me/developers-api for more information on Clockify's REST API.

*/
package clockify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Clockify service constants
const (
	ClockifyAPI       = "https://api.clockify.me/api/v1"
	DefaultAppName = "go-clockify"
)

var (
	dlog   = log.New(os.Stderr, "[clockify] ", log.LstdFlags)
	client = &http.Client{}

	// AppName is the application name used when creating timers.
	AppName = DefaultAppName
)

// structures ///////////////////////////

// Session represents an active connection to the Clockify REST API.
type Session struct {
	APIToken string
}

// AccountSettings represents a user account settings.
type AccountSettings struct {
	WeekStart 		string		`json:"weekStart"`
    TimeZone 		string 		`json:"timeZone"`
    TimeFormat 		string 		`json:"timeFormat"`
    DateFormat 		string 		`json:"dateFormat"`
}

// Account represents a user account.
type Account struct {
	// APIToken        string      `json:"api_token"`
	ID              string          `json:"id"`
	Name			string			`json:"name"`
	Email			string			`json:"email"`
	Workspaces      []Workspace 	`json:"workspaces"`
	Clients         []Client    	`json:"clients"`
	Projects        []Project   	`json:"projects"`
	Tasks           []Task      	`json:"tasks"`
	Tags            []Tag       	`json:"tags"`
	TimeEntries     []TimeEntry 	`json:"time_entries"`
	Settings     	AccountSettings `json:"settings"`
}

// Workspace represents a user workspace.
type Workspace struct {
	ID              string    `json:"id"`
	RoundingMinutes int    `json:"rounding_minutes"`
	// Rounding        int    `json:"rounding"`
	Name            string `json:"name"`
	// Premium         bool   `json:"premium"`
}

// Client represents a client.
type Client struct {
	Wid   string    `json:"workspaceId"`
	ID    int    `json:"id"`
	Name  string `json:"name"`
	// Notes string `json:"notes"`
}

// Project represents a project.
type Project struct {
	Wid             string     `json:"workspaceId"`
	ID              string     `json:"id"`
	// Cid             int        `json:"cid"`
	Name            string     `json:"name"`
	Active          bool       `json:"archived"`
	Billable        bool       `json:"billable"`
}

// IsActive indicates whether a project exists and is active
func (p *Project) IsActive() bool {
	return p.Active
}

// Task represents a task.
type Task struct {
	Pid  string `json:"projectId"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Tag represents a tag.
type Tag struct {
	Wid  string `json:"workspaceId"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TimeInterval represents a time interval.
type TimeInterval struct {
	Duration  string `json:"duration"`
	Stop      *time.Time `json:"end,omitempty"`
	Start     *time.Time `json:"start,omitempty"`
}

// TimeEntry represents a single time entry.
type TimeEntry struct {
	Wid          string       `json:"workspaceId,omitempty"`
	ID           string       `json:"id,omitempty"`
	Pid          string       `json:"pid"`
	Tid          string       `json:"taskId"`
	Description  string       `json:"description,omitempty"`
	TimeInterval TimeInterval `json:"timeInterval"`
	Tags         []string     `json:"tagIds"`
	Billable     bool         `json:"billable"`
}

// TimeEntryRequest represents a single time entry request.
type TimeEntryRequest struct {
	Start 		 string		  `json:"start,omitempty"`
	Pid          string       `json:"projectId,omitempty"`
	Tid          string       `json:"taskId,omitempty"`
	Description  string       `json:"description,omitempty"`
	End 		 string		  `json:"end,omitempty"`
	Tags         []string     `json:"tagIds,omitempty"`
	Billable     bool         `json:"billable,omitempty"`
}

// type DetailedTimeEntry struct {
// 	ID              int        `json:"id"`
// 	Pid             int        `json:"pid"`
// 	Tid             int        `json:"tid"`
// 	Uid             int        `json:"uid"`
// 	User            string     `json:"user,omitempty"`
// 	Description     string     `json:"description"`
// 	Project         string     `json:"project"`
// 	ProjectColor    string     `json:"project_color"`
// 	ProjectHexColor string     `json:"project_hex_color"`
// 	Client          string     `json:"client"`
// 	Start           *time.Time `json:"start"`
// 	End             *time.Time `json:"end"`
// 	Updated         *time.Time `json:"updated"`
// 	Duration        int64      `json:"dur"`
// 	Billable        bool       `json:"billable"`
// 	Tags            []string   `json:"tags"`
// }

// functions ////////////////////////////

// OpenSession opens a session using an existing API token.
func OpenSession(apiToken string) Session {
	return Session{APIToken: apiToken}
}

// GetAccount returns a user's account information, including a list of active
// projects and timers.
func (session *Session) GetAccount() (Account, error) {
	data, err := session.get(ClockifyAPI, "/user", nil)
	if err != nil {
		return Account{}, err
	}

	var account Account
	err = decodeAccount(data, &account)
	return account, err
}

// StartTimeEntry creates a new time entry.
func (session *Session) StartTimeEntry(workspaceID string, timeEntryRequest TimeEntryRequest) (TimeEntry, error) {
	path := fmt.Sprintf("/workspaces/%s/time-entries", workspaceID)
	respData, err := session.post(ClockifyAPI, path, timeEntryRequest)
	return requestTimeEntry(respData, err)
}

// GetTimeEntry returns the time entry
func (session *Session) GetTimeEntry(workspaceID, timeEntryID string) (TimeEntry, error) {
	path := fmt.Sprintf("/workspaces/%s/time-entries/%s", workspaceID, timeEntryID)
	data, err := session.get(ClockifyAPI, path, nil)
	if err != nil {
		return TimeEntry{}, err
	}

	return requestTimeEntry(data, err)
}

// DeleteTimeEntry deletes a time entry.
func (session *Session) DeleteTimeEntry(workspaceID, timeEntryID string) ([]byte, error) {
	dlog.Printf("Deleting time entry %v", timeEntryID)
	path := fmt.Sprintf("/workspaces/%s/time-entries/%s", workspaceID, timeEntryID)
	return session.delete(ClockifyAPI, path)
}

// GetTimeEntries returns a list of time entries
// func (session *Session) GetTimeEntries(startDate, endDate time.Time) ([]TimeEntry, error) {
// 	params := make(map[string]string)
// 	params["start_date"] = startDate.Format(time.RFC3339)
// 	params["end_date"] = endDate.Format(time.RFC3339)
// 	data, err := session.get(ClockifyAPI, "/time-entries", params)
// 	if err != nil {
// 		return nil, err
// 	}
// 	results := make([]TimeEntry, 0)
// 	err = json.Unmarshal(data, &results)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return results, nil
// }

// ContinueTimeEntry continues a time entry by creating a new entry
// with the same description. The new entry will have the same description and project ID as
// the existing one.
func (session *Session) ContinueTimeEntry(timer TimeEntry, duronly bool) (TimeEntry, error) {
	dlog.Printf("Continuing timer %v", timer)
	var respData []byte
	var err error
	var timeEntryRequest TimeEntryRequest
	
	timeEntryRequest.Start = time.Now().UTC().Format(time.RFC3339)
	timeEntryRequest.Pid = timer.Pid
	timeEntryRequest.Tid = timer.Tid
	timeEntryRequest.Description  = timer.Description
	timeEntryRequest.Tags = timer.Tags
	timeEntryRequest.Billable = timer.Billable
	
	path := fmt.Sprintf("/workspaces/%s/time-entries", timer.Wid)
	
	respData, err = session.post(ClockifyAPI, path, timeEntryRequest)
	
	return requestTimeEntry(respData, err)
}


// StopTimeEntry stops a running time entry.
func (session *Session) StopTimeEntry(workspaceID, userID string) (TimeEntry, error) {
	dlog.Printf("Stopping timer to user %s", userID)
	path := fmt.Sprintf("/workspaces/{workspaceId}/user/{userId}/time-entries", workspaceID, userID)
	respData, err := session.patch(ClockifyAPI, path, TimeEntryRequest{End: time.Now().UTC().Format(time.RFC3339)})
	return requestTimeEntry(respData, err)
}

// AddRemoveTag adds or removes a tag from the time entry corresponding to a
// given ID.
// func (session *Session) AddRemoveTag(entryID int, tag string, add bool) (TimeEntry, error) {
// 	dlog.Printf("Adding tag to time entry %v", entryID)
// 
// 	action := "add"
// 	if !add {
// 		action = "remove"
// 	}
// 
// 	data := map[string]interface{}{
// 		"time_entry": map[string]interface{}{
// 			"tags":       []string{tag},
// 			"tag_action": action,
// 		},
// 	}
// 	path := fmt.Sprintf("/time_entries/%v", entryID)
// 	respData, err := session.post(ClockifyAPI, path, data)
// 
// 	return requestTimeEntry(respData, err)
// }


// // IsRunning returns true if the receiver is currently running.
// func (e *TimeEntry) IsRunning() bool {
// 	return e.Duration < 0
// }

// GetProjects allows to query for all projects in a workspace
func (session *Session) GetProjects(workspaceID string) (projects []Project, err error) {
	dlog.Printf("Getting projects for workspace %s", workspaceID)
	path := fmt.Sprintf("/workspaces/%s/projects", workspaceID)
	data,err := session.get(ClockifyAPI, path, nil)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &projects)
	dlog.Printf("Unmarshaled '%s' into %#v\n", data, projects)
	return
}

// // CreateProject creates a new project.
// func (session *Session) CreateProject(name string, wid int) (proj Project, err error) {
// 	dlog.Printf("Creating project %s", name)
// 	data := map[string]interface{}{
// 		"project": map[string]interface{}{
// 			"name": name,
// 			"wid":  wid,
// 		},
// 	}
// 
// 	respData, err := session.post(ClockifyAPI, "/projects", data)
// 	if err != nil {
// 		return proj, err
// 	}
// 
// 	var entry struct {
// 		Data Project `json:"data"`
// 	}
// 	err = json.Unmarshal(respData, &entry)
// 	dlog.Printf("Unmarshaled '%s' into %#v\n", respData, entry)
// 	if err != nil {
// 		return proj, err
// 	}
// 
// 	return entry.Data, nil
// }
// 
// // UpdateProject changes information about an existing project.
// func (session *Session) UpdateProject(project Project) (Project, error) {
// 	dlog.Printf("Updating project %v", project)
// 	data := map[string]interface{}{
// 		"project": project,
// 	}
// 	path := fmt.Sprintf("/projects/%v", project.ID)
// 	respData, err := session.put(ClockifyAPI, path, data)
// 
// 	if err != nil {
// 		return Project{}, err
// 	}
// 
// 	var entry struct {
// 		Data Project `json:"data"`
// 	}
// 	err = json.Unmarshal(respData, &entry)
// 	dlog.Printf("Unmarshaled '%s' into %#v\n", data, entry)
// 	if err != nil {
// 		return Project{}, err
// 	}
// 
// 	return entry.Data, nil
// }
// 
// // DeleteProject deletes a project.
// func (session *Session) DeleteProject(project Project) ([]byte, error) {
// 	dlog.Printf("Deleting project %v", project)
// 	path := fmt.Sprintf("/projects/%v", project.ID)
// 	return session.delete(ClockifyAPI, path)
// }
// 
// // CreateTag creates a new tag.
// func (session *Session) CreateTag(name string, wid int) (proj Tag, err error) {
// 	dlog.Printf("Creating tag %s", name)
// 	data := map[string]interface{}{
// 		"tag": map[string]interface{}{
// 			"name": name,
// 			"wid":  wid,
// 		},
// 	}
// 
// 	respData, err := session.post(ClockifyAPI, "/tags", data)
// 	if err != nil {
// 		return proj, err
// 	}
// 
// 	var entry struct {
// 		Data Tag `json:"data"`
// 	}
// 	err = json.Unmarshal(respData, &entry)
// 	dlog.Printf("Unmarshaled '%s' into %#v\n", respData, entry)
// 	if err != nil {
// 		return proj, err
// 	}
// 
// 	return entry.Data, nil
// }
// 
// // UpdateTag changes information about an existing tag.
// func (session *Session) UpdateTag(tag Tag) (Tag, error) {
// 	dlog.Printf("Updating tag %v", tag)
// 	data := map[string]interface{}{
// 		"tag": tag,
// 	}
// 	path := fmt.Sprintf("/tags/%v", tag.ID)
// 	respData, err := session.put(ClockifyAPI, path, data)
// 
// 	if err != nil {
// 		return Tag{}, err
// 	}
// 
// 	var entry struct {
// 		Data Tag `json:"data"`
// 	}
// 	err = json.Unmarshal(respData, &entry)
// 	dlog.Printf("Unmarshaled '%s' into %#v\n", data, entry)
// 	if err != nil {
// 		return Tag{}, err
// 	}
// 
// 	return entry.Data, nil
// }
// 
// // DeleteTag deletes a tag.
// func (session *Session) DeleteTag(tag Tag) ([]byte, error) {
// 	dlog.Printf("Deleting tag %v", tag)
// 	path := fmt.Sprintf("/tags/%v", tag.ID)
// 	return session.delete(ClockifyAPI, path)
// }
// 
// // GetClients returns a list of clients for the current account
// func (session *Session) GetClients() (clients []Client, err error) {
// 	dlog.Println("Retrieving clients")
// 
// 	data, err := session.get(ClockifyAPI, "/clients", nil)
// 	if err != nil {
// 		return clients, err
// 	}
// 	err = json.Unmarshal(data, &clients)
// 	return clients, err
// }
// 
// // CreateClient adds a new client
// func (session *Session) CreateClient(name string, wid int) (client Client, err error) {
// 	dlog.Printf("Creating client %s", name)
// 	data := map[string]interface{}{
// 		"client": map[string]interface{}{
// 			"name": name,
// 			"wid":  wid,
// 		},
// 	}
// 
// 	respData, err := session.post(ClockifyAPI, "/clients", data)
// 	if err != nil {
// 		return client, err
// 	}
// 
// 	var entry struct {
// 		Data Client `json:"data"`
// 	}
// 	err = json.Unmarshal(respData, &entry)
// 	dlog.Printf("Unmarshaled '%s' into %#v\n", respData, entry)
// 	if err != nil {
// 		return client, err
// 	}
// 	return entry.Data, nil
// }
// 
// // Copy returns a copy of a TimeEntry.
// func (e *TimeEntry) Copy() TimeEntry {
// 	newEntry := *e
// 	newEntry.Tags = make([]string, len(e.Tags))
// 	copy(newEntry.Tags, e.Tags)
// 	if e.Start != nil {
// 		newEntry.Start = &(*e.Start)
// 	}
// 	if e.Stop != nil {
// 		newEntry.Stop = &(*e.Stop)
// 	}
// 	return newEntry
// }
// 
// // StartTime returns the start time of a time entry as a time.Time.
// func (e *TimeEntry) StartTime() time.Time {
// 	if e.Start != nil {
// 		return *e.Start
// 	}
// 	return time.Time{}
// }
// 
// // StopTime returns the stop time of a time entry as a time.Time.
// func (e *TimeEntry) StopTime() time.Time {
// 	if e.Stop != nil {
// 		return *e.Stop
// 	}
// 	return time.Time{}
// }
// 
// // HasTag returns true if a time entry contains a given tag.
// func (e *TimeEntry) HasTag(tag string) bool {
// 	return indexOfTag(tag, e.Tags) != -1
// }
// 
// // AddTag adds a tag to a time entry if the entry doesn't already contain the
// // tag.
// func (e *TimeEntry) AddTag(tag string) {
// 	if !e.HasTag(tag) {
// 		e.Tags = append(e.Tags, tag)
// 	}
// }
// 
// // RemoveTag removes a tag from a time entry.
// func (e *TimeEntry) RemoveTag(tag string) {
// 	if i := indexOfTag(tag, e.Tags); i != -1 {
// 		e.Tags = append(e.Tags[:i], e.Tags[i+1:]...)
// 	}
// }
// 
// // SetDuration sets a time entry's duration. The duration should be a value in
// // seconds. The stop time will also be updated. Note that the time entry must
// // not be running.
// func (e *TimeEntry) SetDuration(duration int64) error {
// 	if e.IsRunning() {
// 		return fmt.Errorf("TimeEntry must be stopped")
// 	}
// 
// 	e.Duration = duration
// 	newStop := e.Start.Add(time.Duration(duration) * time.Second)
// 	e.Stop = &newStop
// 
// 	return nil
// }
// 
// // SetStartTime sets a time entry's start time. If the time entry is stopped,
// // the stop time will also be updated.
// func (e *TimeEntry) SetStartTime(start time.Time, updateEnd bool) {
// 	e.Start = &start
// 
// 	if !e.IsRunning() {
// 		if updateEnd {
// 			newStop := start.Add(time.Duration(e.Duration) * time.Second)
// 			e.Stop = &newStop
// 		} else {
// 			e.Duration = e.Stop.Unix() - e.Start.Unix()
// 		}
// 	}
// }
// 
// // SetStopTime sets a time entry's stop time. The duration will also be
// // updated. Note that the time entry must not be running.
// func (e *TimeEntry) SetStopTime(stop time.Time) (err error) {
// 	if e.IsRunning() {
// 		return fmt.Errorf("TimeEntry must be stopped")
// 	}
// 
// 	e.Stop = &stop
// 	e.Duration = int64(stop.Sub(*e.Start) / time.Second)
// 
// 	return nil
// }
// 
// func indexOfTag(tag string, tags []string) int {
// 	for i, t := range tags {
// 		if t == tag {
// 			return i
// 		}
// 	}
// 	return -1
// }
// 
// // UnmarshalJSON unmarshals a TimeEntry from JSON data, converting timestamp
// // fields to Go Time values.
// func (e *TimeEntry) UnmarshalJSON(b []byte) error {
// 	var entry tempTimeEntry
// 	err := json.Unmarshal(b, &entry)
// 	if err != nil {
// 		return err
// 	}
// 	te, err := entry.asTimeEntry()
// 	if err != nil {
// 		return err
// 	}
// 	*e = te
// 	return nil
// }

// support /////////////////////////////////////////////////////////////

func (session *Session) request(method string, requestURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, requestURL, body)

	if session.APIToken != "" {
		req.Header.Add("X-Api-Key", session.APIToken)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return content, fmt.Errorf(resp.Status)
	}

	return content, nil
}

func (session *Session) get(requestURL string, path string, params map[string]string) ([]byte, error) {
	requestURL += path

	if params != nil {
		data := url.Values{}
		for key, value := range params {
			data.Set(key, value)
		}
		requestURL += "?" + data.Encode()
	}

	dlog.Printf("GETing from URL: %s", requestURL)
	return session.request("GET", requestURL, nil)
}

func (session *Session) post(requestURL string, path string, data interface{}) ([]byte, error) {
	requestURL += path
	var body []byte
	var err error

	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}

	dlog.Printf("POSTing to URL: %s", requestURL)
	dlog.Printf("data: %s", body)
	return session.request("POST", requestURL, bytes.NewBuffer(body))
}

func (session *Session) put(requestURL string, path string, data interface{}) ([]byte, error) {
	requestURL += path
	var body []byte
	var err error

	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}

	dlog.Printf("PUTing to URL %s: %s", requestURL, string(body))
	return session.request("PUT", requestURL, bytes.NewBuffer(body))
}

func (session *Session) patch(requestURL string, path string, data interface{}) ([]byte, error) {
	requestURL += path
	var body []byte
	var err error

	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}

	dlog.Printf("PATCHing to URL %s: %s", requestURL, string(body))
	return session.request("PATCH", requestURL, bytes.NewBuffer(body))
}

func (session *Session) delete(requestURL string, path string) ([]byte, error) {
	requestURL += path
	dlog.Printf("DELETINGing URL: %s", requestURL)
	return session.request("DELETE", requestURL, nil)
}

func decodeSession(data []byte, session *Session) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(session)
	if err != nil {
		return err
	}
	return nil
}

func decodeAccount(data []byte, account *Account) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(account)
	if err != nil {
		return err
	}
	return nil
}

// func decodeSummaryReport(data []byte, report *SummaryReport) error {
// 	dlog.Printf("Decoding %s", data)
// 	dec := json.NewDecoder(bytes.NewReader(data))
// 	err := dec.Decode(&report)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func decodeDetailedReport(data []byte, report *DetailedReport) error {
// 	dlog.Printf("Decoding %s", data)
// 	dec := json.NewDecoder(bytes.NewReader(data))
// 	err := dec.Decode(&report)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// This is an alias for TimeEntry that is used in tempTimeEntry to prevent the
// unmarshaler from infinitely recursing while unmarshaling.
// type embeddedTimeEntry TimeEntry
// 
// // tempTimeEntry is an intermediate type used as for decoding TimeEntries.
// type tempTimeEntry struct {
// 	embeddedTimeEntry
// 	Stop  string `json:"stop"`
// 	Start string `json:"start"`
// }
// 
// func (t *tempTimeEntry) asTimeEntry() (entry TimeEntry, err error) {
// 	entry = TimeEntry(t.embeddedTimeEntry)
// 
// 	parseTime := func(s string) (t time.Time, err error) {
// 		t, err = time.Parse("2006-01-02T15:04:05Z", s)
// 		if err != nil {
// 			t, err = time.Parse("2006-01-02T15:04:05-07:00", s)
// 		}
// 		return
// 	}
// 
// 	if t.Start != "" {
// 		var start time.Time
// 		start, err = parseTime(t.Start)
// 		if err != nil {
// 			return
// 		}
// 		entry.Start = &start
// 	}
// 
// 	if t.Stop != "" {
// 		var stop time.Time
// 		stop, err = parseTime(t.Stop)
// 		if err != nil {
// 			return
// 		}
// 		entry.Stop = &stop
// 	}
// 
// 	return
// }

func requestTimeEntry(data []byte, err error) (TimeEntry, error) {
	if err != nil {
		return TimeEntry{}, err
	}

	var entry TimeEntry
	
	err = json.Unmarshal(data, &entry)
	dlog.Printf("Unmarshaled '%s' into %#v\n", data, entry)
	if err != nil {
		return TimeEntry{}, err
	}

	return entry, nil
}

// DisableLog disables output to stderr
func DisableLog() {
	dlog.SetFlags(0)
	dlog.SetOutput(ioutil.Discard)
}

// EnableLog enables output to stderr
func EnableLog() {
	logFlags := dlog.Flags()
	dlog.SetFlags(logFlags)
	dlog.SetOutput(os.Stderr)
}
