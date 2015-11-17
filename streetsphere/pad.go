package streetsphere

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
)

type XMPMeta struct {
	XMLName  xml.Name  `xml:"xmpmeta"`
	PanoOpts *PanoOpts `xml:"RDF>Description"`
}

type PanoOpts struct {
	TotalHeight int     `xml:"FullPanoHeightPixels,attr"`
	TotalWidth  int     `xml:"FullPanoWidthPixels,attr"`
	Top         int     `xml:"CroppedAreaTopPixels,attr"`
	Left        int     `xml:"CroppedAreaLeftPixels,attr"`
	Height      int     `xml:"CroppedAreaImageHeightPixels,attr"`
	Width       int     `xml:"CroppedAreaImageWidthPixels,attr"`
	Heading     float64 `xml:"PoseHeadingDegrees,attr"`
	NS          string  `xml:"GPano,attr"`

	// Element instead of attribute.
	ETotalHeight int     `xml:"FullPanoHeightPixels"`
	ETotalWidth  int     `xml:"FullPanoWidthPixels"`
	ETop         int     `xml:"CroppedAreaTopPixels"`
	ELeft        int     `xml:"CroppedAreaLeftPixels"`
	EHeight      int     `xml:"CroppedAreaImageHeightPixels"`
	EWidth       int     `xml:"CroppedAreaImageWidthPixels"`
	EHeading     float64 `xml:"PoseHeadingDegrees"`
}

func (p *PanoOpts) Normalize() {
	if p.TotalHeight == 0 {
		p.TotalHeight = p.ETotalHeight
	}
	if p.TotalWidth == 0 {
		p.TotalWidth = p.ETotalWidth
	}
	if p.Top == 0 {
		p.Top = p.ETop
	}
	if p.Left == 0 {
		p.Left = p.ELeft
	}
	if p.Height == 0 {
		p.Height = p.EHeight
	}
	if p.Heading == 0 {
		p.Heading = p.EHeading
	}
}

// Pad reads a photosphere image and writes a padded 360 degree x 180 degree image to a given writer.
func Pad(w io.Writer, ir io.Reader) (pano *PanoOpts, err error) {
	d, err := ioutil.ReadAll(ir)
	if err != nil {
		return nil, err
	}

	r := bufio.NewReader(bytes.NewReader(d))
	for {
		s, err := NextSection(r, APP1)
		if err != nil {
			return nil, err
		}
		if s == nil {
			break
		}

		if IsXMP(s) {
			xmp := ExtractXMP(s)
			meta := new(XMPMeta)
			err := xml.Unmarshal(xmp, meta)
			if err != nil {
				// Not XMP?
				continue
			}
			if meta.PanoOpts.NS != "http://ns.google.com/photos/1.0/panorama/" {
				// Different XMP
				continue
			}
			pano = meta.PanoOpts
			break
		}
	}

	if pano == nil {
		return nil, errors.New("image provided had no photo sphere metadata")
	}

	pano.Normalize()

	src, err := jpeg.Decode(bytes.NewReader(d))
	if err != nil {
		return nil, err
	}
	srcB := src.Bounds()

	// Sometimes the height in the metadata doesn't match the actual height.
	// If that's the case, scale all of the other parameters.
	if srcB.Dy() != pano.Height {
		scale := float64(srcB.Dy()) / float64(pano.Height)
		pano.TotalHeight = int(float64(pano.TotalHeight) * scale)
		pano.TotalWidth = int(float64(pano.TotalWidth) * scale)
		pano.Top = int(float64(pano.Top) * scale)
		pano.Left = int(float64(pano.Left) * scale)
	}

	dst := image.NewRGBA(image.Rect(0, 0, pano.TotalWidth, pano.TotalHeight))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.Black), image.ZP, draw.Src)
	compositeBounds := image.Rect(pano.Left, pano.Top, pano.Left+srcB.Dx(), pano.Top+srcB.Dy())
	draw.Draw(dst, compositeBounds, src, image.ZP, draw.Src)

	err = jpeg.Encode(w, dst, nil)
	return pano, err
}

var xmpSentinel = []byte("http://ns.adobe.com/xap/1.0/\x00")

func IsXMP(s *Section) bool {
	return bytes.HasPrefix(s.Data, xmpSentinel)
}

func ExtractXMP(s *Section) []byte {
	return s.Data[len(xmpSentinel):]
}
