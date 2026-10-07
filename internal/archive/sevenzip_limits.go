package archive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

var (
	errSevenZipHeader        = errors.New("invalid 7z header")
	errSevenZipDecoderMemory = errors.New("7z decoder memory exceeds the extraction cap")
)

const (
	sevenZipMaxFolders  = uint64(DefaultMaxEntries)
	sevenZipMaxCoders   = 64
	sevenZipMaxStreams  = 64
	sevenZipMaxProperty = 16 << 20
)

func checkSevenZipLimits(r io.ReaderAt, size int64) error {
	if size < 32 {
		return errSevenZipHeader
	}

	var start [32]byte
	if _, err := io.ReadFull(io.NewSectionReader(r, 0, int64(len(start))), start[:]); err != nil {
		return errSevenZipHeader
	}
	nextOffset := binary.LittleEndian.Uint64(start[12:20])
	nextSize := binary.LittleEndian.Uint64(start[20:28])
	if nextOffset > uint64(size-32) || nextSize > uint64(size-32)-nextOffset {
		return errSevenZipHeader
	}
	offset, err := sevenZipInt64(nextOffset)
	if err != nil {
		return err
	}
	length, err := sevenZipInt64(nextSize)
	if err != nil {
		return err
	}

	c := sevenZipCursor{r: io.NewSectionReader(r, 32+offset, length)}
	return parseSevenZipHeader(&c)
}

func sevenZipInt64(n uint64) (int64, error) {
	if n > ^uint64(0)>>1 {
		return 0, errSevenZipHeader
	}
	return int64(n), nil
}

type sevenZipCursor struct {
	r io.Reader
}

func (c *sevenZipCursor) byte() (byte, error) {
	var b [1]byte
	if _, err := io.ReadFull(c.r, b[:]); err != nil {
		return 0, errSevenZipHeader
	}
	return b[0], nil
}

func (c *sevenZipCursor) uint() (uint64, error) {
	first, err := c.byte()
	if err != nil {
		return 0, err
	}
	mask := byte(0x80)
	extra := 0
	for extra < 8 && first&mask != 0 {
		extra++
		mask >>= 1
	}
	var value uint64
	if extra < 8 {
		value = uint64(first & (mask - 1))
	}
	for i := 0; i < extra; i++ {
		b, err := c.byte()
		if err != nil {
			return 0, err
		}
		value = value<<8 | uint64(b)
	}
	return value, nil
}

func (c *sevenZipCursor) skip(n uint64) error {
	if n > sevenZipMaxProperty {
		return errSevenZipHeader
	}
	if _, err := io.CopyN(io.Discard, c.r, int64(n)); err != nil {
		return errSevenZipHeader
	}
	return nil
}

func parseSevenZipHeader(c *sevenZipCursor) error {
	id, err := c.byte()
	if err != nil {
		return err
	}
	switch id {
	case 0x17:
		return parseSevenZipStreamsInfo(c)
	case 0x01:
		return parseSevenZipMainHeader(c)
	default:
		return errSevenZipHeader
	}
}

func parseSevenZipMainHeader(c *sevenZipCursor) error {
	for {
		id, err := c.byte()
		if err != nil {
			return err
		}
		switch id {
		case 0:
			return nil
		case 0x02:
			if err := skipSevenZipProperties(c); err != nil {
				return err
			}
		case 0x03, 0x04:
			if err := parseSevenZipStreamsInfo(c); err != nil {
				return err
			}
		case 0x05:
			if err := skipSevenZipFilesInfo(c); err != nil {
				return err
			}
		default:
			return errSevenZipHeader
		}
	}
}

func skipSevenZipProperties(c *sevenZipCursor) error {
	for {
		id, err := c.byte()
		if err != nil {
			return err
		}
		if id == 0 {
			return nil
		}
		size, err := c.uint()
		if err != nil {
			return err
		}
		if err := c.skip(size); err != nil {
			return err
		}
	}
}

func skipSevenZipFilesInfo(c *sevenZipCursor) error {
	n, err := c.uint()
	if err != nil || n > sevenZipMaxFolders {
		return errSevenZipHeader
	}
	return skipSevenZipProperties(c)
}

func parseSevenZipStreamsInfo(c *sevenZipCursor) error {
	for {
		id, err := c.byte()
		if err != nil {
			return err
		}
		switch id {
		case 0:
			return nil
		case 0x06:
			if err := parseSevenZipPackInfo(c); err != nil {
				return err
			}
		case 0x07:
			folders, err := parseSevenZipUnpackInfo(c)
			if err != nil {
				return err
			}
			if err := parseSevenZipStreamTail(c, folders); err != nil {
				return err
			}
			return nil
		case 0x08:
			return errSevenZipHeader
		default:
			return errSevenZipHeader
		}
	}
}

func parseSevenZipPackInfo(c *sevenZipCursor) error {
	if _, err := c.uint(); err != nil {
		return err
	}
	n, err := c.uint()
	if err != nil || n > sevenZipMaxFolders {
		return errSevenZipHeader
	}
	for {
		id, err := c.byte()
		if err != nil {
			return err
		}
		switch id {
		case 0:
			return nil
		case 0x09:
			for range n {
				if _, err := c.uint(); err != nil {
					return err
				}
			}
		case 0x0A:
			if err := parseSevenZipCRCs(c, n); err != nil {
				return err
			}
		default:
			return errSevenZipHeader
		}
	}
}

type sevenZipFolder struct {
	inStreams  uint64
	outStreams uint64
}

func parseSevenZipUnpackInfo(c *sevenZipCursor) ([]sevenZipFolder, error) {
	id, err := c.byte()
	if err != nil || id != 0x0B {
		return nil, errSevenZipHeader
	}
	n, err := c.uint()
	if err != nil || n == 0 || n > sevenZipMaxFolders {
		return nil, errSevenZipHeader
	}
	external, err := c.byte()
	if err != nil || external != 0 {
		return nil, errSevenZipHeader
	}
	folders := make([]sevenZipFolder, 0, n)
	for range n {
		folder, err := parseSevenZipFolder(c)
		if err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}
	for {
		id, err := c.byte()
		if err != nil {
			return nil, err
		}
		switch id {
		case 0:
			return folders, nil
		case 0x0C:
			for _, folder := range folders {
				for i := uint64(0); i < folder.outStreams; i++ {
					if _, err := c.uint(); err != nil {
						return nil, err
					}
				}
			}
		case 0x0A:
			if err := parseSevenZipCRCs(c, n); err != nil {
				return nil, err
			}
		default:
			return nil, errSevenZipHeader
		}
	}
}

func parseSevenZipFolder(c *sevenZipCursor) (sevenZipFolder, error) {
	n, err := c.uint()
	if err != nil || n == 0 || n > sevenZipMaxCoders {
		return sevenZipFolder{}, errSevenZipHeader
	}
	var folder sevenZipFolder
	for range n {
		main, err := c.byte()
		if err != nil {
			return sevenZipFolder{}, err
		}
		idSize := int(main & 0x0F)
		if idSize == 0 || idSize > 16 {
			return sevenZipFolder{}, errSevenZipHeader
		}
		var method [16]byte
		if _, err := io.ReadFull(c.r, method[:idSize]); err != nil {
			return sevenZipFolder{}, errSevenZipHeader
		}
		inStreams, outStreams := uint64(1), uint64(1)
		if main&0x10 != 0 {
			inStreams, err = c.uint()
			if err != nil {
				return sevenZipFolder{}, err
			}
			outStreams, err = c.uint()
			if err != nil {
				return sevenZipFolder{}, err
			}
			if inStreams == 0 || outStreams == 0 || inStreams > sevenZipMaxStreams || outStreams > sevenZipMaxStreams {
				return sevenZipFolder{}, errSevenZipHeader
			}
		}
		folder.inStreams += inStreams
		folder.outStreams += outStreams
		if main&0x20 == 0 {
			continue
		}
		propsSize, err := c.uint()
		if err != nil {
			return sevenZipFolder{}, err
		}
		if propsSize <= 8 {
			props := make([]byte, propsSize)
			if _, err := io.ReadFull(c.r, props); err != nil {
				return sevenZipFolder{}, errSevenZipHeader
			}
			if err := checkSevenZipCoder(method[:idSize], propsSize, props); err != nil {
				return sevenZipFolder{}, err
			}
		} else {
			if err := c.skip(propsSize); err != nil {
				return sevenZipFolder{}, err
			}
			if err := checkSevenZipCoder(method[:idSize], propsSize, nil); err != nil {
				return sevenZipFolder{}, err
			}
		}
	}
	bindPairs := folder.outStreams - 1
	for range bindPairs {
		if _, err := c.uint(); err != nil {
			return sevenZipFolder{}, err
		}
		if _, err := c.uint(); err != nil {
			return sevenZipFolder{}, err
		}
	}
	if folder.inStreams < bindPairs {
		return sevenZipFolder{}, errSevenZipHeader
	}
	packedStreams := folder.inStreams - bindPairs
	if packedStreams == 0 || packedStreams > sevenZipMaxStreams {
		return sevenZipFolder{}, errSevenZipHeader
	}
	if packedStreams != 1 {
		for range packedStreams {
			if _, err := c.uint(); err != nil {
				return sevenZipFolder{}, err
			}
		}
	}
	return folder, nil
}

func checkSevenZipCoder(method []byte, propsSize uint64, props []byte) error {
	switch {
	case bytes.Equal(method, []byte{0x21}):
		if propsSize != 1 {
			return errSevenZipHeader
		}
		p := props[0]
		if p > 40 {
			return errSevenZipHeader
		}
		dictCap := uint64(2|p&1) << (uint(p)/2 + 11)
		if dictCap > uint64(DefaultMaxEntryBytes) {
			return errSevenZipDecoderMemory
		}
	case bytes.Equal(method, []byte{0x03, 0x01, 0x01}), bytes.Equal(method, []byte{0x03, 0x04, 0x01}):
		if propsSize != 5 {
			return errSevenZipHeader
		}
		if uint64(binary.LittleEndian.Uint32(props[1:])) > uint64(DefaultMaxEntryBytes) {
			return errSevenZipDecoderMemory
		}
	}
	return nil
}

func parseSevenZipStreamTail(c *sevenZipCursor, folders []sevenZipFolder) error {
	for {
		id, err := c.byte()
		if err != nil {
			return err
		}
		switch id {
		case 0:
			return nil
		case 0x08:
			return skipSevenZipSubStreams(c, folders)
		default:
			return errSevenZipHeader
		}
	}
}

func skipSevenZipSubStreams(c *sevenZipCursor, folders []sevenZipFolder) error {
	counts := make([]uint64, len(folders))
	for i := range counts {
		counts[i] = 1
	}
	for {
		id, err := c.byte()
		if err != nil {
			return err
		}
		switch id {
		case 0:
			return nil
		case 0x0D:
			var total uint64
			for i := range counts {
				counts[i], err = c.uint()
				if err != nil || counts[i] == 0 {
					return errSevenZipHeader
				}
				total += counts[i]
			}
			if total > sevenZipMaxFolders {
				return errSevenZipHeader
			}
		case 0x09:
			for _, count := range counts {
				for range count - 1 {
					if _, err := c.uint(); err != nil {
						return err
					}
				}
			}
		case 0x0A:
			var total uint64
			for _, count := range counts {
				total += count
			}
			if err := parseSevenZipCRCs(c, total); err != nil {
				return err
			}
		default:
			return errSevenZipHeader
		}
	}
}

func parseSevenZipCRCs(c *sevenZipCursor, n uint64) error {
	allDefined, err := c.byte()
	if err != nil {
		return err
	}
	if allDefined != 0 {
		for range n {
			var crc [4]byte
			if _, err := io.ReadFull(c.r, crc[:]); err != nil {
				return errSevenZipHeader
			}
		}
		return nil
	}
	defined := make([]byte, (n+7)/8)
	for i := range defined {
		var err error
		defined[i], err = c.byte()
		if err != nil {
			return errSevenZipHeader
		}
	}
	for i := range n {
		if defined[i/8]&(1<<uint(i%8)) == 0 {
			continue
		}
		var crc [4]byte
		if _, err := io.ReadFull(c.r, crc[:]); err != nil {
			return errSevenZipHeader
		}
	}
	return nil
}
