package models

// Container representa um contentor Docker.
//
// Esta estrutura pertence ao Dock e nunca depende
// da SDK oficial do Docker.
type Container struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Status string `json:"status"`
	URL    string `json:"url"`
}
