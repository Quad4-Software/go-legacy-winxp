package main

import "crypto/rand"

func randReadPackage(buf []byte) (int, error) {
	return rand.Read(buf)
}
