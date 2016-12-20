package jirardeau

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
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

// Project holds JIRA Project
type Project struct {
	Id   string `json:"id"`
	Self string `json:"self"`
	Key  string `json:"key"`
	Name string `json:"name"`
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

// Issue holds issue data
type Issue struct {
	Id     string            `json:"id"`
	Self   string            `json:"self"`
	Key    string            `json:"key"`
	Fields IssueFields       `json:"fields"`
	Expand string            `json:"expand"`
	Names  map[string]string `json:"names"`
}

// IssueFields
type IssueFields struct {
	Project      Project      `json:"project"`
	Summary      string       `json:"summary"`
	IssueType    IssueType    `json:"issuetype"`
	FixVersions  []FixVersion `json:"fixVersions"`
	Status       Status       `json:"status"`
	Created      string       `json:"created"`
	Description  string       `json:"description"`
	Comment      CommentField `json:"comment"`
	CustomFields CustomField  `json:"-"`
}

// CustomField
type CustomField map[string]string

// IssueType
type IssueType struct {
	Id          string `json:"id"`
	Self        string `json:"self"`
	Name        string `json:"name"`
	SubTask     bool   `json:"subtask"`
	Description string `json:"description"`
}

// CommentField holds Issue Comments
type CommentField struct {
	StartAt    int       `json:"startAt"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	Comments   []Comment `json:"comments"`
}

// Comment of Issue
type Comment struct {
	Id           string `json:"id"`
	Self         string `json:"self"`
	Author       Author `json:"author"`
	UpdateAuthor Author `json:"updateAuthor"`
	Body         string `json:"body"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
}

// Author of Issue or Comment
type Author struct {
	Self         string `json:"self"`
	Active       bool   `json:"active"`
	Name         string `json:"name"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

// Status of Issue
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

	var buf bytes.Buffer
	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()

		_, err = buf.ReadFrom(resp.Body)
		if err != nil {
			err = fmt.Errorf("Failed to read response from JIRA request %s %s: %s", method, absUrl.String(), err)
			jira.Log.Println(err)
			return
		}
		respBody = &buf

		if resp.StatusCode >= 400 {
			err = fmt.Errorf("Failed to JIRA request %s %s with HTTP code %d: %s", method, absUrl.String(), resp.StatusCode, buf.String())
			jira.Log.Println(err)
			return
		}
	}

	if err != nil {
		err = fmt.Errorf("Failed to JIRA request %s %s: %s", method, absUrl.String(), err)
		jira.Log.Println(err)
		return
	}

	jira.Log.Println("StatusCode:", resp.StatusCode)
	jira.Log.Println("Headers:", resp.Header)

	jira.Log.Println("DONE", method, absUrl.String())
	return
}

// GetFixVersions returns versions of Jira.Project
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
func (jira *Jira) GetIssues(fixVersion FixVersion) (issues map[string]Issue, err error) {
	var result struct {
		Issues []Issue `json:"issues"`
	}

	parameters := url.Values{}
	parameters.Add("jql", fmt.Sprintf(`project = %s AND fixVersion = "%s"`, jira.Project, fixVersion.Name))
	parameters.Add("fields", "id,key,self,summary,issuetype,status,description,created")
	relUrl := fmt.Sprintf("/search?%s", parameters.Encode())

	resp, err := jira.request("GET", relUrl, nil)
	if err != nil {
		return
	}
	err = json.NewDecoder(resp).Decode(&result)
	if err != nil {
		err = errors.Wrap(err, "decode failed")
		return
	}

	issues = make(map[string]Issue)
	for _, issue := range result.Issues {
		issues[issue.Key] = issue
	}

	return
}

// GetIssue by id/key
func (jira *Jira) GetIssue(id string, expand []string) (issue Issue, err error) {
	parameters := url.Values{}
	if expand != nil {
		parameters.Add("expand", strings.Join(expand, ","))
	}

	relUrl := fmt.Sprintf("/issue/%s?%s", id, parameters.Encode())

	resp, err := jira.request("GET", relUrl, nil)
	if err != nil {
		return
	}

	err = json.NewDecoder(resp).Decode(&issue)
	if err != nil {
		err = errors.Wrap(err, "decode failed")
		return
	}

	return
}

// CreateIssue creates issue based on filled fields
func (jira *Jira) CreateIssue(issue Issue) (Issue, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(issue)
	if err != nil {
		return Issue{}, errors.Wrap(err, "failed create issue")
	}

	resp, err := jira.request("POST", "/issue", &buf)
	if err != nil {
		return Issue{}, errors.Wrap(err, "failed create issue")
	}

	err = json.NewDecoder(resp).Decode(&issue)
	if err != nil {
		return Issue{}, errors.Wrap(err, "failed create issue, failed to decode response")
	}

	return issue, nil
}

func (fields IssueFields) MarshalJSON() ([]byte, error) {
	cf := make(map[string]CustomField)

	for key, val := range fields.CustomFields {
		subCf := make(CustomField)
		subCf["value"] = val
		cf[key] = subCf
	}

	bytesCf, err := json.Marshal(cf)
	if err != nil {
		return nil, err
	}

	type AliasIssueFields  struct {
		Project      Project      `json:"project"`
		Summary      string       `json:"summary"`
		IssueType    IssueType    `json:"issuetype"`
		FixVersions  []FixVersion `json:"fixVersions"`
		Description  string       `json:"description"`
	}

	issueFields := AliasIssueFields{}
	issueFields.Description = fields.Description
	issueFields.FixVersions = fields.FixVersions
	issueFields.IssueType = fields.IssueType
	issueFields.Project = fields.Project
	issueFields.Summary = fields.Summary

	bytesFields, err := json.Marshal(issueFields)
	if err != nil {
		return nil, err
	}
	bytesCf = bytes.TrimSuffix(bytesCf, []byte("}"))
	bytesFields = bytes.TrimPrefix(bytesFields, []byte("{"))

	allFields := [][]byte{
		bytesCf,
		bytesFields,
	}
	allBytes := bytes.Join(allFields, []byte(","))

	return allBytes, nil
}

func (fields *IssueFields) UnmarshalJSON(data []byte) (err error) {
	type AliasIssueFields IssueFields
	issueFields := AliasIssueFields{}
	err = json.Unmarshal(data, &issueFields)
	if err != nil {
		return
	}

	fields.Comment = issueFields.Comment
	fields.Created = issueFields.Created
	fields.Description = issueFields.Description
	fields.FixVersions = issueFields.FixVersions
	fields.IssueType = issueFields.IssueType
	fields.Project = issueFields.Project
	fields.Status = issueFields.Status
	fields.Summary = issueFields.Summary

	cf := make(map[string]interface{})

	err = json.Unmarshal(data, &cf)
	if err != nil {
		return
	}

	if fields.CustomFields == nil {
		fields.CustomFields = make(CustomField)
	}

	for key, val := range cf {
		if strings.HasPrefix(key, "customfield_") {

			switch val.(type) {
			case map[string]interface{}:
				for subKey, subVal := range val.(map[string]interface{}) {
					if strings.HasPrefix(subKey, "value") {
						switch subVal.(type) {
							case string: fields.CustomFields[key] = subVal.(string)
						}
					}
				}
			case string: fields.CustomFields[key] = val.(string)
			case nil: fields.CustomFields[key] = ""
			}
		}
	}

	return
}