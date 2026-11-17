package thumbnail

// isProgressiveJPEG reports whether JPEG data may be progressive, so that it is held to the lower pixel limit. It is
// false only when the frame header which Go's decoder (image/jpeg) would read is a sequential one (SOF0 or SOF1, the
// only sequential ones it decodes), and true for anything else, including data which can not be followed: the
// lower limit is the safe answer, and the decoder refuses such data anyway.
//
// It reads the markers as the decoder does, so the two can not be made to disagree. The decoder skips what is not a
// marker (bytes before an 0xFF, an 0xFF 0x00 pair, fill bytes and restart markers), and this once returned false
// ("not progressive") on any of those, which let a progressive JPEG past the lower limit with a single byte added.
//
// A progressive JPEG keeps every DCT coefficient in memory until the last scan has been read,
// which takes 7.5 bytes per pixel (15 with full colour resolution, 4:4:4), against 1.5 (3) for a
// baseline JPEG, which is about five times as much.
func isProgressiveJPEG(data []byte) bool {
	if len(data) < 2 || data[0] != jpegMarkerPrefix || data[1] != jpegSOI {
		return true
	}

	pos := 2
	for {
		// as the decoder does, whatever is before the next 0xFF is skipped
		for pos < len(data) && data[pos] != jpegMarkerPrefix {
			pos++
		}
		if pos+1 >= len(data) {
			return true
		}

		marker := data[pos+1]
		pos += 2

		if marker == 0x00 {
			// 0xFF 0x00 is skipped as well
			continue
		}

		// fill bytes
		for marker == jpegMarkerPrefix {
			if pos >= len(data) {
				return true
			}
			marker = data[pos]
			pos++
		}

		if marker >= jpegRSTMin && marker <= jpegRSTMax {
			// no length
			continue
		}

		switch marker {
		case jpegSOF0, jpegSOF1:
			return false
		case jpegSOF2:
			return true
		case jpegEOI, jpegSOS:
			// no frame header before the image data, which the decoder refuses
			return true
		}

		if pos+2 > len(data) {
			return true
		}
		size := int(data[pos])<<8 | int(data[pos+1])
		if size < 2 {
			return true
		}

		switch {
		case marker == jpegDHT, marker == jpegDQT, marker == jpegDRI, marker == jpegCOM,
			marker >= jpegAPP0 && marker <= jpegAPP15:
			// a segment which the decoder reads past
			pos += size
		default:
			// the decoder refuses any other marker (other frame header types are not supported)
			return true
		}
	}
}
