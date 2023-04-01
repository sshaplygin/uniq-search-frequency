//go:generate mockgen -source=interface.go -destination=mocks.go -package=$GOPACKAGE

package main

type TextScanner interface {
	Text() string
	Scan() bool
}
