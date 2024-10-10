package rutubedl

import (
	"context"
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
	MaxRetries       = 3
	Timeout          = 60 * time.Second
	RetryDelay       = 2 * time.Second
	ForbiddenChars   = "/\\:*?\"<>|"
	OutputDir        = "./downloads"
)

// VideoAbstract interface defines the common methods for video types
type VideoAbstract interface {
	GetID() string
	GetTitle() string
	GetResolution() string
	GetVideoFileSegments() []string
	GetVideoFileSegmentsCount() int
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

// getWithRetry performs an HTTP GET request with retry logic.
// It retries the request up to MaxRetries times with a delay between attempts.
func getWithRetry(url string, timeout time.Duration, maxRetries int, retryDelay time.Duration) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempts := 1; attempts <= maxRetries; attempts++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, reqErr := http.NewRequestWithContext(ctx, "GET", url, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("error creating request: %v", reqErr)
		}

		client := &http.Client{}
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		if attempts < maxRetries {
			log.Printf("Attempt %d/%d: Failed to fetch URL, retrying in %v...\n", attempts, maxRetries, retryDelay)
			time.Sleep(retryDelay)
		} else {
			return nil, fmt.Errorf("failed to fetch URL after %d attempts: %v", maxRetries, err)
		}
	}

	return nil, err
}

// fetchPlaylistSegments fetches and parses the .m3u8 playlist to extract segment URLs
func fetchPlaylistSegments(playlistURL string) ([]string, string, error) {
	resp, err := getWithRetry(playlistURL, Timeout, MaxRetries, RetryDelay)
	if err != nil {
		return nil, "", fmt.Errorf("error fetching playlist: %v", err)
	}
	defer resp.Body.Close()

	playlist, listType, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing playlist: %v", err)
	}

	var segmentURLs []string
	var resolution string = "unknown"

	switch listType {
	case m3u8.MEDIA:
		mediaPlaylist := playlist.(*m3u8.MediaPlaylist)
		for _, segment := range mediaPlaylist.Segments {
			if segment != nil {
				segmentURLs = append(segmentURLs, segment.URI)
			}
		}
		return segmentURLs, resolution, nil
	case m3u8.MASTER:
		masterPlaylist := playlist.(*m3u8.MasterPlaylist)
		for _, variant := range masterPlaylist.Variants {
			if variant != nil {
				if strings.HasPrefix(variant.Resolution, "1920x") {
					resolution = variant.Resolution
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
	resp, err := getWithRetry(url, Timeout, MaxRetries, RetryDelay)
	if err != nil {
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

// DownloadFile downloads the video file using multiple concurrent workers.
func DownloadFile(fileLink string, numWorkers int) error {
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

	// Create channels for segment URLs and results
	segmentChan := make(chan string, numSegments)
	errChan := make(chan error, numWorkers)
	var segmentFiles []string
	var segmentFilesMutex sync.Mutex

	// Launch workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for segmentURL := range segmentChan {
				segmentFile, err := downloadSegmentWithContext(segmentURL, tmpOutputDir, bar)
				if err != nil {
					errChan <- fmt.Errorf("error downloading segment: %v", err)
					return
				}
				segmentFilesMutex.Lock()
				segmentFiles = append(segmentFiles, segmentFile)
				segmentFilesMutex.Unlock()
			}
		}()
	}

	for _, segmentURL := range video.GetVideoFileSegments() {
		segmentChan <- segmentURL
	}
	close(segmentChan)

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	err = mergeSegments(segmentFiles, outputFileName)
	if err != nil {
		return fmt.Errorf("error merging segments: %v", err)
	}

	// Cleanup temporary output directory
	err = os.RemoveAll(tmpOutputDir)
	if err != nil {
		return fmt.Errorf("error removing temporary directory: %v", err)
	}

	log.Printf("Video downloaded and saved to: %s\n", outputFileName)
	return nil
}

// downloadSegmentWithContext downloads a media segment using a context for timeout control.
func downloadSegmentWithContext(url, outputDir string, bar *progressbar.ProgressBar) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	var fileName string
	for attempts := 1; attempts <= MaxRetries; attempts++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("error creating request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			log.Printf("Attempt %d: Failed to download segment, retrying...\n", attempts)
			time.Sleep(RetryDelay)
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
			log.Printf("Attempt %d: Error writing to file %s: %v. Retrying...\n", attempts, fileName, err)
			time.Sleep(RetryDelay)
			continue
		}

		bar.Add(1)
		return fileName, nil
	}

	return "", fmt.Errorf("failed to download segment %s after %d attempts", url, MaxRetries)
}
