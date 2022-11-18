package main

import (
	"encoding/json"
	"log"
	"os"
)

func readJsonMap(filepath string) (map[string]any, error) {
	var result = make(map[string]interface{})
	input, err := os.ReadFile(filepath)
	if err != nil {
		log.Println(err)
	}
	if err := json.Unmarshal(input, &result); err != nil {
		log.Println(err)
	}
	return result, err
}
