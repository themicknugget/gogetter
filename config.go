package main

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Hosts []HostConfig `yaml:"hosts"`
	UID   int          `yaml:"uid,omitempty"` // Optional
	GID   int          `yaml:"gid,omitempty"` // Optional
}

type HostConfig struct {
	SSHServer       string `yaml:"ssh_server"`
	SSHPort         int    `yaml:"ssh_port"`
	SSHUser         string `yaml:"ssh_user"`
	SSHKey          string `yaml:"ssh_key"`
	HostKey         string `yaml:"host_key,omitempty"`
	RemoteDirectory string `yaml:"remote_directory"`
	LocalDirectory  string `yaml:"local_directory"`
	Interval        int    `yaml:"interval"` // Interval in seconds
}

func LoadConfig(path string) (*Config, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	var config Config
	if err = yaml.Unmarshal(configFile, &config); err != nil {
		return nil, fmt.Errorf("error unmarshaling configuration: %w", err)
	}

	// Override UID/GID from environment variables if present
	if uid, present := os.LookupEnv("IPUID"); present {
		if parsedUID, err := strconv.Atoi(uid); err == nil {
			config.UID = parsedUID
		} else {
			fmt.Printf("Warning: Could not parse IPUID='%s' as integer\n", uid)
		}
	}

	if gid, present := os.LookupEnv("PGID"); present {
		if parsedGID, err := strconv.Atoi(gid); err == nil {
			config.GID = parsedGID
		} else {
			fmt.Printf("Warning: Could not parse PGID='%s' as integer\n", gid)
		}
	}

	return &config, nil
}
