package utils

import "charm.land/lipgloss/v2"

func CenterLayer(width, height int, layer string) *lipgloss.Layer {
	layerWidth, layerHeight := lipgloss.Size(layer)
	return lipgloss.NewLayer(layer).X(width/2 - layerWidth/2).Y(height/2 - layerHeight/2)
}
