// XP integration smoke test for go-legacy-winxp Docker and CI runs.
package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("smoke: starting go-legacy-winxp guest checks")
	fmt.Printf("go-legacy-winxp smoke ok version=%s goos=%s goarch=%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	passed := 0
	runAllChecks(&passed)

	fmt.Printf("smoke: %d checks passed\n", passed)
	fmt.Println("smoke: all checks passed")
}
