package main

import (
	"flag"
	"log"

	"github.com/titan-x/titan"
)

var ext = flag.Bool("ext", false, "Run external client test case.")

func main() {
	addr := "127.0.0.1:3000"
	if *ext {
		addr = "127.0.0.1:3001"
		log.Printf("-ext flag is provided, starting external client test case.")
	}

	s, err := titan.NewServer(addr)
	if err != nil {
		log.Fatalf("error creating server: %v", err)
	}
	defer s.Close()

	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("error closing server: %v", err)
	}
}