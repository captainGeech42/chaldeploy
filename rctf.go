package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

/*
r = requests.get(BASE_URL + "/api/v1/integrations/ctftime/leaderboard", headers={"Authorization": f"Bearer {auth_token}"})
*/

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

// Validate the login token from the user and get a auth token back
// If there is an error getting an auth token, returns (nil, error)
// If comms are successful but auth is bad, returns ("", nil)
// Otherwise, returns (authToken, nil)
func authToRctf(loginToken string) (string, error) {
	reqBody, err := json.Marshal(map[string]string{
		"teamToken": loginToken,
	})

	if err != nil {
		return "", err
	}

	resp, err := http.Post(RCTF_SERVER+"/api/v1/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

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
