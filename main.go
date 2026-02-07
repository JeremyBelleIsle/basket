package main

import (
	"image/color"
	"log"

	"github.com/JeremyBelleIsle/gameutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

type player struct {
	x, y            float64
	r               float64
	propulsionPower float64
	VelX            float64
	VelY            float64
	clr             color.RGBA
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

func Mouvement(px, py, pr float64, VelX, VelY float64, platforms []gameutil.Platform) (float64, float64, float64, float64) {

	VelX *= 0.97 // friction
	VelY += 0.25 // gravité

	newX := px + VelX
	newY := py + VelY

	// collisions X
	for _, p := range platforms {
		if gameutil.CircleRectCollision(newX, py, pr, p.X, p.Y, p.W, p.H) {
			VelX = -VelX * 0.7 // rebond
			newX = px
		}
	}

	// collisions Y
	for _, p := range platforms {
		if gameutil.CircleRectCollision(newX, newY, pr, p.X, p.Y, p.W, p.H) {
			VelY = -VelY * 0.6
			newY = py
		}
	}

	if py < screenHeight-48 {
		return newX, newY, VelX, VelY
	}
	speed := 2.0
	canMoveLeft := true
	canMoveRight := true
	for _, p := range platforms {
		if gameutil.CircleRectCollision(px-speed, py-10, pr, p.X, p.Y, p.W, p.H) {
			canMoveLeft = false
		}

		if gameutil.CircleRectCollision(px+speed, py-10, pr, p.X, p.Y, p.W, p.H) {
			canMoveRight = false
		}
	}

	if canMoveRight {
		if ebiten.IsKeyPressed(ebiten.KeyRight) {
			newX += speed

			py = screenHeight - 100
		}
	}

	if canMoveLeft {
		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			newX -= speed

			py = screenHeight - 100
		}
	}

	return newX, newY, VelX, VelY
}

func loadLevel(L int, platforms *[]gameutil.Platform, b *basket) {
	switch L {

	case 0:
		*platforms = []gameutil.Platform{

			// ===== Bordures arène =====

			// sol
			{X: 0, Y: screenHeight - 30, W: screenWidth, H: 30, Clr: color.RGBA{128, 0, 128, 255}},

			// plafond
			{X: 0, Y: 0, W: screenWidth, H: 20, Clr: color.RGBA{128, 0, 128, 255}},

			// mur gauche
			{X: 0, Y: 0, W: 20, H: screenHeight, Clr: color.RGBA{128, 0, 128, 255}},

			// mur droit
			{X: screenWidth - 20, Y: 0, W: 20, H: screenHeight, Clr: color.RGBA{128, 0, 128, 255}},

			// ===== Obstacles rebond =====

			{X: 180, Y: 330, W: 120, H: 20, Clr: color.RGBA{150, 0, 150, 255}},
			{X: 350, Y: 250, W: 120, H: 20, Clr: color.RGBA{150, 0, 150, 255}},

			// ===== contour du basket =====
			{X: b.x, Y: b.y + b.h - 5, W: b.w, H: 5, Clr: color.RGBA{0, 0, 0, 0}},
			{X: b.x + b.w - 5, Y: b.y, W: 5, H: b.h, Clr: color.RGBA{0, 0, 0, 0}},
			{X: b.x, Y: b.y, W: 5, H: b.h, Clr: color.RGBA{0, 0, 0, 0}},
		}

		// 🏀 panier en hauteur (coin droit)
		b.w = 50
		b.h = 75
		b.x = screenWidth - 90
		b.y = 180
	}
}

// func PlatformCollisions(playerX, playerY, playerW, playerH float64, playerVelX, playerVelY *float64, platforms []gameutil.Platform) (float64, float64) {

// 	newX := playerX + *playerVelX
// 	newY := playerY + *playerVelY

// 	// ===== X AXIS =====
// 	for _, p := range platforms {
// 		if gameutil.RectColl(newX, playerY, playerW, playerH, p.X, p.Y, p.W, p.H) {
// 			if *playerVelX > 0 {
// 				newX = p.X - playerW
// 			} else if *playerVelX < 0 {
// 				newX = p.X + p.W
// 			}
// 			*playerVelX = -*playerVelX * 0.7
// 		}
// 	}

// 	// ===== Y AXIS =====
// 	for _, p := range platforms {
// 		if gameutil.RectColl(newX, newY, playerW, playerH, p.X, p.Y, p.W, p.H) {
// 			if *playerVelY > 0 {
// 				newY = p.Y - playerH
// 			} else if *playerVelY < 0 {
// 				newY = p.Y + p.H
// 			}
// 			*playerVelY = -*playerVelY * 0.3
// 		}
// 	}

// 	return newX, newY
// }

func detectJump(VelX, VelY float64, px, py, pr float64, platforms []gameutil.Platform, propulsionPower float64) (float64, float64, float64) {

	for _, p := range platforms {

		if gameutil.CircleRectCollision(px, py+2, pr, p.X, p.Y, p.W, p.H) {

			if inpututil.IsKeyJustReleased(ebiten.KeySpace) {
				VelX = propulsionPower / 10
				VelY = -propulsionPower / 5
				propulsionPower = 0
			}

			if ebiten.IsKeyPressed(ebiten.KeySpace) {
				propulsionPower++
				if propulsionPower > 120 {
					propulsionPower = 120
				}
			}

			return VelX, VelY, propulsionPower
		}
	}

	return VelX, VelY, propulsionPower
}

func (g *Game) Update() error {
	loadLevel(g.Level, &g.platforms, &g.basket)

	g.player.VelX, g.player.VelY, g.player.propulsionPower = detectJump(g.player.VelX, g.player.VelY, g.player.x, g.player.y, g.player.r, g.platforms, g.player.propulsionPower)

	g.player.x, g.player.y, g.player.VelX, g.player.VelY = Mouvement(g.player.x, g.player.y, g.player.r, g.player.VelX, g.player.VelY, g.platforms)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Basket
	vector.StrokeRect(screen, float32(g.basket.x), float32(g.basket.y), float32(g.basket.w), float32(g.basket.h), 5, g.basket.clr, true)

	// Player
	vector.DrawFilledCircle(screen, float32(g.player.x), float32(g.player.y), float32(g.player.r), g.player.clr, true)

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
			r:    17,
			clr:  color.RGBA{255, 110, 0, 255},
		},
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
