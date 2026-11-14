package thumbnail

// isProgressiveJPEG reports whether JPEG data is progressive (or any other multi-scan,
// coefficient-buffered encoding), by finding the start-of-frame marker. Data that is not a JPEG,
// or has no frame header before the image data, is not progressive.
//
// A progressive JPEG keeps every DCT coefficient in memory until the last scan has been read,
// which takes about 14 bytes per pixel, against about 1.4 for a baseline JPEG.
func isProgressiveJPEG(data []byte) bool {
	if len(data) < 4 || data[0] != jpegMarkerPrefix || data[1] != jpegSOI {
		return false
	}

	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != jpegMarkerPrefix {
			return false
		}

		marker := data[pos+1]

		switch {
		case marker == jpegMarkerPrefix: // fill byte
			pos++
			continue
		case marker == jpegTEM || (marker >= jpegRSTMin && marker <= jpegRSTMax): // standalone
			pos += 2
			continue
		case marker == jpegEOI || marker == jpegSOS: // the frame header is always before the image data
			return false
		case marker == jpegSOF2 || marker == jpegSOF6 || marker == jpegSOF10 || marker == jpegSOF14:
			return true
		case marker >= jpegSOF0 && marker <= jpegSOF15 && marker != jpegDHT && marker != jpegJPG && marker != jpegDAC:
			// a sequential frame header
			return false
		}

		size := int(data[pos+2])<<8 | int(data[pos+3])
		if size < 2 || pos+2+size > len(data) {
			return false
		}

		pos += 2 + size
	}

	return false
}
