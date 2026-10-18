package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	input := bufio.NewScanner(os.Stdin)
	for input.Scan() {
		if input.Text() == "expire" {
			fmt.Println("二维码已过期 / QrCode State: CodeExpired")
		}
	}
}
