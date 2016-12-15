package jirardeau

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Jira holds Url like https://jira.tld
type Jira struct {
	Log       *log.Logger
	Login     string
	Password  string
	Project   string
	ProjectID string
	Url       string
}

// JiraRelease holds JIRA Version
type JiraRelease struct {
	Archived        bool   `json:"archived"`
	Id              string `json:"id"`
	Name            string `json:"name"`
	Overdue         bool   `json:"overdue"`
	ProjectID       int    `json:"projectId"`
	ReleaseDate     string `json:"releaseDate"`
	Released        bool   `json:"released"`
	Self            string `json:"self"`
	StartDate       string `json:"startDate"`
	UserReleaseDate string `json:"userReleaseDate"`
	UserStartDate   string `json:"userStartDate"`
}

func (jira *Jira) request(method, url string, reqBody io.Reader) (respBody io.Reader, err error) {
	absUrl := jira.Url + url
	jira.Log.Println("STRT", method, absUrl)

	req, err := http.NewRequest(method, absUrl, reqBody)
	if err != nil {
		err = fmt.Errorf("Failed to build HTTP request %s %s: %s", method, absUrl, err)
		jira.Log.Println(err)
		return
	}
	req.Header.Set("content-type", "application/json")
	req.SetBasicAuth(jira.Login, jira.Password)

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		err = fmt.Errorf("Failed to JIRA request %s %s: %s", method, absUrl, err)
		jira.Log.Println(err)
		return
	}

	jira.Log.Println("StatusCode:", resp.StatusCode)
	jira.Log.Println("Headers:", resp.Header)
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		err = fmt.Errorf("Failed to read response from JIRA request %s %s: %s", method, absUrl, err)
		jira.Log.Println(err)
		return
	}
	respBody = &buf

	jira.Log.Println("DONE", method, absUrl)
	return
}

func (jira *Jira) GetReleases() (releases []JiraRelease, err error) {
	url := fmt.Sprintf("/project/%s/versions", jira.Project)
	resp, err := jira.request("GET", url, nil)
	if err != nil {
		return
	}
	err = json.NewDecoder(resp).Decode(&releases)
	if err != nil {
		return
	}

	return
}

func (jira *Jira) GetRelease(id string) {

}
