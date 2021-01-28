package cmdctrl

import (
	"io"
)

// In is used to send commands to terminal on windows
var In io.WriteCloser
