package update

type Status string

const (
	StatusChecking Status = "checking"
	StatusUpdate   Status = "update"
	StatusCurrent  Status = "current"
	StatusError    Status = "error"
)

type Result struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`

	Image string `json:"-"`

	Current string `json:"current"`
	Status  Status `json:"status"`
}