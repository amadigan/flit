package main

import (
	"fmt"
	"os"
	"strings"

	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/x/term"
)

func main() {
	if len(os.Args) == 1 {
		renderREADME()
		return
	}

	if os.Args[1] == "go" {
		renderGo()
		return
	}

	fmt.Println("Usage: cli [go]")
}

func renderREADME() {
	f, err := os.ReadFile("./README.md")
	if err != nil {
		panic(err)
	}

	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		panic(err)
	}

	r := markdown.Render(string(f), min(80, w), 0)
	fmt.Println(string(r))
}

func renderGo() {
	f, err := os.ReadFile("./main.go")
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(f), "\n")

	var out []string
	for i := 26; i <= 39; i++ {
		out = append(out, lines[i])
	}

	err = quick.Highlight(os.Stdout, strings.Join(out, "\n")+"\n", "go", "terminal16m", "monokai")
	if err != nil {
		panic(err)
	}
}
