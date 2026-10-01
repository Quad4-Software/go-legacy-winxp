// Small XP guest demo used in README screenshots.
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("go-legacy-winxp hello")
	fmt.Printf("go=%s goos=%s goarch=%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
