package streetsphere

import (
	"bufio"
	"bytes"
	"encoding/xml"
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
			meta := new(XMPMeta)
			err := xml.Unmarshal(ExtractXMP(s), meta)
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
