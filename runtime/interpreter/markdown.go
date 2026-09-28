package interpreter

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Markdown renders text written by people as HTML. Raw HTML in the input
// is not passed through (goldmark's default) and dangerous link schemes
// such as javascript: are neutralised, so the output is safe to embed.
var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

func Markdown(src string) string {
	var b bytes.Buffer
	if err := md.Convert([]byte(src), &b); err != nil {
		return ""
	}
	return b.String()
}

func registerMarkdown(interp *Interpreter) {
	interp.RegisterModule("markdown", map[string]ModuleFunc{
		"html": func(c *Call, args []any) any { return Markdown(toString(c.Arg(args, 0, "texto"))) },
	})
}
