package main

import (
	"fmt"

	"github.com/StanislavKH/rutube-dl/pkg/rutubedl"
)

func main() {
	urls := []string{
		"https://rutube.ru/video/707bb0163d5eafb21be3c4c36ccb3212/", // 101
		"https://rutube.ru/video/72c9a7faba41d4ddb9b936de430bdf9b/", // 102
		"https://rutube.ru/video/7340cfc976fd32504496113ec04575b3/", // 103
		"https://rutube.ru/video/9185cf1ad9d95c578ff5f494e20778ea/", // 104
		"https://rutube.ru/video/6e47b3a5de6584bc5d5bb2cac7315838/", // 105
		"https://rutube.ru/video/31a041c7e1d2a2d6ca555a6f7bddae83/", // 106
		"https://rutube.ru/video/f44c9378fdce5d731d49cfd4cca0720a/", // 107
		"https://rutube.ru/video/0626231f358970ed1f931909b7d88e85/", // 108
		"https://rutube.ru/video/dc6ba1d4be7bc4be04e3f02cba202457/", // 109
	}

	for _, file := range urls {

		err := rutubedl.DownloadFile(file, 4)
		if err != nil {
			fmt.Println(err)
		}
	}
}
