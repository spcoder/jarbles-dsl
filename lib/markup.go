package lib

import (
	_ "embed"
	. "github.com/spcoder/rumble"
)

//go:embed markup.css
var css string

type CardDefaultOptions struct {
	ExtensionName string
	Title         string
	Description   string
	Href          string
}

func CardDefault(options CardDefaultOptions) string {
	return Fragment(
		Style(css),
		A(Href(options.Href), Class("card"),
			Div(Class("card__header"),
				Div(Class("card__extension-name"), options.ExtensionName),
			),
			Div(Class("card__title"), options.Title),
			Div(Class("card__description"), options.Description),
		),
	).Render()
}
