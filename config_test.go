package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFullConfig(t *testing.T) {
	t.Setenv("CHALDEPLOY_NAME", "test chal name")
	t.Setenv("CHALDEPLOY_PORT", "12345")
	t.Setenv("CHALDEPLOY_IMAGE", "testimg:latest")
	t.Setenv("CHALDEPLOY_SESSION_KEY", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("CHALDEPLOY_RCTF_SERVER", "https://2021.redpwn.net")
	t.Setenv("CHALDEPLOY_K8SCONFIG", "/asdf/zxcv")

	config, err := loadConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "test chal name", config.ChallengeName)
	assert.Equal(t, 12345, config.ChallengePort)
	assert.Equal(t, "testimg:latest", config.ChallengeImage)
	assert.Equal(t, "https://2021.redpwn.net", config.RctfServer)
	assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", config.SessionKey)
	assert.Equal(t, "/asdf/zxcv", config.K8sConfigPath)
}

func TestPartialConfig(t *testing.T) {
	t.Setenv("CHALDEPLOY_NAME", "test chal name")
	t.Setenv("CHALDEPLOY_PORT", "12345")
	t.Setenv("CHALDEPLOY_IMAGE", "testimg:latest")
	t.Setenv("CHALDEPLOY_RCTF_SERVER", "https://2021.redpwn.net")
	t.Setenv("CHALDEPLOY_SESSION_KEY", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	config, err := loadConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config)

	assert.Equal(t, "test chal name", config.ChallengeName)
	assert.Equal(t, 12345, config.ChallengePort)
	assert.Equal(t, "testimg:latest", config.ChallengeImage)
	assert.Equal(t, "https://2021.redpwn.net", config.RctfServer)
	assert.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", config.SessionKey)
	assert.Equal(t, "", config.K8sConfigPath)
}

func TestInvalidConfig(t *testing.T) {
	t.Setenv("CHALDEPLOY_NAME", "test chal name")
	t.Setenv("CHALDEPLOY_PORT", "12345")
	t.Setenv("CHALDEPLOY_SESSION_KEY", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	config, err := loadConfig()
	assert.NotNil(t, err)
	assert.Nil(t, config)
}

func TestInvalidPortConfig(t *testing.T) {
	t.Setenv("CHALDEPLOY_NAME", "test chal name")
	t.Setenv("CHALDEPLOY_PORT", "zzz")
	t.Setenv("CHALDEPLOY_IMAGE", "testimg:latest")
	t.Setenv("CHALDEPLOY_RCTF_SERVER", "https://2021.redpwn.net")
	t.Setenv("CHALDEPLOY_SESSION_KEY", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	config, err := loadConfig()
	assert.NotNil(t, err)
	assert.Nil(t, config)
}
