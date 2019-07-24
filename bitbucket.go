package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/xerrors"
)

type Status string

const (
	StatusNotSet  Status = ""
	StatusSuccess Status = "success"
	StatusFail    Status = "fail"
)

func getBuildStatus(username, password, repo string) (Status, error) {
	uri := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%v/pipelines/?sort=-created_on&pagelen=1", repo)
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

	var output struct {
		Values []struct {
			State struct {
				Result struct {
					Name string
				}
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return StatusNotSet, xerrors.Errorf("unable to decode bitbucket response: %w", err)
	}

	if len(output.Values) == 0 || output.Values[0].State.Result.Name != "SUCCESSFUL" {
		return StatusFail, nil
	}

	return StatusSuccess, nil
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
