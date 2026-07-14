package dto

type UploadImageResponse struct {
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	URL          string `json:"url"`
	Path         string `json:"path"`
	ContentType  string `json:"content_type"`
	Size         int64  `json:"size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Category     string `json:"category"`
}