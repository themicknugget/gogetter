package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func dropPrivileges(uid, gid int) error {
	// Set GID first to avoid permission issues
	if err := syscall.Setgid(gid); err != nil {
		return fmt.Errorf("failed to set GID: %w", err)
	}
	if err := syscall.Setuid(uid); err != nil {
		return fmt.Errorf("failed to set UID: %w", err)
	}
	return nil
}

func main() {
	configPath := flag.String("cfg", "/config/config.yaml", "path to the configuration file")
	flag.Parse()

	config, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading configuration from %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	// Drop privileges based on the config or environment variables
	if err := dropPrivileges(config.UID, config.GID); err != nil {
		fmt.Printf("Error dropping privileges: %v\n", err)
		os.Exit(1)
	}

	shutdownCh := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		fmt.Println("\nReceived shutdown signal, cleaning up...")
		close(shutdownCh)
		os.Exit(0)
	}()

	for _, hostConfig := range config.Hosts {
		go MonitorHost(hostConfig, shutdownCh)
	}

	select {}
}
