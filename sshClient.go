package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
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

// FindAndDownloadFiles searches for files modified in the last minute and downloads them.
func FindAndDownloadFiles(connection *ssh.Client, pair DirectoryPair) {
	session, err := connection.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session: %v\n", err)
		return
	}
	defer session.Close()

	// Find files modified more than a minute ago
	cmd := fmt.Sprintf("find %s -type f -mmin +1", pair.RemoteDirectory)
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		fmt.Printf("Failed to find files: %v\n", err)
		return
	}

	filePaths := strings.Split(string(output), "\n")
	if len(filePaths) == 0 || (len(filePaths) == 1 && filePaths[0] == "") {
		return
	}

	for _, filePath := range filePaths {
		if filePath != "" {
			downloadFile(connection, filePath, pair)
		}
	}
}
