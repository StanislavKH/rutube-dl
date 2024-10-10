package main

import (
	"fmt"

	"github.com/StanislavKH/rutube-dl/pkg/rutubedl"
)

func main() {
	urls := []string{
		"https://rutube.ru/video/d9c26d9e604e47278cf0e3a415317b66/",
		"https://rutube.ru/video/6d755680604d87b8aa6b26b62f894e61/",
		"https://rutube.ru/video/2928a257bd23fef3ca8c6374759f667b/",
	}

	for _, file := range urls {

		err := rutubedl.DownloadFile(file)
		if err != nil {
			fmt.Println(err)
		}
	}
}
