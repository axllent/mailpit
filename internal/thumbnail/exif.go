package thumbnail

import (
	"bytes"
	"encoding/binary"
)

const (
	exifOrientationTag = 0x0112
	tiffTypeShort      = 3
	ifdEntrySize       = 12

	jpegMarkerPrefix = 0xFF
	jpegSOI          = 0xD8 // start of image
	jpegEOI          = 0xD9 // end of image
	jpegSOS          = 0xDA // start of scan (entropy-coded data follows)
	jpegAPP1         = 0xE1
	jpegTEM          = 0x01 // standalone, no length
	jpegRSTMin       = 0xD0 // RSTn markers are standalone, no length
	jpegRSTMax       = 0xD7
)

// exifHeader precedes the TIFF block in an APP1 Exif segment.
var exifHeader = []byte("Exif\x00\x00")

// readOrientation returns the EXIF orientation (1-8) of JPEG data, or 0 if
// the data is not a JPEG, has no orientation tag, or is malformed.
//
// This is a deliberately minimal reader: it only inspects IFD0 of the first
// EXIF APP1 segment and never follows next-IFD or sub-IFD pointers. The work
// done is therefore bounded by the segment size (max 64 KiB) regardless of
// file content. Any anomaly results in 0 (no rotation).
//
// The orientation entry must be type SHORT with a count of 1, as written by
// cameras and editors. Other encodings (e.g. LONG) are deliberately treated
// as invalid, which is stricter than goexif. Only the first orientation
// entry is considered: if it is malformed the result is 0 and later
// duplicates are not consulted, as they would be ambiguous.
func readOrientation(data []byte) int {
	seg := findExifSegment(data)
	if len(seg) < 8 {
		return 0
	}

	var order binary.ByteOrder
	switch string(seg[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	if order.Uint16(seg[2:4]) != 42 {
		return 0
	}

	ifd := int64(order.Uint32(seg[4:8]))
	if ifd+2 > int64(len(seg)) {
		return 0
	}
	count := int64(order.Uint16(seg[ifd : ifd+2]))

	for i := int64(0); i < count; i++ {
		start := ifd + 2 + i*ifdEntrySize
		if start+ifdEntrySize > int64(len(seg)) {
			return 0
		}
		e := seg[start : start+ifdEntrySize]
		if order.Uint16(e[0:2]) != exifOrientationTag {
			continue
		}
		// type SHORT, count 1, value stored inline
		if order.Uint16(e[2:4]) != tiffTypeShort || order.Uint32(e[4:8]) != 1 {
			return 0
		}
		o := int(order.Uint16(e[8:10]))
		if o < 1 || o > 8 {
			return 0
		}
		return o
	}

	return 0
}

// findExifSegment returns the TIFF payload of the first APP1 "Exif"
// segment of a JPEG, or nil if there isn't one before the image data.
// The returned slice aliases data and must be treated as read-only.
func findExifSegment(data []byte) []byte {
	if len(data) < 4 || data[0] != jpegMarkerPrefix || data[1] != jpegSOI {
		return nil
	}

	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != jpegMarkerPrefix {
			return nil
		}
		marker := data[pos+1]
		switch {
		case marker == jpegMarkerPrefix: // fill byte
			pos++
			continue
		case marker == jpegTEM || (marker >= jpegRSTMin && marker <= jpegRSTMax): // standalone
			pos += 2
			continue
		case marker == jpegEOI || marker == jpegSOS: // no metadata can follow
			return nil
		}

		// The length includes its own two bytes, so anything below 2 is
		// corrupt. Fail closed rather than guessing where the next segment is.
		size := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if size < 2 || pos+2+size > len(data) {
			return nil
		}
		payload := data[pos+4 : pos+2+size]
		if marker == jpegAPP1 && bytes.HasPrefix(payload, exifHeader) {
			return payload[len(exifHeader):]
		}
		pos += 2 + size
	}

	return nil
}
