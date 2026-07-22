package main

import (
"log"

"dock/internal/server"
)

func main() {
s, err := server.New()
if err != nil {
log.Fatal(err)
}

log.Println("Dock a escutar em :8081")

if err := s.Listen(":8081"); err != nil {
log.Fatal(err)
}
}
