package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/JeremyBelleIsle/gameutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 640
	screenHeight = 480

	netW = 50.0
	netH = 75.0
)

type player struct {
	x, y            float64
	r               float64
	propulsionPower float64
	VelX            float64
	VelY            float64
	clr             color.RGBA
}

type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StateLevelComplete
)

type Point struct {
	X, Y float64
}

type level struct {
	platforms []gameutil.Platform
	basketX   float64
	basketY   float64
}

type Game struct {
	platforms    []gameutil.Platform
	points       []Point
	currentLevel level
	player       player
	starImage    *ebiten.Image
	attempt      int
	level        int
	Score        int
	stars        int
	state        GameState
}

var mplusSource *text.GoTextFaceSource

//go:embed RobotoMono-VariableFont_wght.ttf
var roboto []byte

var faceSource *text.GoTextFaceSource

func Ballphysics(px, py, vx, vy, pr float64, platforms []gameutil.Platform) (float64, float64, float64, float64) {
	vx *= 0.97
	vy += 0.25

	newX := px + vx
	newY := py + vy

	for _, p := range platforms {
		if gameutil.CircleRectCollision(newX, py, pr, p.X, p.Y, p.W, p.H) {
			vx = -vx * 0.7
			newX = px
		}
	}

	for _, p := range platforms {
		if gameutil.CircleRectCollision(newX, newY, pr, p.X, p.Y, p.W, p.H) {
			vy = -vy * 0.6
			newY = py
		}
	}

	return newX, newY, vx, vy
}

func stars(score int) int {
	if score >= 800 {
		return 3
	}
	if score >= 500 {
		return 2
	}
	if score > 0 {
		return 1
	}
	return 0
}

func input(p *player, attempt *int, currentLevel level) (float64, float64, level) {

	if p.y > screenHeight-48 {
		speed := 2.0
		canMoveLeft := true
		canMoveRight := true
		for _, pt := range currentLevel.platforms {
			if gameutil.CircleRectCollision(p.x-speed, p.y-10, p.r, pt.X, pt.Y, pt.W, pt.H) {
				canMoveLeft = false
			}

			if gameutil.CircleRectCollision(p.x+speed, p.y-10, p.r, pt.X, pt.Y, pt.W, pt.H) {
				canMoveRight = false
			}
		}

		if canMoveRight {
			if ebiten.IsKeyPressed(ebiten.KeyRight) {
				p.x += speed

				p.y = screenHeight - 100
			}
		}

		if canMoveLeft {
			if ebiten.IsKeyPressed(ebiten.KeyLeft) {
				p.x -= speed

				p.y = screenHeight - 100
			}
		}
	}

	if *attempt < 4 {
		return p.x, p.y, currentLevel
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		p.resetPlayer()
		*attempt = 0
		currentLevel.platforms, currentLevel.basketX, currentLevel.basketY = GenerateLevel(rand.Intn(2) == 1, rand.Intn(3)+3, 50, 120, currentLevel.basketX, currentLevel.basketY, level{})
	}

	return p.x, p.y, currentLevel
}

func simulateTrajectory(x, y float64, propulsionPower float64, r float64, platforms []gameutil.Platform) []Point {

	points := make([]Point, 0)

	vx := propulsionPower / 10
	vy := -propulsionPower / 5

	for i := 0; i < int(propulsionPower); i++ {
		x, y, vx, vy = Ballphysics(x, y, vx, vy, r, platforms)
		if i%18 == 0 {
			points = append(points, Point{x, y})
		}
	}

	return points
}

func GenerateLevel(wall bool, platformsNumber int, MinDist, Maxdist float64, basketX float64, basketY float64, currentLevel level) ([]gameutil.Platform, float64, float64) {

	if Maxdist < MinDist {
		panic("maxDist < minDist | impossible to make random on it")
	}

	const maxJumpY = 120.0

	platforms := []gameutil.Platform{}

	// place net
	basketX = screenWidth - netW - 40
	basketY = 150

	w := netW
	h := 75.0

	platforms = append(platforms,
		gameutil.Platform{X: basketX, Y: basketY + h - 5, W: w, H: 5},
		gameutil.Platform{X: basketX, Y: basketY, W: 5, H: h},
		gameutil.Platform{X: basketX + w - 5, Y: basketY, W: 5, H: h},
	)

	// START SAFE ZONE
	platX := 40.0
	platY := float64(screenHeight - 200)

	for i := 0; i < platformsNumber; i++ {

		// clamp vertical
		if platY < 80 {
			platY = 80
		}
		if platY > screenHeight-120 {
			platY = screenHeight - 120
		}

		platforms = append(platforms, gameutil.Platform{
			X:   platX,
			Y:   platY,
			W:   75,
			H:   15,
			Clr: color.RGBA{128, 0, 128, 255},
		})

		// distance horizontale contrôlée
		distX := MinDist + rand.Float64()*(Maxdist-MinDist)
		platX += 75 + distX

		// variation verticale limitée
		deltaY := rand.Float64()*maxJumpY - (maxJumpY / 2)
		platY += deltaY
	}

	// mur sécurisé
	if wall {

		// plateforme garantie vers le net
		lastX := basketX - 140
		lastY := basketY + 40

		platforms = append(platforms, gameutil.Platform{
			X:   lastX,
			Y:   lastY,
			W:   75,
			H:   15,
			Clr: color.RGBA{128, 0, 128, 255},
		})

		XWall := float64(rand.Intn(screenWidth-400) + 200)

		// on ne place pas le mur si ça coupe la dernière plateforme
		if math.Abs(XWall-lastX) > 120 {

			Height := float64(rand.Intn(screenHeight / 2))
			Spacing := float64(rand.Intn(200) + 120)

			platforms = append(platforms,
				gameutil.Platform{X: XWall, Y: 0, W: 40, H: Height, Clr: color.RGBA{128, 0, 128, 255}},
				gameutil.Platform{X: XWall, Y: Height + Spacing, W: 40, H: screenHeight - Height - Spacing, Clr: color.RGBA{128, 0, 128, 255}},
			)
		}
	}

	// bordures
	platforms = append(platforms,
		gameutil.Platform{X: 0, Y: 0, W: screenWidth, H: 20, Clr: color.RGBA{128, 0, 128, 255}},
		gameutil.Platform{X: 0, Y: 0, W: 20, H: screenHeight, Clr: color.RGBA{128, 0, 128, 255}},
		gameutil.Platform{X: screenWidth - 20, Y: 0, W: 20, H: screenHeight, Clr: color.RGBA{128, 0, 128, 255}},
	)

	return platforms, basketX, basketY
}

func NetIsAccessible(basketX, basketY float64, platforms []gameutil.Platform) bool {
	for _, p := range platforms {

		if p.X == basketX && p.Y >= basketY && p.Y <= basketY+netH {
			continue
		}

		if gameutil.RectColl(basketX, basketY-50, netW, netH+50, p.X, p.Y, p.W, p.H) {
			return false
		}

	}

	return true
}

func detectJump(VelX, VelY float64, px, py, pr float64, platforms []gameutil.Platform, propulsionPower float64, Points []Point) (float64, float64, float64, []Point) {

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

				Points = simulateTrajectory(px, py, propulsionPower, 17, platforms)
			} else {
				Points = Points[:0]
			}

			return VelX, VelY, propulsionPower, Points
		}
	}

	return VelX, VelY, propulsionPower, Points
}

func (p *player) resetPlayer() {
	p.x = 50
	p.y = screenHeight/2 - 50
	p.VelX = 0
	p.VelY = 0
}

func LevelIsDone(basketX, basketY float64, pX, pY, pR float64) bool {
	return gameutil.CircleRectCollision(pX, pY, pR, basketX, basketY, netW, netH-20)
}

func (p *player) Dead(attempt *int) {
	if p.y >= screenHeight {
		*attempt++

		p.resetPlayer()
	}
}

func (g *Game) Update() error {

	switch g.state {

	case StateMenu:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.state = StatePlaying
			g.player.resetPlayer()
			g.currentLevel.platforms, g.currentLevel.basketX, g.currentLevel.basketY = GenerateLevel(true, rand.Intn(3)+3, 50, 120, g.currentLevel.basketX, g.currentLevel.basketY, level{})

			for i := 0; i < 100; i++ {
				if NetIsAccessible(g.currentLevel.basketX, g.currentLevel.basketY, g.currentLevel.platforms) {
					break
				}

				g.currentLevel.platforms, g.currentLevel.basketX, g.currentLevel.basketY = GenerateLevel(true, rand.Intn(3)+3, 50, 120, g.currentLevel.basketX, g.currentLevel.basketY, level{})
			}
		}
		return nil

	case StatePlaying:

		g.player.x, g.player.y, g.currentLevel = input(&g.player, &g.attempt, g.currentLevel)

		g.player.VelX, g.player.VelY, g.player.propulsionPower, g.points = detectJump(g.player.VelX, g.player.VelY, g.player.x, g.player.y, g.player.r, g.currentLevel.platforms, g.player.propulsionPower, g.points)

		g.player.x, g.player.y, g.player.VelX, g.player.VelY = Ballphysics(g.player.x, g.player.y, g.player.VelX, g.player.VelY, g.player.r, g.currentLevel.platforms)

		g.player.Dead(&g.attempt)

		if LevelIsDone(g.currentLevel.basketX, g.currentLevel.basketY, g.player.x, g.player.y, g.player.r) {
			g.stars = stars(1000 - g.attempt*200)
			g.state = StateLevelComplete
		}

	case StateLevelComplete:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.currentLevel.platforms, g.currentLevel.basketX, g.currentLevel.basketY = GenerateLevel(rand.Intn(2) == 1, rand.Intn(3)+3, 50, 120, g.currentLevel.basketX, g.currentLevel.basketY, g.currentLevel)

			for i := 0; i < 100; i++ {
				if NetIsAccessible(g.currentLevel.basketX, g.currentLevel.basketY, g.currentLevel.platforms) {
					break
				}

				g.currentLevel.platforms, g.currentLevel.basketX, g.currentLevel.basketY = GenerateLevel(true, rand.Intn(3)+3, 50, 120, g.currentLevel.basketX, g.currentLevel.basketY, level{})
			}

			g.player.resetPlayer()
			g.level++
			g.attempt = 0
			g.state = StatePlaying
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyM) {
			g.state = StateMenu
		}
		return nil
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	switch g.state {

	case StateMenu:
		screen.Fill(color.RGBA{0, 175, 0, 255})
		gameutil.DrawText("BASKET BALL", 50, screenWidth, screenWidth/2-200, 150, 0, screen, color.RGBA{255, 255, 0, 255}, mplusSource)
		gameutil.DrawText("Press SPACE to start", 25, screenWidth, screenWidth/2-170, 300, 0, screen, color.RGBA{255, 255, 255, 255}, faceSource)
		return

	case StatePlaying:
		screen.Fill(color.RGBA{135, 206, 235, 255})

		for _, plat := range g.currentLevel.platforms {
			vector.DrawFilledRect(screen, float32(plat.X), float32(plat.Y), float32(plat.W), float32(plat.H), plat.Clr, true)
		}

		vector.StrokeRect(screen, float32(g.currentLevel.basketX), float32(g.currentLevel.basketY), float32(netW), float32(netH), 5, color.RGBA{0, 0, 255, 255}, true)

		vector.DrawFilledCircle(screen, float32(g.player.x), float32(g.player.y), float32(g.player.r), g.player.clr, true)

		for _, pt := range g.points {
			vector.DrawFilledCircle(screen, float32(pt.X), float32(pt.Y), 5, color.RGBA{255, 255, 255, 120}, true)
		}

		gameutil.DrawText(fmt.Sprintf("Level: %d/inf", g.level), 30, screenWidth, 20, 20, 0, screen, color.RGBA{255, 255, 0, 255}, mplusSource)

		if g.attempt >= 4 {
			gameutil.DrawText("Press S to skip", 25, screenWidth, screenWidth-200, 20, 0, screen, color.RGBA{255, 255, 255, 255}, faceSource)
		}

	case StateLevelComplete:

		vector.DrawFilledRect(screen, 0, 0, screenWidth, screenHeight, color.RGBA{0, 0, 0, 180}, true)

		gameutil.DrawText("LEVEL COMPLETE!", 40, screenWidth, screenWidth/2-150, 150, 0, screen, color.RGBA{255, 255, 0, 255}, mplusSource)

		for i := 0; i < g.stars; i++ {
			op := &ebiten.DrawImageOptions{}
			x := float64(screenWidth/2 - 200 + (i * 130))
			y := 230.0
			op.GeoM.Scale(0.22, 0.22)
			op.GeoM.Translate(x, y)
			screen.DrawImage(g.starImage, op)
		}

		gameutil.DrawText("Press SPACE to continue.", 25, screenWidth, screenWidth/2-160, 350, 0, screen, color.RGBA{255, 255, 255, 255}, faceSource)

		gameutil.DrawText("Press M to go to the menu.", 20, screenWidth, screenWidth/2-160, 450, 0, screen, color.RGBA{255, 255, 255, 255}, faceSource)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func greet(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
  <title>Bouton fuyant 😂</title>
  <style>
    body {
      height: 100vh;
      margin: 0;
      overflow: hidden;
      display: flex;
      justify-content: center;
      align-items: center;
      font-family: sans-serif;
    }

    button {
      position: absolute;
      padding: 15px 30px;
      font-size: 18px;
      cursor: pointer;
    }
  </style>
</head>
<body>

<button id="runaway">Ne me clique pas!</button>

<script>
const btnTest = document.getElementById("runaway");

function moveButton() {
  const x = Math.random() * (window.innerWidth - btnTest.offsetWidth);
  const y = Math.random() * (window.innerHeight - btnTest.offsetHeight);

  btnTest.style.left = x + "px";
  btnTest.style.top = y + "px";
}

btnTest.addEventListener("click", () => {
  alert("Ouille... tu m'as blessé");
});
</script>

</body>
</html>`)
}

func hello(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<html>
<head>
</head>
<body>
    <h3>This is the HELLO page!</h3>
</body>
</html>`)
}

func main() {
	// http.HandleFunc("/", greet)
	// http.HandleFunc("/hello.html", hello)
	// http.ListenAndServe("127.0.0.1:8080", nil)

	rand.Seed(time.Now().UnixNano())

	ebiten.SetFullscreen(true)

	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.PressStart2P_ttf))

	s2, err2 := text.NewGoTextFaceSource(bytes.NewReader(roboto))

	if err != nil {
		log.Fatal(err)
	}
	if err2 != nil {
		log.Fatal(err2)
	}

	mplusSource = s
	faceSource = s2
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Basket ball")

	g := &Game{
		state: StateMenu,
		level: 1,

		player: player{
			x:    50,
			y:    screenHeight/2 - 50,
			VelX: 0,
			VelY: 3,
			r:    17,
			clr:  color.RGBA{255, 110, 0, 255},
		},
	}

	img, _, err := ebitenutil.NewImageFromFile("star (1).png")
	if err != nil {
		log.Fatal(err)
	}

	g.starImage = img

	g.currentLevel.platforms, g.currentLevel.basketX, g.currentLevel.basketY = GenerateLevel(true, rand.Intn(3)+3, 50, 120, g.currentLevel.basketX, g.currentLevel.basketY, level{})

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
