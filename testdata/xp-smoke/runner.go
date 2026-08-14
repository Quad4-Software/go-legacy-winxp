package main

import (
	"fmt"
	"os"
)

func check(name string, fn func() error) {
	fmt.Printf("check %s: ", name)
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC %v\n", r)
			os.Exit(2)
		}
	}()
	if err := fn(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func section(name string) {
	fmt.Printf("smoke: section %s\n", name)
}

func runCheck(name string, fn func() error, passed *int) {
	check(name, fn)
	*passed++
}
