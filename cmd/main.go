package main

import (
	"flag"
	"fmt"
	"os"

	client "github.com/yaklang/aiclientv2"
)

func main() {
	host := flag.String("host", "127.0.0.1", "server host")
	port := flag.Int("port", 8089, "server port")
	prefix := flag.String("prefix", "/agent", "API route prefix")
	token := flag.String("token", "", "auth token (JWT)")
	flag.Parse()

	baseURL := fmt.Sprintf("http://%s:%d%s", *host, *port, *prefix)
	c := client.NewAgentClient(baseURL)
	if *token != "" {
		c.SetAuthToken(*token)
	}

	if err := client.RunTUI(c, baseURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
