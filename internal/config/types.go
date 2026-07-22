package config

type Config struct {
	Server  Server   `yaml:"server"`
	Actions []Action `yaml:"actions"`
}

type Server struct {
	Port  int    `yaml:"port"`
	Title string `yaml:"title"`
}

type Action struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Icon   string `yaml:"icon"`
	Script string `yaml:"script"`
}
