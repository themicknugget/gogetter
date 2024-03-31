package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// downloadFile downloads a file from the remote server to the local directory and then deletes the remote file.
// It now replicates the directory structure of the source system on the destination system.
func downloadFile(client *ssh.Client, remoteFilePath string, config DirectoryPair) {
	// Create a new SFTP client
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		fmt.Printf("Failed to create SFTP client: %v\n", err)
		return
	}
	defer sftpClient.Close()

	// Open the remote file
	remoteFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		fmt.Printf("Failed to open remote file %s: %v\n", remoteFilePath, err)
		return
	}
	defer remoteFile.Close()

	// Ensure the local directory structure exists
	localFilePath := computeLocalFilePath(remoteFilePath, config)
	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
		fmt.Printf("Failed to create local directories for '%s': %v\n", localFilePath, err)
		return
	}

	// Create the local file
	localFile, err := os.Create(localFilePath)
	if err != nil {
		fmt.Printf("Failed to create local file '%s': %v\n", localFilePath, err)
		return
	}
	defer localFile.Close()

	// Copy the file from remote to local
	bytesCopied, err := remoteFile.WriteTo(localFile)
	if err != nil {
		fmt.Printf("Failed to copy file from remote to local: %v\n", err)
		return
	}

	fmt.Printf("Successfully downloaded %s to %s (%d bytes copied)\n", remoteFilePath, localFilePath, bytesCopied)
}

func computeLocalFilePath(remoteFilePath string, config DirectoryPair) string {
	relativePath, err := filepath.Rel(config.RemoteDirectory, remoteFilePath)
	if err != nil {
		// Handle the error, e.g., log it or default to using the base name of the remote file.
		fmt.Printf("Error computing relative path for '%s': %v\n", remoteFilePath, err)
		// Fallback to base name if the relative path cannot be computed.
		return filepath.Join(config.LocalDirectory, filepath.Base(remoteFilePath))
	}
	return filepath.Join(config.LocalDirectory, relativePath)
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
		fmt.Println("No files modified in the last minute.")
		return
	}

	for _, filePath := range filePaths {
		if filePath != "" {
			downloadFile(connection, filePath, pair)
		}
	}
}
