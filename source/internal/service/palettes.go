package service

import "slices"

const DefaultPalette = "matchmark"

var Palettes = []string{
	"matchmark",
	"arctic-blue",
	"nordic-forest",
	"royal-competition",
	"copper-graphite",
	"graphite-lime",
}

func ValidPalette(id string) bool {
	return slices.Contains(Palettes, id)
}
