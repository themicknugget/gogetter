package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// downloadFile downloads a file from the remote server to the local directory. It prints out start messages,
// checks file size after transfer, and deletes the remote file if the transfer was successful.
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

	// Get remote file size for later comparison
	remoteFileInfo, err := remoteFile.Stat()
	if err != nil {
		fmt.Printf("Failed to stat remote file %s: %v\n", remoteFilePath, err)
		return
	}
	remoteFileSize := remoteFileInfo.Size()

	localFilePath := computeLocalFilePath(remoteFilePath, config)

	fmt.Printf("Starting download: %s to %s\n", remoteFilePath, localFilePath)

	localFile, err := createLocalFile(localFilePath)
	if err != nil {
		fmt.Printf("Failed to create local file '%s': %v\n", localFilePath, err)
		return
	}
	defer localFile.Close()

	bytesCopied, err := remoteFile.WriteTo(localFile)
	if err != nil {
		fmt.Printf("Failed to copy file from remote to local: %v\n", err)
		return
	}

	// Check if the entire file was copied
	if bytesCopied != remoteFileSize {
		fmt.Printf("File size mismatch: copied %d bytes; expected %d bytes\n", bytesCopied, remoteFileSize)
		return
	}

	// Delete the remote file if the size matches
	if err := sftpClient.Remove(remoteFilePath); err != nil {
		fmt.Printf("Failed to delete remote file %s after successful download: %v\n", remoteFilePath, err)
		return
	}

	fmt.Printf("Successfully downloaded and deleted %s (%d bytes copied)\n", remoteFilePath, bytesCopied)
}

func computeLocalFilePath(remoteFilePath string, config DirectoryPair) string {
	relativePath, err := filepath.Rel(config.RemoteDirectory, remoteFilePath)
	if err != nil {
		fmt.Printf("Error computing relative path for '%s': %v; using base name.\n", remoteFilePath, err)
		relativePath = filepath.Base(remoteFilePath)
	}
	return filepath.Join(config.LocalDirectory, relativePath)
}

func createLocalFile(localFilePath string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create local directories for '%s': %v", localFilePath, err)
	}
	return os.Create(localFilePath)
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
