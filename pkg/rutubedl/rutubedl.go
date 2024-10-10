package rutubedl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/grafov/m3u8"
	"github.com/schollz/progressbar/v3"
)

// Constants for API URL templates and settings
const (
	DataURLTemplate  = "https://rutube.ru/api/play/options/%s/?no_404=true&referer=https%%253A%%252F%%252Frutube.ru&pver=v2"
	YappyURLTemplate = "https://rutube.ru/pangolin/api/web/yappy/yappypage/?client=wdp&source=shorts&videoId=%s"
	MaxRetries       = 5
	Timeout          = 120 * time.Second
	RetryDelay       = 2 * time.Second
	ForbiddenChars   = "/\\:*?\"<>|"
	OutputDir        = "./downloads"
)

// VideoType enum to represent different video types
//type VideoType string

/*const (
	Video  VideoType = "video"
	Shorts VideoType = "shorts"
	Yappy  VideoType = "yappy"
)*/

// VideoDownloader struct to hold video details and configuration
type VideoDownloader struct {
	VideoURL  string
	VideoData VideoAbstract
	OutputDir string
}

// NewVideoDownloader initializes a new VideoDownloader instance
func NewVideoDownloader(videoURL, outputDir string) (*VideoDownloader, error) {
	if outputDir == "" {
		outputDir = OutputDir
	}

	data, err := fetchVideoDetails(videoURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching video data: %v", err)
	}

	return &VideoDownloader{
		VideoURL:  videoURL,
		VideoData: data,
		OutputDir: outputDir,
	}, nil
}

// VideoAbstract interface defines the common methods for video types
type VideoAbstract interface {
	GetID() string
	GetTitle() string
	GetResolution() string
	GetVideoFileSegments() []string
	GetVideoFileSegmentsCount() int
	Write(stream io.Writer, workers int) error
}

type VideoData struct {
	Title         string        `json:"title"`
	VideoBalancer VideoBalancer `json:"video_balancer"`
}

type VideoBalancer struct {
	Default string `json:"default"`
	M3U8    string `json:"m3u8"`
}

// RutubeVideo struct represents a Rutube video
type RutubeVideo struct {
	ID              string
	VideoTitle      string
	BasePath        string
	VideoResolution string
	SegmentURLs     []string
}

func (rv *RutubeVideo) GetID() string {
	return rv.ID
}

func (rv *RutubeVideo) GetVideoFileSegments() []string {
	return rv.SegmentURLs
}

func (rv *RutubeVideo) GetVideoFileSegmentsCount() int {
	return len(rv.SegmentURLs)
}

// GetTitle returns the title of the Rutube video
func (rv *RutubeVideo) GetTitle() string {
	return rv.VideoTitle
}

// GetResolution returns the resolution of the Rutube video
func (rv *RutubeVideo) GetResolution() string {
	return rv.VideoResolution
}

// Write downloads video segments concurrently and writes them to the stream
func (rv *RutubeVideo) Write(stream io.Writer, workers int) error {
	var wg sync.WaitGroup
	segmentChan := make(chan string, len(rv.SegmentURLs))

	// Start workers to download segments
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for uri := range segmentChan {
				data, err := getSegmentData(uri)
				if err == nil {
					stream.Write(data)
				}
			}
		}()
	}

	// Feed segment URLs to channel
	for _, uri := range rv.SegmentURLs {
		segmentChan <- uri
	}
	close(segmentChan)
	wg.Wait()

	return nil
}

// YappyVideo struct represents a Yappy video
type YappyVideo struct {
	ID              string
	VideoLink       string
	VideoResolution string
}

// GetTitle returns the title of the Yappy video
func (yv *YappyVideo) GetTitle() string {
	return yv.ID
}

// GetResolution returns the resolution of the Yappy video
func (yv *YappyVideo) GetResolution() string {
	return "1920x1080"
}

// Write downloads the Yappy video content and writes it to the stream
func (yv *YappyVideo) Write(stream io.Writer, workers int) error {
	resp, err := http.Get(yv.VideoLink)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Yappy video: %v", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(stream, resp.Body)
	return err
}

// getSegmentData fetches the data for a given segment URL with retries
func getSegmentData(uri string) ([]byte, error) {
	var err error
	var resp *http.Response

	for retries := MaxRetries; retries > 0; retries-- {
		resp, err = http.Get(uri)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			return io.ReadAll(resp.Body)
		}
		time.Sleep(Timeout)
	}
	return nil, fmt.Errorf("failed to download segment after retries: %v", err)
}

// fetchPlaylistSegments fetches and parses the .m3u8 playlist to extract segment URLs
func fetchPlaylistSegments(playlistURL string) ([]string, string, error) {
	resp, err := http.Get(playlistURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("error fetching playlist: %v", err)
	}
	defer resp.Body.Close()

	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing playlist: %v", err)
	}

	var segmentURLs []string
	var resolution string

	switch listType {
	case m3u8.MEDIA:
		mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
		for _, segment := range mediaPlaylist.Segments {
			if segment != nil {
				segmentURLs = append(segmentURLs, segment.URI)
			}
		}
		resolution = "unknown"
		return segmentURLs, resolution, nil
	case m3u8.MASTER:
		masterPlaylist := playlist.(*m3u8.MasterPlaylist)
		for _, variant := range masterPlaylist.Variants {
			if variant != nil {
				if variant.Resolution == "1920x1080" {
					ok, mediaList, err := checkM3U8Availability(variant.URI)
					if !ok || err != nil {
						log.Println("segments URI not available, will try next one")
						continue
					}
					for _, entry := range mediaList.Segments {
						if entry != nil {
							link := buildMediaLinkForSegment(variant.URI, entry.URI)
							segmentURLs = append(segmentURLs, link)
						}
					}
					break
				}
			}
		}
		resolution = "1920x1080"
		return segmentURLs, resolution, nil
	default:
		return segmentURLs, resolution, errors.New("unknown play list type")
	}
}

// buildMediaLinkForSegment constructs the segment URL using the base URL and the segment postfix
func buildMediaLinkForSegment(originalLink, postfix string) string {
	baseURL := originalLink[:strings.LastIndex(originalLink, "/")]
	segmentURL := fmt.Sprintf("%s/%s", baseURL, postfix)
	return segmentURL
}

func checkM3U8Availability(url string) (bool, *m3u8.MediaPlaylist, error) {
	client := http.Client{
		Timeout: Timeout,
	}

	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("URL not available: %v", err)
	}
	defer resp.Body.Close()

	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return false, nil, fmt.Errorf("error parsing M3U8 playlist: %v", err)
	}

	if listType == m3u8.MEDIA {
		return true, playlist.(*m3u8.MediaPlaylist), nil
	}

	return false, nil, fmt.Errorf("not a MEDIA playlist")

}

// fetchVideoData fetches video metadata from the Rutube API
func fetchVideoData(videoID string) (*RutubeVideo, error) {
	url := fmt.Sprintf(DataURLTemplate, videoID)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching video data: %v", err)
	}
	defer resp.Body.Close()

	var data VideoData
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("error decoding video data: %v", err)
	}

	return &RutubeVideo{
		VideoTitle: data.Title,
		BasePath:   data.VideoBalancer.M3U8,
	}, nil
}

// fetchVideoDetails retrieves video details from a Rutube video link
func fetchVideoDetails(videoURL string) (VideoAbstract, error) {
	videoID, err := extractVideoID(videoURL)
	if err != nil {
		return nil, fmt.Errorf("error extracting video ID: %v", err)
	}

	// Fetch Rutube video data
	videoData, err := fetchVideoData(videoID)
	if err != nil {
		return nil, fmt.Errorf("error fetching Rutube video data: %v", err)
	}

	segmentURLs, resolution, err := fetchPlaylistSegments(videoData.BasePath)
	if err != nil {
		return nil, fmt.Errorf("error fetching playlist segments: %v", err)
	}

	videoData.ID = videoID
	videoData.SegmentURLs = segmentURLs
	videoData.VideoResolution = resolution

	return videoData, nil
}

// extractVideoID extracts the video ID from the given URL using a regular expression
func extractVideoID(videoURL string) (string, error) {
	re := regexp.MustCompile(`video/([\w\d]+)`)
	matches := re.FindStringSubmatch(videoURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("no video ID found in URL: %s", videoURL)
	}
	return matches[1], nil
}

// downloadSegment downloads a media segment and writes it to a temporary file with retry logic
func downloadSegment(url string, outputDir string, bar *progressbar.ProgressBar) (string, error) {
	var fileName string

	for attempts := 1; attempts <= MaxRetries; attempts++ {
		resp, err := http.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			fmt.Printf("Attempt %d: Error downloading segment: %v. Retrying...\n", attempts, err)
			time.Sleep(RetryDelay) // Wait before the next retry
			continue
		}
		defer resp.Body.Close()

		fileName = filepath.Join(outputDir, filepath.Base(url))
		file, err := os.Create(fileName)
		if err != nil {
			return "", fmt.Errorf("error creating file %s: %v", fileName, err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			fmt.Printf("Attempt %d: Error writing to file %s: %v. Retrying...\n", attempts, fileName, err)
			time.Sleep(RetryDelay) // Wait before the next retry
			continue
		}

		bar.Add(1)
		return fileName, nil
	}

	// If all retries fail, return an error
	return "", fmt.Errorf("failed to download segment %s after %d attempts", url, MaxRetries)
}

// mergeSegments merges all downloaded segments into a single output file
func mergeSegments(segmentFiles []string, outputFileName string) error {
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("error creating output file %s: %v", outputFileName, err)
	}
	defer outputFile.Close()

	for _, segmentFile := range segmentFiles {
		file, err := os.Open(segmentFile)
		if err != nil {
			return fmt.Errorf("error opening segment file %s: %v", segmentFile, err)
		}
		defer file.Close()

		_, err = io.Copy(outputFile, file)
		if err != nil {
			return fmt.Errorf("error writing segment file %s to output file: %v", segmentFile, err)
		}
	}

	log.Printf("All segments merged into: %s\n", outputFileName)
	return nil
}

func DownloadFile(fileLink string) error {
	video, err := fetchVideoDetails(fileLink)
	if err != nil {
		return fmt.Errorf("error fetching video details: %v", err)
	}
	outputFileName := fmt.Sprintf("%s.mp4", video.GetTitle())

	// Ensure the output directory exists
	tmpOutputDir := fmt.Sprintf("%s/%s", OutputDir, video.GetID())
	err = os.MkdirAll(tmpOutputDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	// Create a progress bar with the total number of segments
	numSegments := video.GetVideoFileSegmentsCount()
	bar := progressbar.NewOptions(numSegments,
		progressbar.OptionSetDescription("Downloading segments"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionClearOnFinish(),
	)

	// Download each segment to the output directory
	var segmentFiles []string
	for _, segmentURL := range video.GetVideoFileSegments() {
		segmentFile, err := downloadSegment(segmentURL, tmpOutputDir, bar)
		if err != nil {
			return fmt.Errorf("error downloading segment: %v", err)
		}
		segmentFiles = append(segmentFiles, segmentFile)
	}

	// Merge all downloaded segments into a single video file
	err = mergeSegments(segmentFiles, outputFileName)
	if err != nil {
		return fmt.Errorf("error merging segments: %v", err)
	}
	err = os.RemoveAll(tmpOutputDir)
	if err != nil {
		return fmt.Errorf("error removing folder: %v", err)
	}
	log.Printf("Video downloaded and saved to: %s\n", outputFileName)

	return nil
}
