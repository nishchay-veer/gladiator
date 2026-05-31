package termui

import "github.com/gdamore/tcell/v2"

type renderStyles struct {
	floorA   tcell.Style
	floorB   tcell.Style
	wall     tcell.Style
	wallEdge tcell.Style
	player1  tcell.Style
	player2  tcell.Style
	bullet   tcell.Style
}

type hudStyleSet struct {
	primary tcell.Style
	help    tcell.Style
}

func arenaStyles() renderStyles {
	return renderStyles{
		floorA:   tcell.StyleDefault.Foreground(tcell.NewRGBColor(10, 18, 28)).Background(tcell.NewRGBColor(9, 13, 22)),
		floorB:   tcell.StyleDefault.Foreground(tcell.NewRGBColor(13, 23, 36)).Background(tcell.NewRGBColor(9, 13, 22)),
		wall:     tcell.StyleDefault.Foreground(tcell.NewRGBColor(82, 116, 163)).Background(tcell.NewRGBColor(26, 48, 82)).Bold(true),
		wallEdge: tcell.StyleDefault.Foreground(tcell.NewRGBColor(38, 70, 111)).Background(tcell.NewRGBColor(15, 30, 54)),
		player1:  tcell.StyleDefault.Foreground(tcell.NewRGBColor(100, 255, 112)).Background(tcell.NewRGBColor(12, 48, 24)).Bold(true),
		player2:  tcell.StyleDefault.Foreground(tcell.NewRGBColor(255, 82, 82)).Background(tcell.NewRGBColor(54, 12, 22)).Bold(true),
		bullet:   tcell.StyleDefault.Foreground(tcell.NewRGBColor(83, 230, 255)).Background(tcell.NewRGBColor(9, 13, 22)).Bold(true),
	}
}

func hudStyles() hudStyleSet {
	return hudStyleSet{
		primary: tcell.StyleDefault.Foreground(tcell.NewRGBColor(185, 197, 214)).Background(tcell.NewRGBColor(3, 7, 12)),
		help:    tcell.StyleDefault.Foreground(tcell.NewRGBColor(104, 116, 132)).Background(tcell.NewRGBColor(3, 7, 12)),
	}
}
