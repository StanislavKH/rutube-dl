package rutubedl

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const playlistFeedURI = "https://rutube.ru/api/metainfo/tv/%s/video/?type\x3d2\x26show_all\x3d1\x26sort\x3dseries_d\x26show_hidden_videos\x3dFalse\x26fields\x3dextra_params,future_publication"

type PlaylistFeed struct {
	HasNext  bool        `json:"has_next"`
	Next     interface{} `json:"next"`
	Previous interface{} `json:"previous"`
	Page     int         `json:"page"`
	PerPage  int         `json:"per_page"`
	Results  []Result    `json:"results"`
}

type Result struct {
	ID                   string       `json:"id"`
	ThumbnailURL         string       `json:"thumbnail_url"`
	VideoURL             string       `json:"video_url"`
	Duration             int          `json:"duration"`
	PictureURL           string       `json:"picture_url"`
	Author               Author       `json:"author"`
	PgRating             PgRating     `json:"pg_rating"`
	OriginType           string       `json:"origin_type"`
	PreviewURL           string       `json:"preview_url"`
	IsClub               bool         `json:"is_club"`
	IsClassic            bool         `json:"is_classic"`
	IsPaid               bool         `json:"is_paid"`
	ProductID            interface{}  `json:"product_id"`
	StreamType           interface{}  `json:"stream_type"`
	TrackID              int          `json:"track_id"`
	Title                string       `json:"title"`
	CreatedTs            string       `json:"created_ts"`
	IsAudio              bool         `json:"is_audio"`
	Hits                 int          `json:"hits"`
	PublicationTs        string       `json:"publication_ts"`
	Description          string       `json:"description"`
	IsLivestream         bool         `json:"is_livestream"`
	IsAdult              bool         `json:"is_adult"`
	LastUpdateTs         string       `json:"last_update_ts"`
	FeedURL              string       `json:"feed_url"`
	ActionReason         ActionReason `json:"action_reason"`
	IsDeleted            bool         `json:"is_deleted"`
	IsOriginalContent    bool         `json:"is_original_content"`
	IsOriginalSticker2X2 bool         `json:"is_original_sticker_2x2"`
	IsRebornChannel      bool         `json:"is_reborn_channel"`
	KindSignForUser      bool         `json:"kind_sign_for_user"`
	FeedName             string       `json:"feed_name"`
	Season               int          `json:"season"`
	Episode              int          `json:"episode"`
	Fts                  int          `json:"fts"`
	Category             Category     `json:"category"`
	Type                 Type         `json:"type"`
}

type Type struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

type Category struct {
	ID          int    `json:"id"`
	CategoryURL string `json:"category_url"`
	Name        string `json:"name"`
}

type ActionReason struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Author struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	AvatarURL        string `json:"avatar_url"`
	SiteURL          string `json:"site_url"`
	IsAllowedOffline bool   `json:"is_allowed_offline"`
}

type PgRating struct {
	Age  int    `json:"age"`
	Logo string `json:"logo"`
}

type ProcessedOut struct {
	Title    string
	VideoURL string
	FeedName string
}

func GetItemsListFromFeedURI(feedID string) ([]ProcessedOut, error) {
	url := fmt.Sprintf(playlistFeedURI, feedID)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching feed data: %v", err)
	}
	defer resp.Body.Close()

	var data PlaylistFeed
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("error decoding feed data: %v", err)
	}

	var result []ProcessedOut
	for _, item := range data.Results {
		outRow := ProcessedOut{
			Title:    item.Title,
			VideoURL: item.VideoURL,
			FeedName: item.FeedName,
		}
		result = append(result, outRow)
	}

	return result, nil
}
