package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

// Ensures gofmt doesn't remove the "net/http" import in stage 1 (feel free to remove this!)
var _ = http.ListenAndServe

// Usage: your_program.sh --origin <url> --detector <url> --policy <path>
func main() {
	flag.String("origin", "", "Origin web application URL")
	flag.String("detector", "", "ML detector URL")
	flag.String("policy", "", "Policy JSON file path")
	flag.Parse()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
