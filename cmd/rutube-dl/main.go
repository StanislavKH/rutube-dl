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
	workers := flag.Int("workers", 1, "number of workers for chunk download; be careful when using more than 5")
	withFfmpeg := flag.Bool("with_ffmpeg", false, "use external ffmpeg for more reliable chunk concatenation")

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
			err := rutubedl.DownloadFile(file.VideoURL, dir, *workers, *withFfmpeg)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	} else if *fileLink != "" {
		log.Printf("Downloading file from URL: %s\n", *fileLink)
		err := rutubedl.DownloadFile(*fileLink, dir, *workers, *withFfmpeg)
		if err != nil {
			log.Fatalf("failed to download file: %v", err)
		}
	} else {
		fmt.Println("Usage: Provide either a -list_id or a -file_link option")
		fmt.Println("Example:")
		fmt.Println("  - To download a list: go run main.go -list_id=<list-id> [-dir=<directory>] [-workers=<number>] [-with_ffmpeg]")
		fmt.Println("  - To download a file: go run main.go -file_link=<file-url> [-dir=<directory>] [-workers=<number>] [-with_ffmpeg]")
		fmt.Println("\nNote: The -dir option is optional. If not specified, it defaults to 'downloads'.")
		fmt.Println("The -workers option is optional and defaults to 1. Be cautious when using more than 5.")
		fmt.Println("The -with_ffmpeg option is optional. If enabled, external ffmpeg is used for more reliable chunk concatenation.")
	}
}
