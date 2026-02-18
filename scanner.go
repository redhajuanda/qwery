package qwery

import "io"

type Scannerer interface {
	ScanStruct(dest any) error
	ScanMap(dest map[string]any) error
	ScanStructs(dest any) error
	ScanMaps(dest *[]map[string]any) error
	ScanWriter(dest io.Writer) error
	Close() error
}

// Scanner is a struct that contains the scanner
type Scanner struct {
	scannerType int
	dest        any
}

const (
	noScanner = iota + 1
	scannerMap
	scannerMaps
	scannerStruct
	scannerStructs
	scannerWriter
)

// newScanner returns a new scanner
func newScanner(scannerType int, dest any) *Scanner {

	return &Scanner{
		scannerType: scannerType,
		dest:        dest,
	}

}
