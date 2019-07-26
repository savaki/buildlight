package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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

func manageColor(apiKey, addr string, colors Colors) func(Event) {
	ch := make(chan Status, 1)

	go func() {
		ticker := time.NewTicker(750 * time.Millisecond)
		defer ticker.Stop()

		status := StatusNotSet
		lastStatus := StatusNotSet
		highlight := false

		for {
			select {
			case <-ticker.C:
				if status == StatusSuccessful {
					if status == lastStatus {
						// do nothing
					} else if err := setColor(apiKey, addr, colors.Green); err != nil {
						fmt.Println(err)
					} else if opts.Debug {
						fmt.Println("changing to green")
					}

				} else if status == StatusInProgress {
					if highlight {
						if err := setColor(apiKey, addr, colors.Purple); err != nil {
							fmt.Println(err)
						} else if opts.Debug {
							fmt.Println("changing to blue")
						}
					} else {
						if err := setColor(apiKey, addr, colors.DarkPurple); err != nil {
							fmt.Println(err)
						} else if opts.Debug {
							fmt.Println("changing to light purple")
						}
					}
					highlight = !highlight

				} else if status == StatusFailed {
					if highlight {
						if err := setColor(apiKey, addr, colors.Red); err != nil {
							fmt.Println(err)
						} else if opts.Debug {
							fmt.Println("changing to red")
						}
					} else {
						if err := setColor(apiKey, addr, colors.Yellow); err != nil {
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

	mutex := &sync.Mutex{}
	statuses := map[string]Status{}

	return func(event Event) {
		mutex.Lock()
		defer mutex.Unlock()

		statuses[event.Repo] = event.Status

		// Rules:
		// 1. if any failed -> red
		// 2. otherwise if any build -> blue
		// 3. else green
		for _, status := range []Status{StatusFailed, StatusInProgress} {
			for _, got := range statuses {
				if got == status {
					ch <- status
					return
				}
			}
		}

		ch <- StatusSuccessful
	}
}
