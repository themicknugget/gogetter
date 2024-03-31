package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

func EstablishConnection(config HostConfig) (*ssh.Client, error) {
	key, err := os.ReadFile(config.SSHKey)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	var hostKeyCallback ssh.HostKeyCallback
	if config.HostKey != "" {
		// Decode the base64 encoded host key.
		hostKeyBytes, decodeErr := base64.StdEncoding.DecodeString(config.HostKey)
		if decodeErr != nil {
			return nil, fmt.Errorf("failed to decode host key: %v", decodeErr)
		}

		hostKey, parseErr := ssh.ParsePublicKey(hostKeyBytes)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse host key: %v", parseErr)
		}

		hostKeyCallback = ssh.FixedHostKey(hostKey)
	} else {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	sshConfig := &ssh.ClientConfig{
		User: config.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.SSHServer, config.SSHPort), sshConfig)
}

// MonitorHost continuously checks the specified host for new files, maintaining the SSH connection.
func MonitorHost(config HostConfig, shutdownCh <-chan struct{}) {
	var client *ssh.Client
	var err error

	for {
		select {
		case <-shutdownCh:
			if client != nil {
				client.Close()
			}
			fmt.Println("Shutdown signal received, closing SSH connection for", config.SSHServer)
			return
		default:
			if client == nil {
				client, err = EstablishConnection(config)
				if err != nil {
					fmt.Printf("Error connecting to %s: %v\n", config.SSHServer, err)
					time.Sleep(10 * time.Second)
					continue
				}
				fmt.Printf("Successfully connected to %s\n", config.SSHServer)
			}

			// New logic to handle multiple directory pairs
			for _, pair := range config.DirectoryPairs {
				// Example operation using pair.RemoteDirectory and pair.LocalDirectory
				FindAndDownloadFiles(client, pair)
			}
			time.Sleep(time.Duration(config.Interval) * time.Second)
		}
	}
}
