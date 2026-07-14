package domain

type UploadCategory string

const (
	UploadCategoryTeam UploadCategory = "team"
	UploadCategoryNews UploadCategory = "news"
)

func (c UploadCategory) IsValid() bool {
	switch c {
	case UploadCategoryTeam,
		UploadCategoryNews:
		return true

	default:
		return false
	}
}