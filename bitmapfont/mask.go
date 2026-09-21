package bitmapfont

import "golang.org/x/image/font/basicfont"

func fontMaskAt(x, y int) (uint32, uint32, uint32, uint32) {
	return basicfont.Face7x13.Mask.At(x, y).RGBA()
}
