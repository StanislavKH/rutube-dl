package main

import (
	"fmt"

	"github.com/StanislavKH/rutube-dl/pkg/rutubedl"
)

func main() {
	urls := []string{
		"https://rutube.ru/video/8c27fc8eee3bbece059b6b7e71847b37/", // 110
		"https://rutube.ru/video/96cb1ed1a1d1542b79e380bdf70f8abb/", // 111
		"https://rutube.ru/video/1c20721c30a63f6a692a08198f3cf0eb/", // 112
		"https://rutube.ru/video/1d9ae4f291967abf68f0819a0f59a303/", // 113
		"https://rutube.ru/video/a73bf5078c5e9be31ae35f1bd7618e09/", // 114
		"https://rutube.ru/video/20e28616723bae668dee5a3b1654f0b1/", // 115
		"https://rutube.ru/video/2a2a73d84d6f23b0092c50f9e7c061ae/", // 116
		"https://rutube.ru/video/fe5a407e8511133a7d525ec7298dc6fd/", // 117
		"https://rutube.ru/video/3d9ca743ee18a11ffe6a0d7ee0cf21e8/", // 118
		"https://rutube.ru/video/0e59b63c3b815cbf406fb728a006bc0a/", // 119
		"https://rutube.ru/video/04b5a735cb673a0529e2b459a8f342ec/", // 120
		"https://rutube.ru/video/76047edaee63a9a7135e3c2701f80baf/", // 121
		"https://rutube.ru/video/03b0e878e2ac1fc384a440d764f4241d/", // 122
	}

	for _, file := range urls {

		err := rutubedl.DownloadFile(file, 4)
		if err != nil {
			fmt.Println(err)
		}
	}
}
