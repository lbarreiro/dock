package update

type Status string

const (
	StatusChecking    Status = "checking"
	StatusUpdate      Status = "update"
	StatusCurrent     Status = "current"
	StatusError       Status = "error"
	StatusUnsupported Status = "unsupported"
)

type Result struct {
	Name      string `json:"name"`
	CanUpdate bool   `json:"can_update"`
	Message   string `json:"message,omitempty"`
	Provider  string `json:"provider"`

	Image string `json:"-"`

	Current string `json:"current"`
	Status  Status `json:"status"`
}
