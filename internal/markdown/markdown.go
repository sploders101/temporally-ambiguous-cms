package markdown

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/go-viper/mapstructure/v2"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
)

//go:embed ashenglass.xml
var ashenglassXML string
var ashenglass *chroma.Style

func init() {
	themebuf := bytes.NewBufferString(ashenglassXML)
	ashenglass = chroma.MustNewXMLStyle(themebuf)
}

type RenderOptions struct {
	EnableXHTML bool
	Highlight   HighlightOptions
}
type HighlightOptions struct {
	Theme        string
	InlineStyles bool
}

type ArticleFrontmatter struct {
	// The slug for the article. Shows up in the URL
	Slug string `mapstructure:"slug"`

	// The title of the article. Shows up in the title bar
	Title string `mapstructure:"title"`

	// Description for link previews & feeds
	Description string `mapstructure:"description"`

	// Assets to associate with the article
	Assets []string `mapstructure:"assets"`
}

func Render(markdown []byte, renderOptions RenderOptions) (ArticleFrontmatter, string, error) {
	var rendererOptions []renderer.Option
	if renderOptions.EnableXHTML {
		rendererOptions = append(rendererOptions, goldmarkhtml.WithXHTML())
	}

	highlightingOptions := []highlighting.Option{highlighting.WithGuessLanguage(false)}
	if renderOptions.Highlight.Theme == "" {
		highlightingOptions = append(
			highlightingOptions,
			highlighting.WithCustomStyle(ashenglass),
		)
	} else {
		highlightingOptions = append(
			highlightingOptions,
			highlighting.WithStyle(renderOptions.Highlight.Theme),
		)
	}
	if renderOptions.Highlight.InlineStyles {
		highlightingOptions = append(
			highlightingOptions,
			highlighting.WithFormatOptions(chromahtml.WithClasses(false)),
		)
	}

	processor := goldmark.New(
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(rendererOptions...),
		goldmark.WithExtensions(
			meta.Meta,
			extension.GFM,
			highlighting.NewHighlighting(highlightingOptions...),
			&anchor.Extender{},
		),
	)
	var buf bytes.Buffer
	context := parser.NewContext()
	if err := processor.Convert(markdown, &buf, parser.WithContext(context)); err != nil {
		panic(err)
	}

	var metadata ArticleFrontmatter
	if err := mapstructure.Decode(meta.Get(context), &metadata); err != nil {
		return ArticleFrontmatter{}, "", err
	}

	return metadata, buf.String(), nil
}

type ErrFrontmatterType struct {
	Varname      string
	ExpectedType string
}

func (err ErrFrontmatterType) Error() string {
	return fmt.Sprintf(
		"invalid frontmatter value for %q. expected %v",
		err.Varname,
		err.ExpectedType,
	)
}
