package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/StanislavKH/rutube-dl/pkg/rutubedl"
)

func main() {
	listID := flag.String("list_id", "", "ID of the list to download")
	fileLink := flag.String("file_link", "", "Direct URL to the file to download")
	dir := flag.String("dir", "", "directory to store files and temporary chunks, default is - downloads")

	flag.Parse()

	if *listID != "" {
		log.Printf("Downloading list with ID: %s\n", *listID)
		list, err := rutubedl.GetItemsListFromFeedURI(*listID)
		if err != nil {
			log.Fatal(err)
			return
		}
		for _, file := range list {
			log.Printf("processing: %s\n", file.Title)
			err := rutubedl.DownloadFile(file.VideoURL, dir, 4)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	} else if *fileLink != "" {
		log.Printf("Downloading file from URL: %s\n", *fileLink)
		err := rutubedl.DownloadFile(*fileLink, dir, 5) // Using 5 workers as an example
		if err != nil {
			log.Fatalf("failed to download file: %v", err)
		}
	} else {
		fmt.Println("Please provide either a -list_id or a -file_link option")
	}
}
