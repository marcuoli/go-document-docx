// Command word-refresh refreshes existing TOCs using desktop Microsoft Word.
package main

import (
	"context"
	"fmt"
	"github.com/marcuoli/go-document-docx/publication"
	"github.com/marcuoli/go-document-docx/word"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: word-refresh <input.docx> <output.docx>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err == nil {
		source := word.WithUpdatedFields(publication.FromBytes(data), word.Options{})
		_, err = publication.Save(context.Background(), os.Args[2], false, source)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
