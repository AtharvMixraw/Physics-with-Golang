package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth = 800
	screenHeight = 900

	boundaryCX = float64(screenWidth)/2
	boundaryCY = 480
	boundaryR  = 380

	ballRadius = 10
	sampleRate = 44100

	minBalls = 1
	maxBalls = 100
)

type Ball struct {
	X, Y	float64
	VX, VY  float64
	R 		float64
	Col 	color.RGBA
}

func randomBall() *Ball {
	angle := rand.Float64() * 2 * math.Pi
	dist  := rand.Float64() * (boundaryR - 3*ballRadius)
	speed := rand.Float64() * 2.5 + 2.5
	dir	  := rand.Float64() * 2 * math.Pi

	hue   := rand.Float64() 

	return &Ball {
		X: boundaryCX + dist * math.Cos(angle),
		Y: boundaryCY + dist * math.Sin(angle),
		VX:  speed * math.Cos(dir),
		VY:  speed * math.Sin(dir),
		R:   ballRadius,
		Col: colorFromHue(hue),
	}
}

func colorFromHue(h float64) color.RGBA {
	s, v := 0.65, 1.0
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	case 5:
		r, g, b = v, p, q
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

type Button struct {
	W,H,X,Y float64
	Label 	string
}

func (b *Button) Contains (x,y float64) bool {
	return x >= b.X && x <= b.X + b.W && y >= b.Y && y <= b.Y + b.H
}

type Game struct {
	balls 				[]*Ball
	audioCtx 			*audio.Context
	collideWav			[]byte
	wallWav				[]byte
	btnPlus, btnMinus 	*Button
}

func NewGame() *Game { 
	g := &Game{
		audioCtx: audio.NewContext(sampleRate),
	}
	g.collideWav = generateTone(880, 0.04, 0.35)
	g.wallWav = generateTone(440, 0.03, 0.2)

	g.balls = append(g.balls, randomBall())
	g.btnMinus = &Button{X: boundaryCX - 90, Y: 30, W: 60, H: 40, Label: "-"}
	g.btnPlus = &Button{X: boundaryCX + 30, Y: 30, W: 60, H: 40, Label: "+"}

	return g
}

func generateTone(freq, durSec, vol float64) []byte {
	n := int(sampleRate * durSec)
	buf := new(bytes.Buffer)

	for i:=0; i<n; i++ {
		t := float64(i) / sampleRate
		envelop := math.Exp(-14 * t / durSec)
		sample := math.Sin(2*math.Pi*freq*t) * envelop * vol
		v := int16(sample * 32767)
		binary.Write(buf, binary.LittleEndian, v) // left
		binary.Write(buf, binary.LittleEndian, v) // right
	}
	return buf.Bytes()
}

var activeSounds int

const maxConcurrentSounds = 24

func (g *Game) playSound(data []byte) {
	if activeSounds > maxConcurrentSounds {
		return
	}
	p := g.audioCtx.NewPlayerFromBytes(data)
	activeSounds++
	p.Play()
	activeSounds--
}

func (g *Game) addBalls(n int) {
	for i := 0; i < n; i++ {
		if len(g.balls) >= maxBalls {
			return
		}
		g.balls = append(g.balls, randomBall())
	}
}
 
func (g *Game) removeBalls(n int) {
	for i := 0; i < n; i++ {
		if len(g.balls) <= minBalls {
			return
		}
		g.balls = g.balls[:len(g.balls)-1]
	}
}

func (g *Game) handleInput() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		x , y := float64(mx), float64(my)
		if g.btnPlus.Contains(x, y) {
			g.addBalls(1)
		} else if g.btnMinus.Contains(x, y) {
			g.removeBalls(1)
		}
	}
}

func (g *Game) Update() error {
	g. handleInput()

	for _, b := range g.balls {
		b.X += b.VX
		b.Y += b.VY

		dx := b.X - boundaryCX
		dy := b.Y - boundaryCY
		dist := math.Hypot(dx, dy)
		if dist + b.R > boundaryR {
			nx, ny := dx/dist , dy/dist
			overlap := dist + b.R - boundaryR
			b.X -= nx * overlap
			b.Y -= ny * overlap
			// reflect velocity across normal
			dot := b.VX*nx + b.VY*ny
			b.VX -= 2 * dot * nx
			b.VY -= 2 * dot * ny
			g.playSound(g.wallWav)
		}
	}
	for i := 0; i < len(g.balls); i++ {
		for j := i + 1; j < len(g.balls); j++ {
			a, b := g.balls[i], g.balls[j]
			dx := b.X - a.X
			dy := b.Y - a.Y
			dist := math.Hypot(dx, dy)
			minDist := a.R + b.R
			if dist == 0 {
				dist = 0.01
				dx = 0.01
			}
			if dist < minDist {
				nx, ny := dx/dist, dy/dist
				// separate overlapping balls
				overlap := (minDist - dist) / 2
				a.X -= nx * overlap
				a.Y -= ny * overlap
				b.X += nx * overlap
				b.Y += ny * overlap
 
				// relative velocity along normal
				rvx := b.VX - a.VX
				rvy := b.VY - a.VY
				velAlongNormal := rvx*nx + rvy*ny
				if velAlongNormal < 0 {
					// equal mass elastic collision: swap normal components
					a.VX += velAlongNormal * nx
					a.VY += velAlongNormal * ny
					b.VX -= velAlongNormal * nx
					b.VY -= velAlongNormal * ny
					g.playSound(g.collideWav)
				}
			}
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
 
	// Boundary circle (drawn as a ring of short segments).
	drawCircleOutline(screen, boundaryCX, boundaryCY, boundaryR, color.RGBA{170, 210, 255, 255}, 3)
 
	for _, b := range g.balls {
		vector.DrawFilledCircle(screen, float32(b.X), float32(b.Y), float32(b.R), b.Col, true)
	}
 
	drawButton(screen, g.btnMinus, color.RGBA{60, 60, 70, 255})
	drawButton(screen, g.btnPlus, color.RGBA{60, 60, 70, 255})
 
	ebitenutil.DebugPrintAt(screen, "Balls: "+fmt.Sprint(len(g.balls)), int(boundaryCX)-35, 80)
	ebitenutil.DebugPrintAt(screen, "click + / - to add or remove balls", 20, screenHeight-24)
}
 
func drawButton(screen *ebiten.Image, b *Button, col color.RGBA) {
	vector.DrawFilledRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), col, true)
	vector.StrokeRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), 2, color.RGBA{200, 200, 200, 255}, true)
	tx := int(b.X + b.W/2 - 4)
	ty := int(b.Y + b.H/2 - 8)
	ebitenutil.DebugPrintAt(screen, b.Label, tx, ty)
}
 
func drawCircleOutline(screen *ebiten.Image, cx, cy, r float64, col color.RGBA, width float32) {
	const segments = 128
	var prevX, prevY float64
	for i := 0; i <= segments; i++ {
		theta := 2 * math.Pi * float64(i) / segments
		x := cx + r*math.Cos(theta)
		y := cy + r*math.Sin(theta)
		if i > 0 {
			vector.StrokeLine(screen, float32(prevX), float32(prevY), float32(x), float32(y), width, col, true)
		}
		prevX, prevY = x, y
	}
}
 
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}
 
func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Bouncing Balls")
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}