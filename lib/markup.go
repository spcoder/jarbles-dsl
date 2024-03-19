package lib

import (
	_ "embed"
	. "github.com/spcoder/rumble"
)

//go:embed markup.css
var css string

func CardDefault(title, description, href string) string {
	return Fragment(
		Style(css),
		Div(Class("card"),
			A(Href(href), Class("card__link"),
				Div(Class("card__title"), title),
				Div(Class("card__description"), description),
			),
		),
	).Render()
}
