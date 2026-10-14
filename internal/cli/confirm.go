package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt+" [y/N] ")

	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	}

	return false
}
