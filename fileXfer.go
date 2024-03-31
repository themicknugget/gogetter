package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// downloadFile downloads a file from the remote server to the local directory and then deletes the remote file.
// It now replicates the directory structure of the source system on the destination system.
func downloadFile(connection *ssh.Client, remoteFilePath string, pair DirectoryPair) {
	session, err := connection.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session: %v\n", err)
		return
	}
	defer session.Close()

	// Open the remote file for reading
	stdout, err := session.StdoutPipe()
	if err != nil {
		fmt.Printf("Failed to open remote file: %v\n", err)
		return
	}

	if err := session.Start(fmt.Sprintf("cat %s", remoteFilePath)); err != nil {
		fmt.Printf("Failed to start file transfer: %v\n", err)
		return
	}

	// Replicates the directory structure relative to the remote base directory.
	relativePath := strings.TrimPrefix(remoteFilePath, pair.RemoteDirectory)
	localFilePath := filepath.Join(pair.LocalDirectory, relativePath)

	// Ensure the local directory structure exists.
	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
		fmt.Printf("Failed to create local directory structure: %v\n", err)
		return
	}

	localFile, err := os.Create(localFilePath)
	if err != nil {
		fmt.Printf("Failed to create local file: %v\n", err)
		return
	}
	defer localFile.Close()

	// Copy the file contents to the local file
	if _, err := io.Copy(localFile, stdout); err != nil {
		fmt.Printf("Failed to copy file contents: %v\n", err)
		return
	}

	// If copy is successful, delete the remote file
	session, err = connection.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session for deletion: %v\n", err)
		return
	}
	defer session.Close()

	if _, err := session.CombinedOutput(fmt.Sprintf("rm %s", remoteFilePath)); err != nil {
		fmt.Printf("Failed to delete %s on the remote server: %v\n", remoteFilePath, err)
	} else {
		fmt.Printf("Downloaded and deleted %s from the remote server. Local copy: %s\n", remoteFilePath, localFilePath)
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
		fmt.Println("No files modified in the last minute.")
		return
	}

	for _, filePath := range filePaths {
		if filePath != "" {
			downloadFile(connection, filePath, pair)
		}
	}
}
