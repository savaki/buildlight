package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/xerrors"
)

type Status string

const (
	StatusNotSet     Status = ""
	StatusSuccessful Status = "success"
	StatusInProgress Status = "in-progress"
	StatusFailed     Status = "fail"
)

func getBuildStatus(username, password, repo string) (Status, error) {
	uri := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%v/pipelines/?sort=-created_on&pagelen=10", repo)
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return StatusNotSet, xerrors.Errorf("unable to construct get request: %w", err)
	}
	req.SetBasicAuth(username, password)

	if opts.Debug {
		fmt.Printf("polling %v\n", uri)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return StatusNotSet, xerrors.Errorf("unable to retrieve status: %w", err)
	}
	defer resp.Body.Close()

	if opts.Debug {
		fmt.Printf("%v\n", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return StatusNotSet, xerrors.Errorf("unable to read build contents: %w", err)
	}

	var output struct {
		Values []struct {
			State struct {
				Name   string
				Result struct {
					Name string
				}
			}
		}
	}
	if err := json.Unmarshal(data, &output); err != nil {
		return StatusNotSet, xerrors.Errorf("unable to decode bitbucket response: %w", err)
	}

	if len(output.Values) == 0 {
		if opts.Debug {
			fmt.Println(string(data))
		}
		return StatusFailed, nil
	}

	// when a build is pending, it will have no status assigned to it
	var stateName string
	for _, value := range output.Values {
		if v := value.State.Name; v == "IN_PROGRESS" {
			stateName = v
			break
		}
		if v := value.State.Result.Name; v != "" {
			stateName = v
			break
		}
	}

	if opts.Debug {
		fmt.Println("got state name,", stateName)
	}

	switch stateName {
	case "SUCCESSFUL":
		return StatusSuccessful, nil

	case "IN_PROGRESS":
		return StatusInProgress, nil

	case "FAILED":
		return StatusFailed, nil

	default:
		fmt.Println("received unexpected state name from bitbucket,", stateName)
		return StatusFailed, nil
	}
}

func pollBuildStatus(username, password, repo string, interval time.Duration, updateFunc func(Status)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		status, err := getBuildStatus(username, password, repo)
		if err != nil {
			fmt.Printf("failed to get build status: %v\n", err)
			continue
		}

		updateFunc(status)

		select {
		case <-ticker.C:
		}
	}
}
