package tabubbles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

var DefaultTheme = Theme{
	PrimaryColor:   lipgloss.Color("#FF4000"),
	SecondaryColor: lipgloss.Color("#11FF11"),
	AccentColor:    lipgloss.Color("#00B0FF"),
	TextColor:      nil,
	MutedColor:     lipgloss.Color("#404040"),
}

type Theme struct {
	PrimaryColor   color.Color
	SecondaryColor color.Color
	AccentColor    color.Color
	TextColor      color.Color
	MutedColor     color.Color

	UnfocusedPostColor   color.Color
	FocusedPostColor     color.Color
	PostTitleColor       color.Color
	PostDescriptionColor color.Color
}

func firstColor(colors ...color.Color) color.Color {
	for _, color := range colors {
		if color != nil {
			return color
		}
	}
	return nil
}

func (theme Theme) GetUnfocusedPostColor() color.Color {
	return firstColor(theme.UnfocusedPostColor, theme.MutedColor)
}

func (theme Theme) GetFocusedPostColor() color.Color {
	return firstColor(theme.FocusedPostColor, theme.PrimaryColor)
}

func (theme Theme) GetPostTitleColor() color.Color {
	return firstColor(theme.PostTitleColor, theme.PrimaryColor)
}

func (theme Theme) GetPostDescriptionColor() color.Color {
	return firstColor(theme.PostDescriptionColor, theme.TextColor)
}
