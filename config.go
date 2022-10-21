package main

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type Config struct {
	// $CHALDEPLOY_NAME: Name of the challenge to deploy
	// TODO: render this on the webpage
	ChallengeName string `env:"CHALDEPLOY_NAME"`

	// $CHALDEPLOY_PORT: Port exposed by the challenge, must be 1-65535
	ChallengePort int `env:"CHALDEPLOY_PORT"`

	// $CHALDEPLOY_IMAGE: Image path for the challenge
	ChallengeImage string `env:"CHALDEPLOY_IMAGE"`

	// $CHALDEPLOY_SESSION_KEY: Secret key used to authenticate session data. Must be 32 or 64 chars long
	SessionKey string `env:"CHALDEPLOY_SESSION_KEY"`

	// $CHALDEPLOY_RCTF_SERVER: rCTF server to auth against
	RctfServer string `env:"CHALDEPLOY_RCTF_SERVER"`

	// $CHALDEPLOY_K8SCONFIG (optional): Path to the k8s config. If not set, k8s config will be loaded from /var/run/secrets or ~/.kube
	K8sConfigPath string `env:"CHALDEPLOY_K8SCONFIG,optional"`
}

// Load the config from env vars. Supports int and string types, along with an 'optional' modifier
// ref:
//   - https://linuxhint.com/golang-struct-tags/
//   - https://stackoverflow.com/a/6396678
func loadConfig() (*Config, error) {
	// init an empty config
	config := Config{}

	// loop over each field in the struct
	t := reflect.TypeOf(config)
	for i := 0; i < t.NumField(); i++ {
		// get the tag data
		f := t.Field(i)
		tag, ok := f.Tag.Lookup("env")
		if !ok {
			return nil, fmt.Errorf("config struct has an invalid field: %s", f.Name)
		}

		// split the tag data
		tagParts := strings.Split(tag, ",")

		// get the env data
		data := os.Getenv(tagParts[0])

		// make sure it's set if not optional
		if data != "" || Contains(tagParts[1:], "optional") {
			// set the value
			if f.Type.Kind() == reflect.Int {
				// need to save as an int
				if intVal, err := strconv.Atoi(data); err != nil {
					return nil, fmt.Errorf("couldn't convert value to integer: %s", data)
				} else {
					reflect.ValueOf(&config).Elem().Field(i).Set(reflect.ValueOf(intVal))
				}
			} else {
				// can save as a string
				reflect.ValueOf(&config).Elem().Field(i).Set(reflect.ValueOf(data))
			}
		} else {
			// a value was needed, error
			return nil, fmt.Errorf("a necessary environment variable was not set: $%s", tagParts[0])
		}
	}

	return &config, nil
}
