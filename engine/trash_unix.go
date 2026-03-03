//go:build !windows

package engine

import "github.com/laurent22/go-trash"

// MoveToTrash moves a file or directory to the OS trash/recycle bin.
func MoveToTrash(absPath string) error {
	_, err := trash.MoveToTrash(absPath)
	return err
}
