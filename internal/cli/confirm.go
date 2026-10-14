package cli

import (
	"bufio"
	"fmt"
	"io"
)

func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprint(out, prompt+" [y/N] ")

	answer, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		return false
	}

	if answer == "y\n" || answer == "Y\n" {
		return true
	}

	return false
}
