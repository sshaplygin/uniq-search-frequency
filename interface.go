//go:generate mockgen -source=main.go -destination=mocks.go -package=$GOPACKAGE

package main

type TextScanner interface {
	Text() string
	Scan() bool
}
