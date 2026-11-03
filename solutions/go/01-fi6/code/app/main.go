package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

// Usage: your_program.sh --origin <url> --detector <url> --policy <path>
func main() {
	flag.String("origin", "", "Origin chat completions API URL")
	flag.String("detector", "", "Classifier service URL")
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

	// Remove this line when you uncomment the code above.
	_ = http.ListenAndServe
}
