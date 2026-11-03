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
	flag.String("origin", "", "Origin chat completions API URL")
	flag.String("detector", "", "Classifier service URL")
	flag.String("policy", "", "Policy JSON file path")
	flag.Parse()

	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintf(os.Stderr, "Logs from your program will appear here!\n")

	// TODO: Uncomment the code below to pass the first stage!
	//
	// http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	// 	if r.Method != http.MethodGet {
	// 		w.WriteHeader(http.StatusMethodNotAllowed)
	// 		return
	// 	}
	// 	w.WriteHeader(http.StatusOK)
	// })
	// if err := http.ListenAndServe(":8080", nil); err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	os.Exit(1)
	// }
}
