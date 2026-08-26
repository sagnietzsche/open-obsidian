package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ObsidianTheme is a dark theme inspired by Obsidian.
type ObsidianTheme struct{}

func (o ObsidianTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantDark {
		switch name {
		case theme.ColorNameBackground:
			return color.RGBA{0x1e, 0x1e, 0x22, 0xff}
		case theme.ColorNameForeground:
			return color.RGBA{0xdc, 0xdc, 0xdc, 0xff}
		case theme.ColorNamePrimary:
			return color.RGBA{0x7c, 0x3a, 0xed, 0xff}
		case theme.ColorNameInputBackground:
			return color.RGBA{0x26, 0x26, 0x2a, 0xff}
		case theme.ColorNameButton:
			return color.RGBA{0x2d, 0x2d, 0x30, 0xff}
		}
	}
	return theme.DefaultTheme().Color(name, variant)
}
func (o ObsidianTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return theme.DefaultTheme().Icon(name) }
func (o ObsidianTheme) Font(style fyne.TextStyle) fyne.Resource   { return theme.DefaultTheme().Font(style) }
func (o ObsidianTheme) Size(name fyne.ThemeSizeName) float32       { return theme.DefaultTheme().Size(name) }
