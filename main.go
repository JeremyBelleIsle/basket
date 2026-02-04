package main

import (
	"image/color"
	"log"

	"github.com/JeremyBelleIsle/gameutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type player struct {
	x, y float64
	w, h float64
	VelX float64
	VelY float64
	clr  color.RGBA
}

type basket struct {
	x, y float64
	w, h float64
	clr  color.RGBA
}

type Game struct {
	platforms []gameutil.Platform
	player    player
	basket    basket
	Level     int
}

func Mouvement(px, py, pw, ph float64, VelX float64, VelY float64, platforms []gameutil.Platform) (float64, float64, float64) {
	newX := px + VelX
	newY := py + VelY

	for _, p := range platforms {
		if gameutil.RectColl(newX, newY, pw, ph, p.X, p.Y, p.W, p.H) {
			return px, py, VelY
		}
	}

	py += VelY
	VelY += 0.03

	return px, py, VelY
}

func loadLevel(L int, platforms *[]gameutil.Platform) {
	switch L {
	case 0:
		*platforms = []gameutil.Platform{
			{X: 0, Y: screenHeight - 200, W: 640, H: 100, Clr: color.RGBA{128, 0, 128, 255}},
		}
	}
}

func PlatformCollisions(playerX, playerY, playerW, playerH float64, playerVelX, playerVelY *float64, platforms []gameutil.Platform) (float64, float64) {

	newX := playerX + *playerVelX
	newY := playerY + *playerVelY

	// ===== X AXIS =====
	for _, p := range platforms {
		if gameutil.RectColl(newX, playerY, playerW, playerH, p.X, p.Y, p.W, p.H) {
			if *playerVelX > 0 {
				newX = p.X - playerW
			} else if *playerVelX < 0 {
				newX = p.X + p.W
			}
			*playerVelX = -*playerVelX * 0.7
		}
	}

	// ===== Y AXIS =====
	for _, p := range platforms {
		if gameutil.RectColl(newX, newY, playerW, playerH, p.X, p.Y, p.W, p.H) {
			if *playerVelY > 0 {
				newY = p.Y - playerH
			} else if *playerVelY < 0 {
				newY = p.Y + p.H
			}
			*playerVelY = -*playerVelY * 0.3
		}
	}

	return newX, newY
}

func (g *Game) Update() error {
	g.player.x, g.player.y, g.player.VelY = Mouvement(g.player.x, g.player.y, g.player.w, g.player.h, g.player.VelX, g.player.VelY, g.platforms)

	loadLevel(g.Level, &g.platforms)

	PlatformCollisions(g.player.x, g.player.y, g.player.w, g.player.h, &g.player.VelX, &g.player.VelY, g.platforms)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Basket
	vector.StrokeRect(screen, float32(g.basket.x), float32(g.basket.y), float32(g.basket.w), float32(g.basket.h), 5, g.basket.clr, true)

	// Player
	vector.DrawFilledRect(screen, float32(g.player.x), float32(g.player.y), float32(g.player.w), float32(g.player.h), g.player.clr, true)

	// Platforms
	for _, plat := range g.platforms {
		vector.DrawFilledRect(screen, float32(plat.X), float32(plat.Y), float32(plat.W), float32(plat.H), plat.Clr, true)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Hello, World!")

	g := &Game{
		basket: basket{
			x:   screenWidth - 200,
			y:   screenHeight/2 - 37.5,
			w:   50,
			h:   75,
			clr: color.RGBA{0, 0, 255, 255},
		},
		player: player{
			x:    50,
			y:    screenHeight/2 - 50,
			VelX: 0,
			VelY: 3,
			w:    35,
			h:    35,
			clr:  color.RGBA{255, 0, 0, 255},
		},
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
