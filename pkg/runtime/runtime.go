package runtime

import (
	"fmt"

	"github.com/adrg/xdg"
)

const (
	XDGName = "jot"
)

func File(filename string) (string, error) {
	return xdg.RuntimeFile(fmt.Sprintf("%s/%s", XDGName, filename))
}
