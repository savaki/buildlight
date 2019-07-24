package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/xerrors"
)

func discover() (string, error) {
	resp, err := http.Get("https://discovery.meethue.com")
	if err != nil {
		return "", xerrors.Errorf("unable to discovery hue ip: %w", err)
	}
	defer resp.Body.Close()

	var output []struct {
		Internalipaddress string
	}
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return "", xerrors.Errorf("unable to decode hue discovery info: %w", err)
	}
	if len(output) == 0 {
		return "", xerrors.Errorf("no hue bridges found")
	}

	return output[0].Internalipaddress, nil
}

func setColor(apiKey, addr string, hue int64) error {
	type content struct {
		Hue int64 `json:"hue"`
		On  bool  `json:"on"`
		Bri int64 `json:"bri"`
	}

	data, err := json.Marshal(content{
		Hue: hue,
		On:  true,
		Bri: 150,
	})
	if err != nil {
		return xerrors.Errorf("unable to marshal request: %w", err)
	}

	uri := fmt.Sprintf("http://%v/api/%v/lights/3/state", addr, apiKey)
	req, err := http.NewRequest(http.MethodPut, uri, bytes.NewReader(data))
	if err != nil {
		return xerrors.Errorf("unable to construct request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return xerrors.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()

	if opts.Debug {
		fmt.Println(resp.Status)
	}

	return nil
}

func manageColor(apiKey, addr string, green, red, yellow int64) func(Status) {
	ch := make(chan Status, 1)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		status := StatusNotSet
		lastStatus := StatusNotSet
		highlight := false

		for {
			select {
			case <-ticker.C:
				if status == StatusSuccess {
					if status == lastStatus {
						// do nothing
					} else if err := setColor(apiKey, addr, green); err != nil {
						fmt.Println(err)
					} else if opts.Debug {
						fmt.Println("changing to green")
					}

				} else if status == StatusFail {
					if highlight {
						if err := setColor(apiKey, addr, red); err != nil {
							fmt.Println(err)
						} else if opts.Debug {
							fmt.Println("changing to red")
						}
					} else {
						if err := setColor(apiKey, addr, yellow); err != nil {
							fmt.Println(err)
						} else if opts.Debug {
							fmt.Println("changing to light red")
						}
					}
					highlight = !highlight
				}
				lastStatus = status

			case newStatus := <-ch:
				status = newStatus
			}
		}
	}()

	return func(status Status) {
		fmt.Println("got", status)
		//if status == StatusSuccess {
		//	ch <- StatusFail
		//} else {
		//	ch <- StatusSuccess
		//}
		ch <- status
	}
}
