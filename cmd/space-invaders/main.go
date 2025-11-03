package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	// This package does something important, I'm not entirely sure what but
	// you load it with '_' prefixed since we only care about its side effects
	_ "image/png"
	"log"
	"math/rand"
	"space-invaders/assets"
	"space-invaders/font"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// We save some global configs to use later
const (
	screenWidth  = 750
	screenHeight = 750
	fontSize     = 40
	laserWidth   = 5.0
	laserHeight  = 15.0
)

// We create an State enum to track our GameState
type State int

// C-style enum
const (
	GameOn State = iota
	GameOver
	Title
)

// We set up some more global variables that we are going to use later
// Assets
var shipImage, alien1, alien2, alien3 *ebiten.Image
var aliens [3]*ebiten.Image

// Lasers
var laserColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}
var laserImage = ebiten.NewImage(5, 10)

// Text
var fontFace *text.GoTextFaceSource

// We have a couple init functions, a bunch of boiler plate unfortunately
func init() {
	// We need to extract the file of interest from the embedded pngs
	shipeFile, err := assets.EmbeddedAssets.ReadFile("spaceship.png")
	// We error check and return out incase we failed to extract the image
	if err != nil {
		log.Fatal(err)
	}

	// Once extracted we try to decode it
	img, _, err := image.Decode(bytes.NewReader(shipeFile))

	// We error check again
	if err != nil {
		log.Fatal(err)
	}

	// If everything is kosheer we can assign the ship image
	shipImage = ebiten.NewImageFromImage(img)
}

func init() {
	alien1File, err := assets.EmbeddedAssets.ReadFile("alien_1.png")
	if err != nil {
		log.Fatal(err)
	}

	img, _, err := image.Decode(bytes.NewReader(alien1File))

	if err != nil {
		log.Fatal(err)
	}

	alien1 = ebiten.NewImageFromImage(img)
}

func init() {
	alien2File, err := assets.EmbeddedAssets.ReadFile("alien_2.png")
	if err != nil {
		log.Fatal(err)
	}

	img, _, err := image.Decode(bytes.NewReader(alien2File))

	if err != nil {
		log.Fatal(err)
	}

	alien2 = ebiten.NewImageFromImage(img)
}

func init() {
	alien3File, err := assets.EmbeddedAssets.ReadFile("alien_3.png")
	if err != nil {
		log.Fatal(err)
	}

	img, _, err := image.Decode(bytes.NewReader(alien3File))

	if err != nil {
		log.Fatal(err)
	}

	alien3 = ebiten.NewImageFromImage(img)
}

func init() {
	aliens = [3]*ebiten.Image{alien1, alien2, alien3}
}

func init() {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(font.Font))
	if err != nil {
		log.Fatal(err)
	}

	fontFace = source
}

func init() {
	rand.NewSource(time.Now().UnixNano())
}

// We create an interfacte so that we can check all of the
// entities that have collisions
type CollidableEntity interface {
	PosX() float64
	PosY() float64
	Width() float64
	Height() float64
}

// Our first object is our Alien
type Alien struct {
	type_  int
	posX   float64
	posY   float64
	active bool
}

// We make sure it implements all the interface methods
func (a *Alien) PosX() float64   { return a.posX }
func (a *Alien) PosY() float64   { return a.posY }
func (a *Alien) Width() float64  { return float64(aliens[a.type_].Bounds().Dx()) }
func (a *Alien) Height() float64 { return float64(aliens[a.type_].Bounds().Dy()) }

// This function is a small helper method to move our aliens
func (a *Alien) move(direction float64) {
	a.posX += direction
}

// Our next collidable object is the Laser
type Laser struct {
	posX   float64
	posY   float64
	active bool
}

// We give it all the necessary interface methods
func (l *Laser) PosX() float64   { return l.posX }
func (l *Laser) PosY() float64   { return l.posY }
func (l *Laser) Width() float64  { return 5 }
func (l *Laser) Height() float64 { return 10 }

// The final collidable object is our Ship
type Ship struct {
	posX  float64
	posY  float64
	dir   float64
	speed float64
}

// We give it all the necessary interface methods
func (s *Ship) PosX() float64   { return s.posX }
func (s *Ship) PosY() float64   { return s.posY }
func (s *Ship) Width() float64  { return float64(shipImage.Bounds().Dx()) }
func (s *Ship) Height() float64 { return float64(shipImage.Bounds().Dy()) }

// This function takes any Collidable Entity as the two parameters and returns
// a simple AABB bounds checking for collisions
func checkCollisionAABB(a, b CollidableEntity) bool {
	return a.PosX() < b.PosX()+b.Width() &&
		a.PosX()+a.Width() > b.PosX() &&
		a.PosY() < b.PosY()+b.Height() &&
		a.PosY()+a.Height() > b.PosY()
}

// Game implements the ebiten.Game interface
// It requires Update, Draw, and Layout
type Game struct {
	ship          *Ship
	fleet         []*Alien
	lasers        []*Laser
	alienLaser    []*Laser
	lastAlienShot time.Time
	alienCooldown time.Duration
	alienDir      float64
	alienSpeed    float64
	points        int
	lives         int
	gameState     State
}

// At runtime we init our Game object
func (g *Game) init() {
	g.ship = &Ship{
		posX:  float64((screenWidth - shipImage.Bounds().Dx()) / 2),
		posY:  float64(screenHeight - shipImage.Bounds().Dy() - 50),
		dir:   0,
		speed: 5,
	}
	g.alienDir = 1.0
	g.alienSpeed = 1.0
	g.lastAlienShot = time.Time{}
	g.alienCooldown = 800 * time.Millisecond
	g.points = 0
	g.lives = 3
	g.gameState = GameOn
	g.createFleet()
}

// Method to create the alien fleet
func (g *Game) createFleet() {
	// We iterate over 55 aliens
	// 5 rows and 11 columns
	for row := range 5 {
		for column := range 11 {

			// We forward declare the alienType
			var alienType int
			/*
				// For each row, we create 3 types of aliens
				// the first row is alien type 3
				// the two middle rows are alien type 2
				// the last two rows are alien type 1
			*/
			switch row {
			case 0:
				alienType = 2
			case 1, 2:
				alienType = 1
			default:
				alienType = 0
			}
			// We calculate the width and height for a cell size of 55 pixels
			// and spacing of 100 pixels horizontally and 50 pixels vertically
			x := 100.0 + (float64(column) * 55.0)
			y := 50.0 + (float64(row) * 55.0)

			// We can now add a pointer to the Alien to our fleet
			g.fleet = append(g.fleet, &Alien{
				type_:  alienType,
				posX:   x,
				posY:   y,
				active: true,
			})
		}
	}
}

// Helper method to spawn more aliens when empty
func (g *Game) spawnNextFleet() {
	if len(g.fleet) > 0 {
		return
	} else {
		g.createFleet()
	}
}

// This function controls the alien firing rate
func (g *Game) aliensFireLaser() {
	// We check our cooldown and return if not enough time has elapsed
	if time.Since(g.lastAlienShot) < g.alienCooldown {
		return
	}

	// If the fleet is empty we need to return so we don't seg fault
	if len(g.fleet) == 0 {
		return
	}

	// We create a random number using the number of available aliens
	idx := rand.Intn(len(g.fleet))

	// We can then store the aliens information
	al := g.fleet[idx]
	// We make sure the alien was return
	if al == nil {
		return
	}

	// We now need to calculate the sprite width and height
	sprite := aliens[al.type_]
	w := float64(sprite.Bounds().Dx())
	h := float64(sprite.Bounds().Dy())

	// We can finnaly append a pointer to the Laser to our alien laser slice
	g.alienLaser = append(g.alienLaser, &Laser{
		posX:   al.posX + (w / 2.0),
		posY:   al.posY + h,
		active: true,
	})

	// We update the shot time before returning out to reset our cooldown
	g.lastAlienShot = time.Now()
}

// Helper method to make a new game, we run this inside the main func
func NewGame() ebiten.Game {
	g := &Game{}
	g.init()
	return g
}

// Helper method to move left
func (g *Game) moveLeft() bool {
	return ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
}

// Helper method to move right
func (g *Game) moveRight() bool {
	return ebiten.IsKeyPressed(ebiten.KeyArrowRight)
}

// Helper method to fire a laser
func (g *Game) fireLaser() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeySpace)
}

// Helper method to update the game only while the GameState is GameOn
func (g *Game) updatePlaying() {
	// We check movement
	if g.moveLeft() {
		if g.ship.posX <= 0 {
			g.ship.posX = 0
		} else {
			g.ship.posX += g.ship.speed * -1
		}
	}
	if g.moveRight() {
		// We calculate the rightward bound
		bound := screenWidth - shipImage.Bounds().Dx()
		if int(g.ship.posX) >= bound {
			g.ship.posX = float64(bound)
		} else {
			g.ship.posX += g.ship.speed * 1
		}
	}
	if g.fireLaser() {
		x := float64(g.ship.posX) + (float64(shipImage.Bounds().Dx()) / 2)
		y := float64(g.ship.posY) - 2.0
		g.lasers = append(g.lasers, &Laser{
			posX:   x,
			posY:   y,
			active: true,
		})
	}

	// We update alien positons and do screen cleanup
	g.moveAliens()
	g.updateLasers()
	g.updateAlienLasers()
	g.resolveCollisions()
	g.deleteLasers()
	g.deleteAliens()
	g.deleteAlienLasers()
	g.spawnNextFleet()
	g.aliensFireLaser()

	// We also check for the number of lives to end the game
	if g.lives <= 0 {
		g.gameState = GameOver
	}
}

// This is an Interface method needed for our Game to run
func (g *Game) Update() error {
	// We store the escape key as a game close option
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	// We only update the sim if the game state is on
	if g.gameState == GameOn {
		g.updatePlaying()
	}
	return nil
}

// Helper method to update the laser positons
func (g *Game) updateAlienLasers() {
	for _, laser := range g.alienLaser {
		laser.posY += 5.0
	}
}

// This is another interface method needed for our Game to work
func (g *Game) Draw(screen *ebiten.Image) {
	// Dark purple screen
	screen.Fill(color.RGBA{28, 3, 51, 255})
	// We draw the events needed when the game is on
	g.drawShip(screen)
	g.drawAliens(screen)
	g.drawLaser(screen)
	g.drawAlienLasers(screen)
	g.drawScore(screen)
}

// Helper method to update laser positons
func (g *Game) updateLasers() {
	for _, laser := range g.lasers {
		laser.posY -= 5.0
	}
}

// Helper method to delete the lasers fired by the player
func (g *Game) deleteLasers() {
	/*
		// We create a new slice based on the previous slice
		// The [:0] operation creates an empty slice with the same capacity
		// and underlying array as the original. It lets us reuse memory space
		// in a more efficient manner
	*/
	destination := g.lasers[:0]
	// We loop over the slice of lasers
	for _, l := range g.lasers {
		// If the laser is active we move it to the new slice
		if l.active {
			destination = append(destination, l)
		}
	}
	// We can now reassign the game's lasers
	g.lasers = destination
}

// Helper method to delete the lasers fired by the alien fleet
// basically the same as above
func (g *Game) deleteAlienLasers() {
	destination := g.alienLaser[:0]
	for _, al := range g.alienLaser {
		if al.active {
			destination = append(destination, al)
		}
	}
	g.alienLaser = destination
}

// Helper method to delete the inactive aliens
// basically the same as the previous two methods
func (g *Game) deleteAliens() {
	destination := g.fleet[:0]
	for _, a := range g.fleet {
		if a.active {
			destination = append(destination, a)
		}
	}
	g.fleet = destination
}

// Method to resolve laser, alien, and ship collisions
func (g *Game) resolveCollisions() {
	// We loop over the active player lasers
	for li := range g.lasers {
		if !g.lasers[li].active {
			continue
		}

		// We make sure the laser is on screen
		if g.lasers[li].posY < 0 {
			g.lasers[li].active = false
			continue
		}

		// We can now loop over the aliens
		for ai := range g.fleet {
			// We check collsions and break out if there was one
			if checkCollisionAABB(g.lasers[li], g.fleet[ai]) {
				g.lasers[li].active = false
				g.fleet[ai].active = false
				g.points++
				break
			}
		}
	}
	// We loop over the alien lasers that are active
	for al := range g.alienLaser {
		if !g.alienLaser[al].active {
			continue
		}

		// If the laser goes below the screen we remove it as well
		if g.alienLaser[al].posY > screenHeight {
			g.alienLaser[al].active = false
			continue
		}

		// We then check for collsions with the player and decrement our lives
		if checkCollisionAABB(g.alienLaser[al], g.ship) {
			g.alienLaser[al].active = false
			g.lives--
			break
		}
	}
}

// Helper method to draw the lasers
func (g *Game) drawLaser(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	for _, laser := range g.lasers {
		vector.FillRect(laserImage, 0, 0, laserWidth, laserHeight, laserColor, false)
		op.GeoM.Translate(laser.posX, laser.posY)
		screen.DrawImage(laserImage, op)
		// We need to reset each time to update the positions individually
		op.GeoM.Reset()
	}
}

// Helper method to draw alien lasers
func (g *Game) drawAlienLasers(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	for _, laser := range g.alienLaser {
		vector.FillRect(laserImage, 0, 0, laserWidth, laserHeight, laserColor, false)
		op.GeoM.Translate(laser.posX, laser.posY)
		screen.DrawImage(laserImage, op)
		// We need to reset each time to update the positions individually
		op.GeoM.Reset()
	}
}

// Helper method to draw our player
func (g *Game) drawShip(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(g.ship.posX), float64(g.ship.posY))
	screen.DrawImage(shipImage, op)
}

// Helper method to draw the aliens
func (g *Game) drawAliens(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	for _, alien := range g.fleet {
		op.GeoM.Translate(alien.posX, alien.posY)
		screen.DrawImage(aliens[alien.type_], op)
		// We need to reset each time to update the positions individually
		op.GeoM.Reset()
	}
}

// Helper method to draw our score
func (g *Game) drawScore(screen *ebiten.Image) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(screenWidth, 0)
	op.ColorScale.ScaleWithColor(color.RGBA{R: 150, B: 200, G: 150, A: 255})
	op.LineSpacing = fontSize
	op.PrimaryAlign = text.AlignEnd
	text.Draw(screen, fmt.Sprintf("%05d", g.points), &text.GoTextFace{
		Source: fontFace,
		Size:   fontSize,
	}, op)
}

// Helper method to move the aliens on screen
func (g *Game) moveAliens() {
	// We loop over the aliens
	for _, alien := range g.fleet {
		// We need to bounds check the right side
		if (alien.posX + float64(aliens[alien.type_].Bounds().Dx())) > (screenWidth - 25) {
			// We change direction and move the aliens down
			g.alienDir = -1 * g.alienSpeed
			g.aliensDown()
		}

		// We bounds check the right side of the screen
		if alien.posX < 25 {
			// We change direction and move down
			g.alienDir = 1 * g.alienSpeed
			g.aliensDown()
		}
		// We can now call the helper method to move the position
		// of the alien given the direction
		alien.move(g.alienDir)
	}
}

// helper method to shift the aliens down
func (g *Game) aliensDown() {
	for _, alien := range g.fleet {
		alien.posY += 5.0
	}
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	// We need to set the game size
	ebiten.SetWindowSize(screenWidth, screenHeight)
	// We can now set a title
	ebiten.SetWindowTitle("Go - Space Invaders")
	// We start our new game loop using RunGame, we pass in a new Game using
	// our helper method
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
