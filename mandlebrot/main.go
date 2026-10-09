package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth = 800
	screenHeight = 600
	maxIter = 200
)
type Button struct {
	x, y, w, h int
	label      string
}

func (b Button) contains(mx, my int) bool {
	return mx >= b.x && mx < b.x+b.w && my >= b.y && my < b.y+b.h
}

type Game struct {
	pixels 		[]byte
	centerX		float64
	centerY		float64
	scale		float64	
	dirty		bool
	dragging       bool
	lastX, lastY   int
	btnIn, btnOut Button
}

func NewGame() *Game {
	return &Game { 
		pixels: make([]byte, screenHeight*screenWidth*4),
		centerX: -0.5,
		centerY: 0,
		scale: 3.5 / screenWidth,
		dirty: true,
		btnOut: Button{20, 20, 44, 44, "-"},
		btnIn:  Button{74, 20, 44, 44, "+"},
	}
}

func (g *Game) zoomAt(px, py, factor float64) {
	cx := g.centerX + px*g.scale
	cy := g.centerY + py*g.scale

	g.scale *= factor

	g.centerX = cx - px*g.scale
	g.centerY = cy - py*g.scale
	g.dirty = true
}

func escapeTime(cx, cy float64) int {
	x, y := 0.0, 0.0
	for i:=0 ; i< maxIter; i++ {
		x2 , y2 := x*x , y*y
		if x2 + y2 > 4 {
			return i
		}
		y = 2*x*y + cy
		x = x2 - y2 + cx
	}
	return maxIter
}

func palette(t float64) (byte, byte, byte) {
	r := 0.5 + 0.5*math.Cos(2*math.Pi*(t+0.00))
	g := 0.5 + 0.5*math.Cos(2*math.Pi*(t+0.10))
	b := 0.5 + 0.5*math.Cos(2*math.Pi*(t+0.20))
	return byte(r * 255), byte(g * 255), byte(b * 255)
}

func colorFor(iter int) (byte, byte, byte) {
	if iter == maxIter {
		return 0, 0, 0
	}
	return palette(float64(iter) * 0.02)
}

func (g *Game) render() {
	for py := 0; py < screenHeight; py++ {
		for px := 0; px < screenWidth; px++ {
			cx := g.centerX + (float64(px)-screenWidth/2)*g.scale
			cy := g.centerY + (float64(py)-screenHeight/2)*g.scale

			r, gr, b := colorFor(escapeTime(cx, cy))

			i := (py*screenWidth + px) * 4
			g.pixels[i] = r
			g.pixels[i+1] = gr
			g.pixels[i+2] = b
			g.pixels[i+3] = 255
		}
	}
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()
	onButton := g.btnIn.contains(mx, my) || g.btnOut.contains(mx, my)

	// Buttons zoom around the screen center
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.btnIn.contains(mx, my) {
			g.zoomAt(0, 0, 0.5)
		} else if g.btnOut.contains(mx, my) {
			g.zoomAt(0, 0, 2)
		}
	}

	// Wheel zooms around the cursor
	_, wy := ebiten.Wheel()
	if wy != 0 {
		g.zoomAt(float64(mx)-screenWidth/2, float64(my)-screenHeight/2, math.Pow(0.85, wy))
	}

	// Drag to pan (but not when the mouse is over a button)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && !onButton {
		if g.dragging && (mx != g.lastX || my != g.lastY) {
			g.centerX -= float64(mx-g.lastX) * g.scale
			g.centerY -= float64(my-g.lastY) * g.scale
			g.dirty = true
		}
		g.dragging = true
		g.lastX, g.lastY = mx, my
	} else {
		g.dragging = false
	}

	if g.dirty {
		g.render()
		g.dirty = false
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.WritePixels(g.pixels)
	drawButton(screen, g.btnOut)
	drawButton(screen, g.btnIn)
}

func drawButton(screen *ebiten.Image, b Button) {
	r := image.Rect(b.x, b.y, b.x+b.w, b.y+b.h)
	screen.SubImage(r).(*ebiten.Image).Fill(color.RGBA{30, 30, 40, 255})
	ebitenutil.DebugPrintAt(screen, b.label, b.x+b.w/2-3, b.y+b.h/2-8)
}

func (g *Game) Layout(w, h int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Mandelbrot")
	if err := ebiten.RunGame(NewGame()); err != nil {
		panic(err)
	}
}