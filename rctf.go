package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Fields always present in an API response from rCTF
type RctfResponse struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// Data from /api/v1/auth/login
type RctfAuthData struct {
	AuthToken string `json:"authToken"`
}

// Response to /api/v1/auth/login
type RctfAuthResponse struct {
	RctfResponse
	Data RctfAuthData `json:"data"`
}

// Partial struct for the data from /api/v1/users/me
type RctfUserInfoData struct {
	TeamName string `json:"name"`
	Id       string `json:"id"`
}

// Response to /api/v1/users/me
type RctfUserInfoResponse struct {
	RctfResponse
	Data RctfUserInfoData `json:"data"`
}

// Validate the login token from the user and get a auth token back
// If there is an error getting an auth token, returns (nil, error)
// If comms are successful but auth is bad, returns ("", nil)
// Otherwise, returns (authToken, nil)
func authToRctf(loginToken string) (string, error) {
	if config == nil {
		return "", errors.New("config global isn't set")
	}

	reqBody, err := json.Marshal(map[string]string{
		"teamToken": loginToken,
	})

	if err != nil {
		return "", err
	}

	resp, err := http.Post(config.RctfServer+"/api/v1/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	rctfResp := RctfAuthResponse{}
	err = json.Unmarshal(respBody, &rctfResp)
	if err != nil {
		return "", err
	}

	if rctfResp.Kind != "goodLogin" {
		return "", nil
	}

	return rctfResp.Data.AuthToken, nil
}

// Get user info from the rCTF API
func getUserInfo(authToken string) (*RctfUserInfoData, error) {
	if config == nil {
		return nil, errors.New("config global isn't set")
	}

	req, err := http.NewRequest(http.MethodGet, config.RctfServer+"/api/v1/users/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+authToken)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rctfResp := RctfUserInfoResponse{}
	err = json.Unmarshal(respBody, &rctfResp)
	if err != nil {
		return nil, err
	}

	if rctfResp.Kind != "goodUserData" {
		return nil, fmt.Errorf("got bad data from rCTF api (%s): %s", rctfResp.Kind, rctfResp.Message)
	}

	return &rctfResp.Data, nil
}
