//go:build windows

package engine

import (
	"fmt"
	"os/exec"
	"strings"
)

// MoveToTrash moves a file or directory to the Windows Recycle Bin
// using PowerShell's Shell.Application COM object.
func MoveToTrash(absPath string) error {
	// Use PowerShell to move to Recycle Bin via Shell.Application
	script := fmt.Sprintf(`
$shell = New-Object -ComObject Shell.Application
$item = $shell.Namespace(0).ParseName('%s')
if ($item) {
    $item.InvokeVerb('delete')
} else {
    exit 1
}
`, strings.ReplaceAll(absPath, "'", "''"))

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Run()
}
