package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func downloadFile(client *ssh.Client, remoteFilePath string, config DirectoryPair) {
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		fmt.Printf("Failed to create SFTP client: %v\n", err)
		return
	}
	defer sftpClient.Close()

	remoteFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		fmt.Printf("Failed to open remote file %s: %v\n", remoteFilePath, err)
		return
	}
	defer remoteFile.Close()

	localFilePath := computeLocalFilePath(remoteFilePath, config)
	localFile, err := createLocalFile(localFilePath)
	if err != nil {
		fmt.Printf("Failed to create local file '%s': %v\n", localFilePath, err)
		return
	}
	defer localFile.Close()

	var totalBytes int64
	startTime := time.Now()

	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := remoteFile.Read(buf)
		if n > 0 {
			n2, err2 := localFile.Write(buf[:n])
			atomic.AddInt64(&totalBytes, int64(n2))
			elapsed := time.Since(startTime).Seconds()
			rate := float64(totalBytes) / elapsed
			rateStr := formatTransferRate(rate)
			fmt.Printf("\rTransferring %s: %d bytes transferred at %s...", remoteFilePath, totalBytes, rateStr)
			if err2 != nil {
				fmt.Printf("\nFailed to write to local file: %v\n", err2)
				return
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("\nFailed to read from remote file: %v\n", err)
			return
		}
	}

	fmt.Printf("\nSuccessfully downloaded %s to %s\n", remoteFilePath, localFilePath)
}

func formatTransferRate(rate float64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
	)
	switch {
	case rate > MB:
		return fmt.Sprintf("%.2f MB/s", rate/MB)
	case rate > KB:
		return fmt.Sprintf("%.2f KB/s", rate/KB)
	default:
		return fmt.Sprintf("%.2f bytes/s", rate)
	}
}

func computeLocalFilePath(remoteFilePath string, config DirectoryPair) string {
	relativePath, err := filepath.Rel(config.RemoteDirectory, remoteFilePath)
	if err != nil {
		fmt.Printf("Error computing relative path for '%s', using base name: %v\n", remoteFilePath, err)
		return filepath.Join(config.LocalDirectory, filepath.Base(remoteFilePath))
	}
	return filepath.Join(config.LocalDirectory, relativePath)
}

func createLocalFile(localFilePath string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create local directories for '%s': %v", localFilePath, err)
	}
	return os.Create(localFilePath)
}
