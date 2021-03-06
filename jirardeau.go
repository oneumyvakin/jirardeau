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

const (
	// IssueTypeBug holds type id
	IssueTypeBug = "1"
	// IssueTypeFeature holds type id
	IssueTypeFeature = "2"
	// IssueTypeTask holds type id
	IssueTypeTask = "3"
	// IssueTypeImprovement holds type id
	IssueTypeImprovement = "4"
	// IssueTypeSubTask holds type id
	IssueTypeSubTask = "5"
	// IssueTypeSpecSubTask holds type id
	IssueTypeSpecSubTask = "6"
	// IssueTypeDevSubTask holds type id
	IssueTypeDevSubTask = "7"
	// IssueTypeQaSubTask holds type id
	IssueTypeQaSubTask = "8"
	// IssueTypePmTask holds type id
	IssueTypePmTask = "9"
	// IssueTypeDevTask holds type id
	IssueTypeDevTask = "10"
	// IssueTypeQaTask holds type id
	IssueTypeQaTask = "11"
	// IssueTypeUserStory holds type id
	IssueTypeUserStory = "12"
	// IssueTypeDocSubTask holds type id
	IssueTypeDocSubTask = "13"
	// IssueTypeDocTask holds type id
	IssueTypeDocTask = "14"
	// IssueTypeEpic holds type id
	IssueTypeEpic = "15"
	// IssueTypeStory holds type id
	IssueTypeStory = "16"
	// IssueTypeFeatureRequest holds type id
	IssueTypeFeatureRequest = "17"
	// IssueTypeMetaFeature holds type id
	IssueTypeMetaFeature = "18"
	// IssueTypeChangeRequest holds type id
	IssueTypeChangeRequest = "19"
	// IssueTypeVulnerability holds type id
	IssueTypeVulnerability = "20"
	// IssueTypeBuildingProblem holds type id
	IssueTypeBuildingProblem = "21"
	// IssueTypeTechTask holds type id
	IssueTypeTechTask = "22"
	// IssueTypeUxStory holds type id
	IssueTypeUxStory = "23"
	// IssueTypeScenario holds type id
	IssueTypeScenario = "24"
	// IssueTypePostTask holds type id
	IssueTypePostTask = "25"
)

// Jira holds Url like https://jira.tld
type Jira struct {
	Log       *log.Logger
	Login     string
	Password  string
	Project   string
	ProjectID string
	URL       string
}

// Project holds JIRA Project
type Project struct {
	ID   string `json:"id,omitempty"`
	Self string `json:"self,omitempty"`
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
}

// FixVersion holds JIRA Version
// Fields field used to customize issue fields
type FixVersion struct {
	Archived        bool   `json:"archived"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	Overdue         bool   `json:"overdue"`
	ProjectID       int    `json:"projectId"`
	ReleaseDate     string `json:"releaseDate"`
	Released        bool   `json:"released"`
	Self            string `json:"self"`
	StartDate       string `json:"startDate"`
	UserReleaseDate string `json:"userReleaseDate"`
	UserStartDate   string `json:"userStartDate"`
	Fields          string `json:"-"`
}

// Issue holds issue data
type Issue struct {
	ID     string            `json:"id"`
	Self   string            `json:"self"`
	Key    string            `json:"key"`
	Fields *IssueFields       `json:"fields"`
	Expand string            `json:"expand"`
	Names  map[string]string `json:"names"`
}

// IssueFields holds default fields
type IssueFields struct {
	Project      *Project      `json:"project"`
	Summary      string       `json:"summary"`
	IssueType    *IssueType    `json:"issuetype"`
	FixVersions  []*FixVersion `json:"fixVersions"`
	Status       Status       `json:"status"`
	Created      string       `json:"created"`
	Description  string       `json:"description"`
	Comment      CommentField `json:"comment"`
	CustomFields CustomField  `json:"-"`
}

// CustomField holds custom field name and value
type CustomField map[string]string

// IssueType describes Issue type
type IssueType struct {
	ID          string `json:"id"`
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
	ID           string `json:"id"`
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
	ID          string `json:"id"`
	Self        string `json:"self"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// RequestCreateIssue creates issue
type RequestCreateIssue struct {
	Fields ModifyIssueFields `json:"fields"`
}

// RequestUpdateIssue creates issue
type RequestUpdateIssue struct {
	Key    string            `json:"key"`
	Fields ModifyIssueFields `json:"fields"`
}

// ModifyIssueFields used only for creating issues
type ModifyIssueFields struct {
	Project      *Project      `json:"project,omitempty"`
	Summary      string       `json:"summary,omitempty"`
	IssueType    *IssueType    `json:"issuetype,omitempty"`
	FixVersions  []*FixVersion `json:"fixVersions,omitempty"`
	Description  string       `json:"description,omitempty"`
	CustomFields CustomField  `json:"-"`
}

func (jira *Jira) request(method, relURL string, reqBody io.Reader) (respBody io.Reader, err error) {
	absURL, err := url.Parse(jira.URL + relURL)
	if err != nil {
		err = fmt.Errorf("Failed to parse %s and %s to URL: %s", jira.URL, relURL, err)
		jira.Log.Println(err)
		return
	}
	jira.Log.Println("STRT", method, absURL.String())

	req, err := http.NewRequest(method, absURL.String(), reqBody)
	if err != nil {
		err = fmt.Errorf("Failed to build HTTP request %s %s: %s", method, absURL.String(), err)
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
			err = fmt.Errorf("Failed to read response from JIRA request %s %s: %s", method, absURL.String(), err)
			jira.Log.Println(err)
			return
		}
		respBody = &buf
		switch {
		case resp.StatusCode == 401:
			err = fmt.Errorf("Failed to JIRA request %s %s with HTTP code %d: Unauthorized (401)", method, absURL.String(), resp.StatusCode)
			jira.Log.Println(err)
			return
		case resp.StatusCode == 404:
			err = fmt.Errorf("Failed to JIRA request %s %s with HTTP code %d: Wrong request", method, absURL.String(), resp.StatusCode)
			jira.Log.Println(err)
			return
		case resp.StatusCode == 405:
			err = fmt.Errorf("Failed to JIRA request %s %s with HTTP code %d: HTTP method is not allowed for the requested resource", method, absURL.String(), resp.StatusCode)
			jira.Log.Println(err)
			return
		case resp.StatusCode == 415:
			err = fmt.Errorf("Failed to JIRA request %s %s with HTTP code %d: Unsupported Media Type", method, absURL.String(), resp.StatusCode)
			jira.Log.Println(err)
			return
		case resp.StatusCode == 502:
			err = fmt.Errorf("Failed to JIRA request %s %s with HTTP code %d: Bad gateway", method, absURL.String(), resp.StatusCode)
			jira.Log.Println(err)
			return
		case resp.StatusCode >= 400:
			err = fmt.Errorf("Failed to JIRA request %s %s with HTTP code %d: %s", method, absURL.String(), resp.StatusCode, buf.String())
			jira.Log.Println(err)
			return
		}
	}

	if err != nil {
		err = fmt.Errorf("Failed to JIRA request %s %s: %s", method, absURL.String(), err)
		jira.Log.Println(err)
		return
	}

	jira.Log.Println("StatusCode:", resp.StatusCode)
	jira.Log.Println("Headers:", resp.Header)

	jira.Log.Println("DONE", method, absURL.String())
	return
}

// GetFixVersions returns versions of Jira.Project
// https://docs.atlassian.com/jira/REST/6.1/#d2e3195
func (jira *Jira) GetFixVersions() (releases []FixVersion, err error) {
	relURL := fmt.Sprintf("/project/%s/versions", jira.Project)
	resp, err := jira.request("GET", relURL, nil)
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
// https://docs.atlassian.com/jira/REST/6.1/#d2e4071
func (jira *Jira) GetIssues(fixVersion FixVersion) (issues map[string]Issue, err error) {
	var result struct {
		Issues []Issue `json:"issues"`
	}

	parameters := url.Values{}
	parameters.Add("jql", fmt.Sprintf(`project = %s AND fixVersion = "%s"`, jira.Project, fixVersion.Name))
	if fixVersion.Fields == "" {
		parameters.Add("fields", "id,key,self,summary,issuetype,status,description,created,comment")
	} else {
		parameters.Add("fields", fixVersion.Fields)
	}

	relURL := fmt.Sprintf("/search?%s", parameters.Encode())

	resp, err := jira.request("GET", relURL, nil)
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
// https://docs.atlassian.com/jira/REST/6.1/#d2e1160
func (jira *Jira) GetIssue(id string, expand []string) (issue Issue, err error) {
	parameters := url.Values{}
	if expand != nil {
		parameters.Add("expand", strings.Join(expand, ","))
	}

	relURL := fmt.Sprintf("/issue/%s?%s", id, parameters.Encode())

	resp, err := jira.request("GET", relURL, nil)
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
// https://docs.atlassian.com/jira/REST/6.1/#d2e865
func (jira *Jira) CreateIssue(request RequestCreateIssue) (issue Issue, err error) {
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return issue, errors.Wrap(err, "failed create issue")
	}

	resp, err := jira.request("POST", "/issue", &buf)
	if err != nil {
		return issue, errors.Wrap(err, "failed create issue")
	}

	err = json.NewDecoder(resp).Decode(&issue)
	if err != nil {
		return issue, errors.Wrap(err, "failed create issue, failed to decode response")
	}

	issue.Fields = &IssueFields{
		Description: request.Fields.Description,
		Project: request.Fields.Project,
		Summary: request.Fields.Summary,
		IssueType: request.Fields.IssueType,
		FixVersions: request.Fields.FixVersions,
		CustomFields: request.Fields.CustomFields,
	}

	return issue, nil
}

// UpdateIssue update existed issue with new fields values
// https://docs.atlassian.com/jira/REST/6.1/#d2e1209
func (jira *Jira) UpdateIssue(request RequestUpdateIssue) error {
	if request.Key == "" {
		return errors.New("failed update issue: issue Key is empty")
	}
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return errors.Wrap(err, "failed update issue")
	}

	_, err = jira.request("PUT", fmt.Sprintf("/issue/%s", request.Key), &buf)
	if err != nil {
		return errors.Wrap(err, "failed update issue")
	}

	return nil
}

// MarshalJSON encapsulate CustomFields in CreateIssueFields
// and handle JIRA's requirement of allowed fields for POST/PUT query
func (fields ModifyIssueFields) MarshalJSON() (resultBytes []byte, err error) {
	cf := make(map[string]CustomField)

	for key, val := range fields.CustomFields {
		subCf := make(CustomField)
		subCf["value"] = val
		cf[key] = subCf
	}

	var bytesCf []byte
	if len(cf) > 0 {
		bytesCf, err = json.Marshal(cf)
		fmt.Println("json.Marshal(cf)", string(bytesCf), err)
		if err != nil {
			return nil, err
		}
	}

	type AliasIssueFields struct {
		Project     *Project      `json:"project,omitempty"`
		Summary     string       `json:"summary,omitempty"`
		IssueType   *IssueType    `json:"issuetype,omitempty"`
		FixVersions []*FixVersion `json:"fixVersions,omitempty"`
		Description string       `json:"description,omitempty"`
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

	if len(bytesCf) > 0 {
		bytesCf = bytes.TrimSuffix(bytesCf, []byte("}"))
		bytesFields = bytes.TrimPrefix(bytesFields, []byte("{"))
		allFields := [][]byte{
			bytesCf,
			bytesFields,
		}
		resultBytes = bytes.Join(allFields, []byte(","))
	} else {
		resultBytes = bytesFields
	}

	return resultBytes, nil
}

// UnmarshalJSON gather custom fields values into CustomFields
func (fields *IssueFields) UnmarshalJSON(data []byte) (err error) {
	type AliasIssueFields IssueFields
	issueFields := AliasIssueFields{}
	err = json.Unmarshal(data, &issueFields)
	if err != nil {
		return
	}

	fields.Comment = issueFields.Comment
	fields.Status = issueFields.Status
	fields.Created = issueFields.Created
	fields.Description = issueFields.Description
	fields.FixVersions = issueFields.FixVersions
	fields.IssueType = issueFields.IssueType
	fields.Project = issueFields.Project

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
						case string:
							fields.CustomFields[key] = subVal.(string)
						}
					}
				}
			case string:
				fields.CustomFields[key] = val.(string)
			case nil:
				fields.CustomFields[key] = ""
			}
		}
	}

	return
}
