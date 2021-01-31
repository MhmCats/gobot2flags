package main

import (
	"errors"
	"fmt"
	"strings"
)

type CircuitBoard struct {
	width, height int
	chips         []Chip
	startPos      Position
}

func NewCircuitBoard(width, height int) *CircuitBoard {
	return &CircuitBoard{
		width:  width,
		height: height,
		chips:  make([]Chip, width*height),
	}
}

func CircuitBoardFromString(s string) (*CircuitBoard, error) {
	s = strings.TrimSpace(s)
	rows := strings.Split(s, "\n")

	// First check every line start and ends with '|'
	for i, row := range rows {
		if len(row) < 2 || row[0] != '|' || row[len(row)-1] != '|' {
			return nil, fmt.Errorf("line %d should start and end with a '|'", i+1)
		}
		rows[i] = row[1 : len(row)-1]
	}

	if len(rows)%2 != 1 {
		return nil, errors.New("need odd number of lines")
	}
	height := (len(rows) + 1) / 2
	if height == 0 {
		return nil, errors.New("need at least 1 row")
	}
	lr0 := len(rows[0])
	width := (lr0 + 4) / 6
	if width*6-4 != lr0 {
		return nil, errors.New("wrong length for line 1")
	}
	b := NewCircuitBoard(width, height)
	hasStartPos := false
	for i, row := range rows {
		y := i / 2
		if i%2 == 0 {
			// This is a row of chips
			for x := 0; x < width; x++ {

				// Set the chip
				chipCode := row[x*6 : x*6+2]
				chipType, ok := chipTypeMap[chipCode]
				if !ok {
					return nil, fmt.Errorf("invalid chip code at line %d, column %d: %q", i+1, x*6+1, chipCode)
				}
				if chipType == StartChip {
					if hasStartPos {
						return nil, fmt.Errorf("Only one start chip allowed: found second at line %d, column %d", i+1, x*6+1)
					}
					b.startPos = Position{X: x, Y: y}
					hasStartPos = true
				}
				b.SetChipAt(x, y, b.ChipAt(x, y).WithType(chipType))
				if x == width-1 {
					continue
				}

				// Set the arrow
				switch arrCode := row[x*6+3 : x*6+5]; arrCode {
				case "y>":
					b.SetChipAt(x, y, b.ChipAt(x, y).WithArrowYes(East))
				case "n>":
					b.SetChipAt(x, y, b.ChipAt(x, y).WithArrowNo(East))
				case "->":
					b.SetChipAt(x, y, b.ChipAt(x, y).WithArrowYes(East))
				case "<y":
					b.SetChipAt(x+1, y, b.ChipAt(x+1, y).WithArrowYes(West))
				case "<n":
					b.SetChipAt(x+1, y, b.ChipAt(x+1, y).WithArrowNo(West))
				case "<-":
					b.SetChipAt(x+1, y, b.ChipAt(x+1, y).WithArrowYes(West))
				case "..", "  ":
					// No arrow
				default:
					return nil, fmt.Errorf("invalid arrow code at line %d, column %d: %q", i+1, x*6+4, arrCode)
				}
			}
		} else {
			// This is a row of arrows only
			for x := 0; x < width; x++ {
				switch arrCode := row[x*6 : x*6+2]; arrCode {
				case "yv":
					b.SetChipAt(x, y, b.ChipAt(x, y).WithArrowYes(South))
				case "nv":
					b.SetChipAt(x, y, b.ChipAt(x, y).WithArrowNo(South))
				case " v":
					b.SetChipAt(x, y, b.ChipAt(x, y).WithArrowYes(South))
				case "y^":
					b.SetChipAt(x, y+1, b.ChipAt(x, y+1).WithArrowYes(North))
				case "n^":
					b.SetChipAt(x, y+1, b.ChipAt(x, y+1).WithArrowNo(North))
				case " ^":
					b.SetChipAt(x, y+1, b.ChipAt(x, y+1).WithArrowYes(North))
				case "  ", "..":
					// No arrow
				default:
					return nil, fmt.Errorf("invalid arrow code at line %d, column %d: %q", i+1, x*6, arrCode)
				}
			}
		}
	}
	if !hasStartPos {
		return nil, errors.New("start chip missing")
	}
	return b, nil
}

func (b *CircuitBoard) chipIndex(x, y int) int {
	return x + b.width*y
}

func (b *CircuitBoard) Contains(x, y int) bool {
	return x >= 0 && x < b.width && y >= 0 && y < b.height
}

func (b *CircuitBoard) ChipAt(x, y int) Chip {
	return b.chips[b.chipIndex(x, y)]
}

func (b *CircuitBoard) SetChipAt(x, y int, c Chip) {
	if c.Type() == StartChip {
		currentStart := b.ChipAt(b.startPos.X, b.startPos.Y)
		if currentStart.Type() == StartChip {
			b.chips[b.chipIndex(b.startPos.X, b.startPos.Y)] = currentStart.WithType(NoChip)
		}
		b.startPos = Position{x, y}
	}
	p := Position{x, y}
	if o, ok := c.ArrowYes(); ok {
		b.deleteArrow(p.Move(o.VelocityForward()), o.Reverse())
	}
	if o, ok := c.ArrowNo(); ok {
		b.deleteArrow(p.Move(o.VelocityForward()), o.Reverse())
	}
	b.chips[b.chipIndex(x, y)] = c
}

func (b *CircuitBoard) deleteArrow(p Position, o Orientation) {
	b.chips[b.chipIndex(p.X, p.Y)] = b.ChipAt(p.X, p.Y).ClearArrow(o)
}

func (b *CircuitBoard) Draw(c Canvas, r *CircuitBoardRenderer) {
	h := b.height
	w := b.width
	_, _ = w, h

	// Draw the background
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c.Draw(r.Background(x, y))
		}
	}

	// Draw the arrows
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			chip := b.ChipAt(x, y)
			if o, ok := chip.ArrowYes(); ok {
				if chip.IsTest() {
					c.Draw(r.ArrowYes(x, y, o))
				} else {
					c.Draw(r.Arrow(x, y, o))
				}
			}
			if o, ok := chip.ArrowNo(); ok && chip.IsTest() {
				c.Draw(r.ArrowNo(x, y, o))
			}
		}
	}

	// Draw the chips
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			chip := b.ChipAt(x, y)
			if chip.Type() != NoChip {
				c.Draw(r.Chip(chip.Type(), x, y))
			}
		}
	}
}

var chipTypeMap = map[string]ChipType{
	"ST": StartChip,
	"W?": IsWallAheadChip,
	"B?": IsFloorBlueChip,
	"R?": IsFloorRedChip,
	"Y?": IsFloorYellowChip,
	"MF": ForwardChip,
	"TL": TurnLeftChip,
	"TR": TurnRightChip,
	"PB": PaintBlueChip,
	"PR": PaintRedChip,
	"PY": PaintYellowChip,
	"..": NoChip,
	"  ": NoChip,
}