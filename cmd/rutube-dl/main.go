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
	fromEpisode := flag.Int("from_episode", 1, "download everything starting from this episode number (only with -list_id)")

	flag.Parse()

	if *listID != "" {
		log.Printf("Downloading list with ID: %s\n\r", *listID)
		list, err := rutubedl.GetItemsListFromFeedURI(*listID)
		if err != nil {
			log.Fatal(err)
			return
		}
		for _, file := range list {
			log.Printf("processing: %s\n\r", file.Title)
			if file.Episode < *fromEpisode {
				log.Printf("episode %s skipped", file.Title)
				continue
			}
			err := rutubedl.DownloadFile(file.VideoURL, dir, *workers, *withFfmpeg)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	} else if *fileLink != "" {
		log.Printf("Downloading file from URL: %s\n\r", *fileLink)
		err := rutubedl.DownloadFile(*fileLink, dir, *workers, *withFfmpeg)
		if err != nil {
			log.Fatalf("failed to download file: %v", err)
		}
	} else {
		fmt.Println("Usage: Provide either a -list_id or a -file_link option")
		fmt.Println("Example:")
		fmt.Println("  - To download a list: rutube-dl -list_id=<list-id> [-dir=<directory>] [-workers=<number>] [-with_ffmpeg] [-from_episode=<episode-number>]")
		fmt.Println("  - To download a file: rutube-dl -file_link=<file-url> [-dir=<directory>] [-workers=<number>] [-with_ffmpeg]")
		fmt.Println("\nNote:")
		fmt.Println("  - The -dir option is optional. If not specified, it defaults to 'downloads'.")
		fmt.Println("  - The -workers option is optional and defaults to 1. Be cautious when using more than 5.")
		fmt.Println("  - The -with_ffmpeg option is optional. If enabled, external ffmpeg is used for more reliable chunk concatenation.")
		fmt.Println("  - The -from_episode option is optional and can only be used with -list_id. It specifies the episode number from which to start downloading.")
	}
}
