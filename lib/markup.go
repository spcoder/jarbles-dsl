package lib

import (
	_ "embed"
	. "github.com/spcoder/rumble"
)

//go:embed markup.css
var css string

func CardDefault(extensionName, title, description, href string) string {
	return Fragment(
		Style(css),
		A(Href(href), Class("card"),
			Div(Class("card__header"),
				Div(Class("card__extension-name"), extensionName),
			),
			Div(Class("card__title"), title),
			Div(Class("card__description"), description),
			//A(Href(href), Class("card__button"), "Open"),
		),
	).Render()
}
