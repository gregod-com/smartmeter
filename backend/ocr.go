package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ocrDecodeImage(imageData string) (string, error) {
	var payloadObj OpenAIPayload
	payloadObj.Model = config.ChatModel
	payloadObj.Messages = []Message{
		{
			Role: "user",
			Content: []Content{
				{
					Type: "text",
					Text: config.UserPrompt,
				},
				{
					Type: "image_url",
					ImageURL: ImageURL{
						URL: imageData,
					},
				},
			},
		},
	}
	payloadObj.MaxTokens = 300

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return "", err
	}
	payload := string(payloadBytes)

	log.Debug("calling openai")

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(payload))
	if err != nil {
		log.Info(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.OpenaiAPIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var openAIResponse OpenAIResponse
	err = json.Unmarshal(body, &openAIResponse)
	if err != nil {
		return "", err
	}

	// TODO: check the response for error codes and if the value is a digit
	if len(openAIResponse.Choices) > 0 {
		log.Debug("openAIResponse:", openAIResponse.Choices[0].Message.Content)
		return openAIResponse.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("No choices found %v", string(body))
}
