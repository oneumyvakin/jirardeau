package jirardeau

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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

// FixVersion holds JIRA Version
type FixVersion struct {
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

type Issue struct {
	Id     string      `json:"id"`
	Self   string      `json:"self"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary     string       `json:"summary"`
	IssueType   IssueType    `json:"issuetype"`
	FixVersions []FixVersion `json:"fixVersions"`
	Status      Status       `json:"status"`
	Created     string       `json:"created"`
	Description string       `json:"description"`
}

type IssueType struct {
	Id          string `json:"id"`
	Self        string `json:"self"`
	Name        string `json:"name"`
	SubTask     bool   `json:"subtask"`
	Description string `json:"description"`
}

type Status struct {
	Id          string `json:"id"`
	Self        string `json:"self"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (jira *Jira) request(method, relUrl string, reqBody io.Reader) (respBody io.Reader, err error) {
	absUrl, err := url.Parse(jira.Url + relUrl)
	if err != nil {
		err = fmt.Errorf("Failed to parse %s and %s to URL: %s", jira.Url, relUrl, err)
		jira.Log.Println(err)
		return
	}
	jira.Log.Println("STRT", method, absUrl.String())

	req, err := http.NewRequest(method, absUrl.String(), reqBody)
	if err != nil {
		err = fmt.Errorf("Failed to build HTTP request %s %s: %s", method, absUrl.String(), err)
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
		err = fmt.Errorf("Failed to JIRA request %s %s: %s", method, absUrl.String(), err)
		jira.Log.Println(err)
		return
	}

	jira.Log.Println("StatusCode:", resp.StatusCode)
	jira.Log.Println("Headers:", resp.Header)
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		err = fmt.Errorf("Failed to read response from JIRA request %s %s: %s", method, absUrl.String(), err)
		jira.Log.Println(err)
		return
	}
	respBody = &buf

	jira.Log.Println("DONE", method, absUrl.String())
	return
}

func (jira *Jira) GetFixVersions() (releases []FixVersion, err error) {
	relUrl := fmt.Sprintf("/project/%s/versions", jira.Project)
	resp, err := jira.request("GET", relUrl, nil)
	if err != nil {
		return
	}
	err = json.NewDecoder(resp).Decode(&releases)
	if err != nil {
		return
	}

	return
}

// GetIssues returns issues of fixVersion specified by FixVersion
func (jira *Jira) GetIssues(fixVersion FixVersion) ([]Issue, error) {
	var result struct {
		Issues []Issue `json:"issues"`
	}

	parameters := url.Values{}
	parameters.Add("jql", fmt.Sprintf(`project = %s AND fixVersion = "%s"`, jira.Project, fixVersion.Name))
	parameters.Add("fields", "id,key,self,summary,issuetype,status,description,created")
	relUrl := fmt.Sprintf("/search?%s", parameters.Encode())

	resp, err := jira.request("GET", relUrl, nil)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(resp).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Issues, nil
}
