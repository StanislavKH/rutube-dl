
# RutubeDL - Video Downloader

`RutubeDL` is a Go-based video downloader that allows you to download videos from Rutube using either a list of video IDs or a direct URL link. This tool can be used as both a compiled application and as an external package in your own Go projects.

## Features
- Supports downloading videos using a list of video IDs or direct video links.
- Allows specifying a download directory for storing files and temporary chunks.
- Uses concurrent downloads for faster performance.
- Implements retry logic to handle transient network issues.

## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
  - [As a Compiled Application](#as-a-compiled-application)
  - [As an External Package](#as-an-external-package)
- [Flags](#flags)
- [Example Commands](#example-commands)
- [Error Handling](#error-handling)
- [Contributing](#contributing)
- [License](#license)

## Installation

To use this project, you need to have [Go](https://golang.org/dl/) installed on your machine.

1. Clone the repository:
   ```bash
   git clone https://github.com/StanislavKH/rutube-dl.git
   cd rutube-dl
   ```

2. Build the application:
   ```bash
   go build -o rutubedl ./cmd/rutube-dl/main.go
   ```

This will create a binary named `rutubedl` (or `rutubedl.exe` on Windows) in your project directory.

## Usage

### As a Compiled Application

Once the binary is built, you can use it from the command line to download videos by list ID or direct link. You can also specify the directory where files and temporary chunks will be stored.

#### Command-line Options
- `-list_id` : Specify the list ID to download multiple videos.
- `-file_link` : Specify the direct URL to download a single video.
- `-dir` : Specify the directory to store downloaded files and temporary chunks (defaults to `downloads` if not specified).

### Example Commands

- Download videos using a list ID and store them in the `videos` directory:
  ```bash
  ./rutubedl -list_id=123456 -dir=videos
  ```

- Download a video using a direct URL link and store it in the default `downloads` directory:
  ```bash
  ./rutubedl -file_link=https://example.com/video-url
  ```

- Download a video using a direct URL link and specify a custom directory:
  ```bash
  ./rutubedl -file_link=https://example.com/video-url -dir=custom-directory
  ```

### As an External Package

You can also use this project as an external package in your own Go code. To do so, follow these steps:

1. Import the package into your project:
   ```go
   import "github.com/StanislavKH/rutube-dl/pkg/rutubedl"
   ```

2. Use the functions provided by the `rutubedl` package:
   ```go
   package main

   import (
       "log"
       "github.com/StanislavKH/rutube-dl/pkg/rutubedl"
   )

   func main() {
       fileLink := "https://example.com/video-url"
       dir := "videos"
       err := rutubedl.DownloadFile(fileLink, dir, 5) // Using 5 workers for concurrent download
       if err != nil {
           log.Fatalf("Failed to download file: %v", err)
       }
   }
   ```

3. Run your Go application as usual:
   ```bash
   go run yourapp.go
   ```

## Flags

| Flag            | Description                                                  | Example Usage                                    |
|-----------------|--------------------------------------------------------------|--------------------------------------------------|
| `-list_id`      | The ID of the list to download videos from                   | `./rutubedl -list_id=123456`                     |
| `-file_link`    | The direct URL of the file to download                       | `./rutubedl -file_link=https://...`              |
| `-dir`          | The directory to store files and temporary chunks (optional) | `./rutubedl -file_link=https://... -dir=videos`  |
| `-workers`      | The number of workers for chunk download (optional)          | `./rutubedl -file_link=https://... -workers=4`   |
| `-with_ffmpeg`  | Use external ffmpeg for more reliable chunk concatenation    | `./rutubedl -file_link=https://... -with_ffmpeg` |
## Error Handling
If an error occurs during the download process, the program will log the error and proceed to the next item in the list. For individual file downloads, an error will stop the process with an appropriate error message.

## Contributing
Feel free to submit issues or pull requests for improvements or bug fixes.

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
