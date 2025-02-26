package deepseek

import (
	"fmt"
	"net/http"
	"strings"
	"io/ioutil"
	"time"
)

func DSList() {
	url := "https://api.deepseek.com/models"
	method := "GET"
	client := &http.Client {
	}
	rq, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	rq.Header.Add("Accept", "application/json")
	rq.Header.Add("Authorization", "Bearer sk-e0b79d8ab91c428b9948ec1e960c8cf3")

	rs, err := client.Do(rq)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rs.Body.Close()

	body, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

func DSChat(question string) {
	url := "https://api.deepseek.com/chat/completions"
	method := "POST"

	payload :=strings.NewReader(`{
		"model": "deepseek-reasoner",
		"messages": [
			{"role": "user", "content": "` + question + `"}
		],
		"frequency_penalty": 0,
		"max_tokens": 4098,
		"presence_penalty": 0,
		"response_format": {
			"type": "text"
		},
		"stop": null,
		"stream": false,
		"temperature": 1,
		"top_p": 1,
		"tools": null,
		"tool_choice": "none",
		"top_logprobs": null
	}`)

	client := &http.Client {
		Timeout: 30*time.Second,
	}
	rq, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println(err)
		return
	}
	rq.Header.Add("Accept", "application/json")
	rq.Header.Add("Content-Type", "application/json")
	rq.Header.Add("Authorization", "Bearer sk-e0b79d8ab91c428b9948ec1e960c8cf3")
	rs, err := client.Do(rq)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rs.Body.Close()
	body, err := ioutil.ReadAll(rs.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}