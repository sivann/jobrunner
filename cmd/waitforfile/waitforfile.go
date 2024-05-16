package main

import (
	"fmt"
	"os"
	"time"
)

func watchFolder(folderPath, targetFilename string, timeOut time.Duration) {
    startTime := time.Now()

	for {
		fileInfo, err := os.Stat(folderPath)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if fileInfo.IsDir() {
			files, err := os.ReadDir(folderPath)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			for _, file := range files {
				if file.Name() == targetFilename {
					fmt.Fprintf(os.Stderr,"Found %s at %s after %.2fsec.\n", targetFilename, folderPath, time.Since(startTime).Seconds())
                    os.Exit(0)
				}
			}
		}
		if time.Since(startTime) >= timeOut {
			fmt.Fprintln(os.Stderr, "Timeout reached. Exiting.")
			os.Exit(1) // Exit with non-zero status (timeout)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: ./watcher <timeOutSec> <folderPath> <targetFilename>")
		return
	}

	timeOutSec := os.Args[1]
	timeout, err := time.ParseDuration(timeOutSec + "s")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing timeout:", err)
		return
	}

	folderPath := os.Args[2] // Get the folder path from command-line arguments
	targetFilename := os.Args[3] // Get the target filename from command-line arguments
	watchFolder(folderPath, targetFilename, timeout)
}

