package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
)

func readJsonMapFile(filepath string) (map[string]any, error) {
	var result = make(map[string]interface{})
	input, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(input, &result); err != nil {
		return nil, err
	}
	return result, err
}

func readJsonMapHttp(url string) (map[string]any, error) {
	var result = make(map[string]interface{})
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, err
}
