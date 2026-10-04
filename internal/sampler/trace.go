package sampler

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const (
	blockMetadata uint32 = iota
	blockEvent
	blockStack
	blockSP
)

// Frame is a resolved instruction in a sampled stack. The first frame is the
// innermost frame.
type Frame struct {
	IP       uint64
	Method   string
	Type     string
	Assembly string
}

// Sample is one main-thread or worker-thread sample. Timestamp is normalized
// to nanoseconds so callers do not need the trace's counter frequency.
type Sample struct {
	Timestamp uint64
	ThreadID  uint32
	Frames    []Frame
}

// Method is a method-load record used to resolve sampled instruction
// addresses.
type Method struct {
	StartAddress uint64
	Size         uint64
	Namespace    string
	Name         string
	ModuleID     uint64
	Assembly     string
}

// Trace contains the useful subset of a nettrace stream.
type Trace struct {
	TimestampFrequency uint64
	Samples            []Sample
	Methods            []Method
	Assemblies         map[uint64]string
	Modules            map[uint64]uint64
	PointerSize        int
}

type eventField struct {
	name string
	code uint32
}

type eventMetadata struct {
	provider string
	name     string
	eventID  uint32
	fields   []eventField
}

type rawEvent struct {
	timestamp uint64
	threadID  uint32
	metadata  uint32
	stackID   uint64
	payload   []byte
}

type stackRecord struct {
	id     uint64
	frames []uint64
}

// Parse reads a V5 EventPipe nettrace stream.
func Parse(r io.Reader) (*Trace, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(data) < 8 || string(data[:8]) != "Nettrace" {
		return nil, errors.New("sampler: not a nettrace stream")
	}

	trace := &Trace{
		Assemblies: make(map[uint64]string),
		Modules:    make(map[uint64]uint64),
	}
	trace.TimestampFrequency = headerFrequency(data)
	metadata := make(map[uint32]eventMetadata)
	var events []rawEvent
	var stacks []stackRecord
	if !parseV5(data, trace, metadata, &events, &stacks) {
		scanBlocks(data, metadata, &events, &stacks)
	}
	if len(events) == 0 && len(stacks) == 0 {
		return nil, errors.New("sampler: nettrace has no event or stack blocks")
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].timestamp < events[j].timestamp
	})
	stackMap := make(map[uint64][]uint64, len(stacks))
	for _, stack := range stacks {
		stackMap[stack.id] = append([]uint64(nil), stack.frames...)
	}
	for _, event := range events {
		name := metadata[event.metadata].name
		provider := metadata[event.metadata].provider
		if isSampleEvent(provider, name, event.payload) {
			frames := make([]Frame, 0)
			for _, ip := range stackMap[event.stackID] {
				frames = append(frames, Frame{IP: ip})
			}
			trace.Samples = append(trace.Samples, Sample{
				Timestamp: normalizeTimestamp(event.timestamp, trace.TimestampFrequency),
				ThreadID:  event.threadID,
				Frames:    frames,
			})
			continue
		}
		decodeRuntimeEvent(trace, provider, name, metadata[event.metadata].eventID, metadata[event.metadata].fields, event.payload)
	}
	resolveMethods(trace)
	return trace, nil
}

// ParseFile parses a nettrace file produced by the sampler session.
func ParseFile(path string) (*Trace, error) {
	data, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(bytes.NewReader(data))
}

func headerFrequency(data []byte) uint64 {
	for _, offset := range []int{8, 12, 16, 20, 24, 28, 32, 36, 40, 48, 56} {
		if offset+8 <= len(data) {
			value := binary.LittleEndian.Uint64(data[offset : offset+8])
			if value >= 1_000 && value <= 10_000_000_000 {
				return value
			}
		}
	}
	return 1_000_000_000
}

func normalizeTimestamp(value, frequency uint64) uint64 {
	if frequency == 0 || frequency == 1_000_000_000 {
		return value
	}
	return uint64((float64(value) * 1_000_000_000) / float64(frequency))
}

type v5Block struct {
	kind       string
	dataStart  int
	blockStart int
	blockEnd   int
}

func parseV5(data []byte, trace *Trace, metadata map[uint32]eventMetadata, events *[]rawEvent, stacks *[]stackRecord) bool {
	if len(data) < 32 || string(data[12:32]) != "!FastSerialization.1" {
		return false
	}
	blocks := make([]v5Block, 0)
	for offset := 32; offset+15 < len(data); {
		name, typeEnd, ok := v5ObjectType(data, offset)
		if !ok {
			offset++
			continue
		}
		if name == "Trace" {
			parseTraceFields(data[typeEnd+1:], trace)
			offset = typeEnd + 1
			continue
		}
		if name != "EventBlock" && name != "MetadataBlock" && name != "StackBlock" && name != "SPBlock" {
			offset = typeEnd + 1
			continue
		}
		block, ok := v5BlockBounds(data, typeEnd+1, name)
		if !ok {
			offset = typeEnd + 1
			continue
		}
		blocks = append(blocks, block)
		offset = block.blockEnd
		if offset < len(data) && data[offset] == 6 {
			offset++
		}
	}
	if len(blocks) == 0 {
		return false
	}
	for _, block := range blocks {
		switch block.kind {
		case "MetadataBlock":
			parseV5EventBlock(data[block.blockStart:block.blockEnd], metadata, true)
		case "StackBlock":
			*stacks = append(*stacks, parseV5StackBlock(data[block.blockStart:block.blockEnd], trace.PointerSize)...)
		}
	}
	for _, block := range blocks {
		if block.kind == "EventBlock" {
			*events = append(*events, parseV5EventBlock(data[block.blockStart:block.blockEnd], metadata, false)...)
		}
	}
	return true
}

func v5ObjectType(data []byte, offset int) (string, int, bool) {
	if offset+15 > len(data) || data[offset] != 5 || data[offset+1] != 5 || data[offset+2] != 1 {
		return "", 0, false
	}
	nameLength := int(binary.LittleEndian.Uint32(data[offset+11 : offset+15]))
	nameStart := offset + 15
	nameEnd := nameStart + nameLength
	if nameLength < 0 || nameEnd >= len(data) || data[nameEnd] != 6 {
		return "", 0, false
	}
	name := string(data[nameStart:nameEnd])
	switch name {
	case "Trace", "EventBlock", "MetadataBlock", "StackBlock", "SPBlock":
		return name, nameEnd, true
	default:
		return "", 0, false
	}
}

func v5BlockBounds(data []byte, dataStart int, kind string) (v5Block, bool) {
	if dataStart+4 > len(data) {
		return v5Block{}, false
	}
	blockStart := align4(dataStart + 4)
	if blockStart+4 > len(data) {
		return v5Block{}, false
	}
	blockSize := int(binary.LittleEndian.Uint32(data[dataStart : dataStart+4]))
	blockEnd := blockStart + blockSize
	if blockSize < 8 || blockEnd > len(data) {
		return v5Block{}, false
	}
	return v5Block{kind: kind, dataStart: dataStart, blockStart: blockStart, blockEnd: blockEnd}, true
}

func parseTraceFields(data []byte, trace *Trace) {
	if len(data) < 18+8+8+4 {
		return
	}
	cursor := 18
	cursor += 8
	trace.TimestampFrequency = binary.LittleEndian.Uint64(data[cursor : cursor+8])
	cursor += 8
	trace.PointerSize = int(binary.LittleEndian.Uint32(data[cursor : cursor+4]))
}

func parseV5EventBlock(block []byte, metadata map[uint32]eventMetadata, metadataBlock bool) []rawEvent {
	if len(block) < 20 {
		return nil
	}
	headerSize := int(binary.LittleEndian.Uint16(block[:2]))
	blockFlags := binary.LittleEndian.Uint16(block[2:4])
	if headerSize < 20 || headerSize > len(block) {
		return nil
	}
	compressed := blockFlags&1 != 0
	return parseV5Events(block[headerSize:], metadata, metadataBlock, compressed)
}

type v5EventState struct {
	metadataID    uint32
	sequence      uint64
	captureThread uint64
	processor     uint64
	thread        uint64
	stack         uint64
	timestamp     uint64
	payloadSize   uint64
}

func parseV5Events(data []byte, metadata map[uint32]eventMetadata, metadataBlock, compressed bool) []rawEvent {
	var out []rawEvent
	state := v5EventState{}
	if compressed {
		for offset := 0; offset < len(data); {
			event, used, ok := parseCompressedV5Event(data[offset:], &state)
			if !ok || used <= 0 {
				break
			}
			if metadataBlock {
				parseV5Metadata(event.payload, metadata)
			} else if event.metadata != 0 {
				out = append(out, event)
			}
			offset += used
		}
		return out
	}
	for offset := 0; offset+4 <= len(data); {
		eventSize := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		if eventSize <= 0 || offset+4+eventSize > len(data) {
			break
		}
		event, ok := parseUncompressedV5Event(data[offset+4:offset+4+eventSize], &state)
		if ok {
			if metadataBlock {
				parseV5Metadata(event.payload, metadata)
			} else if event.metadata != 0 {
				out = append(out, event)
			}
		}
		offset += 4 + eventSize
	}
	return out
}

func parseCompressedV5Event(body []byte, state *v5EventState) (rawEvent, int, bool) {
	if len(body) < 1 {
		return rawEvent{}, 0, false
	}
	flags := body[0]
	offset := 1
	read := func() (uint64, bool) {
		value, used := readVarint(body[offset:])
		if used == 0 {
			return 0, false
		}
		offset += used
		return value, true
	}
	if flags&1 != 0 {
		value, ok := read()
		if !ok {
			return rawEvent{}, 0, false
		}
		state.metadataID = uint32FromUint64(value)
	}
	if flags&2 != 0 {
		value, ok := read()
		if !ok {
			return rawEvent{}, 0, false
		}
		state.sequence += value
		value, ok = read()
		if !ok {
			return rawEvent{}, 0, false
		}
		state.captureThread = value
		value, ok = read()
		if !ok {
			return rawEvent{}, 0, false
		}
		state.processor = value
	}
	if flags&4 != 0 {
		value, ok := read()
		if !ok {
			return rawEvent{}, 0, false
		}
		state.thread = value
	}
	if flags&8 != 0 {
		value, ok := read()
		if !ok {
			return rawEvent{}, 0, false
		}
		state.stack = value
	}
	value, ok := read()
	if !ok {
		return rawEvent{}, 0, false
	}
	state.timestamp += value
	if flags&16 != 0 {
		if offset+16 > len(body) {
			return rawEvent{}, 0, false
		}
		offset += 16
	}
	if flags&32 != 0 {
		if offset+16 > len(body) {
			return rawEvent{}, 0, false
		}
		offset += 16
	}
	if flags&128 != 0 {
		value, ok = read()
		if !ok {
			return rawEvent{}, 0, false
		}
		state.payloadSize = value
	}
	if state.payloadSize > uint64FromInt(len(body)-offset) {
		return rawEvent{}, 0, false
	}
	payloadEnd := offset + intFromUint64(state.payloadSize)
	payload := body[offset:payloadEnd]
	return rawEvent{timestamp: state.timestamp, threadID: uint32FromUint64(state.thread), metadata: state.metadataID, stackID: state.stack, payload: payload}, payloadEnd, true
}

func parseUncompressedV5Event(body []byte, state *v5EventState) (rawEvent, bool) {
	if len(body) < 4+4+8+8+4+4+8+16+16+4 {
		return rawEvent{}, false
	}
	offset := 0
	metadataID := binary.LittleEndian.Uint32(body[offset:])
	offset += 4
	offset += 4
	thread := binary.LittleEndian.Uint64(body[offset:])
	offset += 8
	offset += 8
	offset += 4
	stack := uint64(binary.LittleEndian.Uint32(body[offset:]))
	offset += 4
	timestamp := binary.LittleEndian.Uint64(body[offset:])
	offset += 8
	offset += 16 + 16
	payloadSize := int(binary.LittleEndian.Uint32(body[offset:]))
	offset += 4
	if payloadSize < 0 || offset+payloadSize > len(body) {
		return rawEvent{}, false
	}
	state.metadataID = metadataID
	state.thread = thread
	state.stack = stack
	state.timestamp = timestamp
	return rawEvent{timestamp: timestamp, threadID: uint32FromUint64(thread), metadata: metadataID, stackID: stack, payload: body[offset : offset+payloadSize]}, true
}

func parseV5Metadata(payload []byte, metadata map[uint32]eventMetadata) {
	cursor := 0
	metadataID, ok := readUint32(payload, &cursor)
	if !ok {
		return
	}
	provider, ok := readWideNull(payload, &cursor)
	if !ok {
		return
	}
	eventID, ok := readUint32(payload, &cursor)
	if !ok {
		return
	}
	name, ok := readWideNull(payload, &cursor)
	if !ok {
		return
	}
	if cursor+8+2*4+4 > len(payload) {
		return
	}
	cursor += 8 + 2*4
	fieldCount, ok := readUint32(payload, &cursor)
	if !ok || fieldCount > 4096 {
		return
	}
	fieldCountInt := intFromUint64(uint64(fieldCount))
	fields := make([]eventField, 0, fieldCountInt)
	for range fieldCountInt {
		code, ok := readUint32(payload, &cursor)
		if !ok {
			return
		}
		fieldName, ok := readWideNull(payload, &cursor)
		if !ok {
			return
		}
		fields = append(fields, eventField{name: fieldName, code: code})
	}
	metadata[metadataID] = eventMetadata{provider: provider, name: name, eventID: eventID, fields: fields}
}

func parseV5StackBlock(block []byte, pointerSize int) []stackRecord {
	if len(block) < 8 {
		return nil
	}
	cursor := 0
	firstID := uint64(binary.LittleEndian.Uint32(block[cursor:]))
	cursor += 4
	count := intFromUint64(uint64(binary.LittleEndian.Uint32(block[cursor:])))
	cursor += 4
	if count < 0 || count > 1_000_000 {
		return nil
	}
	if pointerSize != 4 && pointerSize != 8 {
		pointerSize = 8
	}
	out := make([]stackRecord, 0, count)
	for index := range count {
		if cursor+4 > len(block) {
			return nil
		}
		stackSize := intFromUint64(uint64(binary.LittleEndian.Uint32(block[cursor:])))
		cursor += 4
		if stackSize < 0 || cursor+stackSize > len(block) || stackSize%pointerSize != 0 {
			return nil
		}
		frames := make([]uint64, stackSize/pointerSize)
		for frameIndex := range frames {
			if pointerSize == 4 {
				frames[frameIndex] = uint64(binary.LittleEndian.Uint32(block[cursor:]))
			} else {
				frames[frameIndex] = binary.LittleEndian.Uint64(block[cursor:])
			}
			cursor += pointerSize
		}
		out = append(out, stackRecord{id: firstID + uint64FromInt(index), frames: frames})
	}
	return out
}

func readUint32(data []byte, offset *int) (uint32, bool) {
	if *offset+4 > len(data) {
		return 0, false
	}
	value := binary.LittleEndian.Uint32(data[*offset:])
	*offset += 4
	return value, true
}

func readWideNull(data []byte, offset *int) (string, bool) {
	start := *offset
	for *offset+2 <= len(data) {
		unit := binary.LittleEndian.Uint16(data[*offset:])
		*offset += 2
		if unit == 0 {
			raw := make([]uint16, 0, (*offset-start)/2-1)
			for cursor := start; cursor+2 < *offset; cursor += 2 {
				raw = append(raw, binary.LittleEndian.Uint16(data[cursor:]))
			}
			return string(utf16.Decode(raw)), true
		}
	}
	return "", false
}

func align4(value int) int {
	return (value + 3) &^ 3
}

func scanBlocks(data []byte, metadata map[uint32]eventMetadata, events *[]rawEvent, stacks *[]stackRecord) {
	start := 8
	for start+8 <= len(data) {
		advanced := false
		for _, candidate := range blockHeaders(data, start) {
			if candidate.end <= candidate.payload || candidate.end > len(data) {
				continue
			}
			payload := data[candidate.payload:candidate.end]
			switch candidate.kind {
			case blockMetadata:
				parseMetadataBlock(payload, metadata)
			case blockEvent:
				*events = append(*events, parseEventBlock(payload, metadata)...)
			case blockStack:
				*stacks = append(*stacks, parseStackBlock(payload)...)
			}
			if candidate.end > start {
				start = candidate.end
				advanced = true
				break
			}
		}
		if !advanced {
			start++
		}
	}
}

type blockHeader struct {
	kind    uint32
	payload int
	end     int
}

func blockHeaders(data []byte, offset int) []blockHeader {
	out := make([]blockHeader, 0, 4)
	if offset+8 > len(data) {
		return out
	}
	first, second := binary.LittleEndian.Uint32(data[offset:offset+4]), binary.LittleEndian.Uint32(data[offset+4:offset+8])
	add := func(kind uint32, payload, end int) {
		if kind <= blockSP && end > payload && end <= len(data) {
			out = append(out, blockHeader{kind: kind, payload: payload, end: end})
		}
	}
	if first <= blockSP {
		add(first, offset+8, offset+int(second))
		if offset+16 <= len(data) {
			length := binary.LittleEndian.Uint32(data[offset+8 : offset+12])
			add(first, offset+16, offset+int(length))
		}
	}
	if second <= blockSP {
		add(second, offset+8, offset+int(first))
	}
	return out
}

func parseMetadataBlock(payload []byte, metadata map[uint32]eventMetadata) {
	stringsFound := readableStrings(payload)
	known := make([]string, 0, len(stringsFound))
	for _, found := range stringsFound {
		if isEventName(found.value) || isProviderName(found.value) {
			known = append(known, found.value)
		}
	}
	if len(known) == 0 {
		return
	}

	ids := make([]uint32, 0, len(known))
	for offset := 0; offset+4 <= len(payload); offset += 4 {
		id := binary.LittleEndian.Uint32(payload[offset : offset+4])
		if id > 0 && id < 1_000_000 {
			ids = append(ids, id)
		}
	}
	idIndex := 0
	var provider string
	for _, found := range stringsFound {
		switch {
		case isProviderName(found.value):
			provider = found.value
		case isEventName(found.value):
			id := uint32Length(len(metadata) + 1)
			if idIndex < len(ids) {
				id = ids[idIndex]
				idIndex++
			}
			metadata[id] = eventMetadata{provider: provider, name: found.value}
		}
	}
}

func parseEventBlock(payload []byte, metadata map[uint32]eventMetadata) []rawEvent {
	var out []rawEvent
	seen := make(map[string]bool)
	for offset := 0; offset+16 <= len(payload); offset++ {
		candidates := fixedEventCandidates(payload, offset)
		for _, event := range candidates {
			if event.metadata == 0 || event.timestamp == 0 {
				continue
			}
			if _, ok := metadata[event.metadata]; !ok && !payloadLooksKnown(event.payload) {
				continue
			}
			key := fmt.Sprintf("%d/%d/%d/%d/%x", event.timestamp, event.threadID, event.metadata, event.stackID, event.payload[:minInt(12, len(event.payload))])
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, event)
		}
	}
	if len(out) == 0 {
		out = parseCompressedEvents(payload, metadata)
	}
	return out
}

func fixedEventCandidates(data []byte, offset int) []rawEvent {
	out := make([]rawEvent, 0, 4)
	if offset+24 <= len(data) {
		size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		if size >= 24 && offset+size <= len(data) {
			meta := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
			thread := binary.LittleEndian.Uint32(data[offset+8 : offset+12])
			timestamp := binary.LittleEndian.Uint64(data[offset+12 : offset+20])
			stack := uint64(binary.LittleEndian.Uint32(data[offset+20 : offset+24]))
			out = append(out, rawEvent{timestamp: timestamp, threadID: thread, metadata: meta, stackID: stack, payload: data[offset+24 : offset+size]})
		}
	}
	if offset+28 <= len(data) {
		size := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		if size >= 28 && offset+size <= len(data) {
			meta := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
			timestamp := binary.LittleEndian.Uint64(data[offset+8 : offset+16])
			thread := binary.LittleEndian.Uint32(data[offset+16 : offset+20])
			stack := binary.LittleEndian.Uint64(data[offset+20 : offset+28])
			out = append(out, rawEvent{timestamp: timestamp, threadID: thread, metadata: meta, stackID: stack, payload: data[offset+28 : offset+size]})
		}
	}
	if offset+20 <= len(data) {
		size := int(binary.LittleEndian.Uint16(data[offset : offset+2]))
		if size >= 20 && offset+size <= len(data) {
			meta := uint32(binary.LittleEndian.Uint16(data[offset+2 : offset+4]))
			thread := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
			timestamp := binary.LittleEndian.Uint64(data[offset+8 : offset+16])
			stack := uint64(binary.LittleEndian.Uint32(data[offset+16 : offset+20]))
			out = append(out, rawEvent{timestamp: timestamp, threadID: thread, metadata: meta, stackID: stack, payload: data[offset+20 : offset+size]})
		}
	}
	return out
}

func parseCompressedEvents(data []byte, metadata map[uint32]eventMetadata) []rawEvent {
	var out []rawEvent
	var timestamp uint64
	var thread uint32
	for offset := 0; offset < len(data); {
		size, n := readVarint(data[offset:])
		if n == 0 || size == 0 || size > uint64FromInt(len(data)-offset) {
			offset++
			continue
		}
		record := data[offset : offset+intFromUint64(size)]
		if len(record) < 4 {
			offset++
			continue
		}
		cursor := 1
		metaValue, used := readVarint(record[cursor:])
		if used == 0 {
			offset++
			continue
		}
		cursor += used
		timeDelta, used := readVarint(record[cursor:])
		if used == 0 {
			offset++
			continue
		}
		cursor += used
		threadDelta, used := readVarint(record[cursor:])
		if used == 0 {
			offset++
			continue
		}
		cursor += used
		timestamp += timeDelta
		thread = uint32FromUint64(uint64(thread) + threadDelta)
		stack, used := readVarint(record[cursor:])
		if used > 0 {
			cursor += used
		}
		meta := uint32FromUint64(metaValue)
		if _, ok := metadata[meta]; ok || payloadLooksKnown(record[cursor:]) {
			out = append(out, rawEvent{timestamp: timestamp, threadID: thread, metadata: meta, stackID: stack, payload: record[cursor:]})
		}
		offset += intFromUint64(size)
	}
	return out
}

func parseStackBlock(payload []byte) []stackRecord {
	var out []stackRecord
	for offset := 0; offset+8 <= len(payload); {
		id := binary.LittleEndian.Uint64(payload[offset : offset+8])
		countOffset := offset + 8
		countSize := 4
		if countOffset+4 > len(payload) {
			break
		}
		count := intFromUint64(uint64(binary.LittleEndian.Uint32(payload[countOffset : countOffset+4])))
		if count < 0 || count > 4096 || countOffset+countSize+count*8 > len(payload) {
			id = uint64(binary.LittleEndian.Uint32(payload[offset : offset+4]))
			count = intFromUint64(uint64(binary.LittleEndian.Uint32(payload[offset+4 : offset+8])))
			countOffset = offset
			countSize = 4
			if count < 0 || count > 4096 || offset+8+count*8 > len(payload) {
				offset++
				continue
			}
		}
		frames := make([]uint64, count)
		for i := range frames {
			frames[i] = binary.LittleEndian.Uint64(payload[countOffset+countSize+i*8 : countOffset+countSize+(i+1)*8])
		}
		out = append(out, stackRecord{id: id, frames: frames})
		offset = countOffset + countSize + count*8
	}
	return out
}

func decodeRuntimeEvent(trace *Trace, provider, name string, eventID uint32, fields []eventField, payload []byte) {
	lowerName := strings.ToLower(name)
	switch {
	case strings.Contains(lowerName, "methodload") || strings.Contains(lowerName, "methoddcend"):
		decodeMethod(trace, fields, payload)
	case strings.Contains(lowerName, "moduleload") || strings.Contains(lowerName, "moduledcend"):
		decodeModule(trace, fields, payload)
	case strings.Contains(lowerName, "assemblyload") || strings.Contains(lowerName, "assemblydcend"):
		decodeAssembly(trace, fields, payload)
	case strings.Contains(strings.ToLower(provider), "dotnetruntime") && strings.Contains(lowerName, "method"):
		decodeMethod(trace, fields, payload)
	case strings.Contains(strings.ToLower(provider), "dotnetruntime"):
		decodeRuntimeByID(trace, eventID, fields, payload)
	}
}

func decodeRuntimeByID(trace *Trace, eventID uint32, fields []eventField, payload []byte) {
	switch eventID {
	case 143, 144:
		decodeMethodRuntime(trace, payload)
	case 152, 153:
		decodeModuleRuntime(trace, payload)
	case 154:
		decodeModuleRuntime(trace, payload)
		decodeAssemblyRuntime(trace, payload)
	case 155, 156:
		decodeAssemblyRuntime(trace, payload)
	default:
		if len(fields) != 0 {
			decodeMethod(trace, fields, payload)
		}
	}
}

func decodeMethodRuntime(trace *Trace, payload []byte) {
	if len(payload) < 24 {
		return
	}
	moduleID := binary.LittleEndian.Uint64(payload[8:])
	startAddress := binary.LittleEndian.Uint64(payload[16:])
	sizeOffset := 24
	if sizeOffset+12 > len(payload) {
		return
	}
	size := uint64(binary.LittleEndian.Uint32(payload[sizeOffset:]))
	if startAddress == 0 || size == 0 {
		return
	}
	offset := sizeOffset + 12
	namespace, ok := readWideNull(payload, &offset)
	if !ok {
		return
	}
	name, ok := readWideNull(payload, &offset)
	if !ok {
		return
	}
	trace.Methods = append(trace.Methods, Method{
		StartAddress: startAddress,
		Size:         size,
		ModuleID:     moduleID,
		Namespace:    namespace,
		Name:         name,
	})
}

func decodeModuleRuntime(trace *Trace, payload []byte) {
	if len(payload) < 16 {
		return
	}
	moduleID := binary.LittleEndian.Uint64(payload)
	assemblyID := binary.LittleEndian.Uint64(payload[8:])
	if moduleID != 0 && assemblyID != 0 {
		trace.Modules[moduleID] = assemblyID
	}
}

func decodeAssemblyRuntime(trace *Trace, payload []byte) {
	if len(payload) < 8 {
		return
	}
	assemblyID := binary.LittleEndian.Uint64(payload)
	if assemblyID == 0 {
		return
	}
	for _, found := range readableStrings(payload[8:]) {
		if strings.Contains(found.value, ".") || strings.Contains(found.value, ",") {
			trace.Assemblies[assemblyID] = found.value
			return
		}
	}
}

func decodeMethod(trace *Trace, fields []eventField, payload []byte) {
	values := decodeFields(fields, payload)
	if len(values) > 0 {
		method := Method{
			StartAddress: uint64Value(values, "MethodStartAddress"),
			Size:         uint64Value(values, "MethodSize"),
			ModuleID:     uint64Value(values, "ModuleID"),
			Name:         stringValue(values, "MethodName"),
			Namespace:    stringValue(values, "MethodNamespace"),
		}
		if method.Name != "" && method.StartAddress != 0 && method.Size != 0 {
			trace.Methods = append(trace.Methods, method)
			return
		}
	}
	stringsFound := readableStrings(payload)
	if len(stringsFound) == 0 {
		return
	}
	var methodName, namespace string
	for _, found := range stringsFound {
		if strings.Contains(found.value, ".") {
			parts := strings.Split(found.value, ".")
			if methodName == "" {
				methodName = parts[len(parts)-1]
				namespace = strings.Join(parts[:len(parts)-1], ".")
			}
		} else if methodName == "" && isIdentifier(found.value) {
			methodName = found.value
		}
	}
	if methodName == "" {
		return
	}
	var start, size, module uint64
	for offset := 0; offset+8 <= len(payload); offset += 4 {
		value := binary.LittleEndian.Uint64(payload[offset : offset+8])
		if value > start && value >= 0x1000 {
			start = value
		}
	}
	for offset := 0; offset+4 <= len(payload); offset += 4 {
		value := uint64(binary.LittleEndian.Uint32(payload[offset : offset+4]))
		if value > 0 && value < 1<<30 && (size == 0 || value < size) {
			size = value
		}
	}
	if start == 0 || size == 0 {
		return
	}
	for offset := 0; offset+8 <= len(payload); offset += 4 {
		value := binary.LittleEndian.Uint64(payload[offset : offset+8])
		if value != start && value > 0 && value < 1<<32 {
			module = value
		}
	}
	trace.Methods = append(trace.Methods, Method{StartAddress: start, Size: size, Namespace: namespace, Name: methodName, ModuleID: module})
}

func decodeModule(trace *Trace, fields []eventField, payload []byte) {
	values := decodeFields(fields, payload)
	if moduleID := uint64Value(values, "ModuleID"); moduleID != 0 {
		if assemblyID := uint64Value(values, "AssemblyID"); assemblyID != 0 {
			trace.Modules[moduleID] = assemblyID
			return
		}
	}
	rawValues := make([]uint64, 0, 3)
	for offset := 0; offset+8 <= len(payload) && len(rawValues) < 3; offset += 8 {
		rawValues = append(rawValues, binary.LittleEndian.Uint64(payload[offset:offset+8]))
	}
	if len(rawValues) < 2 {
		return
	}
	trace.Modules[rawValues[0]] = rawValues[1]
}

func decodeAssembly(trace *Trace, fields []eventField, payload []byte) {
	values := decodeFields(fields, payload)
	if assemblyID := uint64Value(values, "AssemblyID"); assemblyID != 0 {
		if name := stringValue(values, "FullyQualifiedAssemblyName"); name != "" {
			trace.Assemblies[assemblyID] = name
			return
		}
	}
	stringsFound := readableStrings(payload)
	if len(stringsFound) == 0 {
		return
	}
	var assembly uint64
	for offset := 0; offset+8 <= len(payload); offset += 8 {
		value := binary.LittleEndian.Uint64(payload[offset : offset+8])
		if value > 0 && value < 1<<32 {
			assembly = value
			break
		}
	}
	if assembly == 0 {
		return
	}
	name := stringsFound[len(stringsFound)-1].value
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		name = strings.TrimSuffix(name[strings.LastIndexAny(name, `/\`)+1:], ".dll")
	}
	trace.Assemblies[assembly] = name
}

type decodedValue struct {
	number uint64
	text   string
}

func decodeFields(fields []eventField, payload []byte) map[string]decodedValue {
	values := make(map[string]decodedValue, len(fields))
	offset := 0
	for _, field := range fields {
		name := strings.ToLower(field.name)
		switch field.code {
		case 3:
			if offset+1 > len(payload) {
				return values
			}
			values[name] = decodedValue{number: uint64(payload[offset])}
			offset++
		case 4, 7, 8:
			size := 2
			if field.code == 4 {
				size = 2
			}
			if offset+size > len(payload) {
				return values
			}
			values[name] = decodedValue{number: uint64(binary.LittleEndian.Uint16(payload[offset:]))}
			offset += size
		case 5, 6:
			if offset+1 > len(payload) {
				return values
			}
			values[name] = decodedValue{number: uint64(payload[offset])}
			offset++
		case 9, 10:
			if offset+4 > len(payload) {
				return values
			}
			values[name] = decodedValue{number: uint64(binary.LittleEndian.Uint32(payload[offset:]))}
			offset += 4
		case 11, 12:
			if offset+8 > len(payload) {
				return values
			}
			values[name] = decodedValue{number: binary.LittleEndian.Uint64(payload[offset:])}
			offset += 8
		case 13:
			if offset+4 > len(payload) {
				return values
			}
			values[name] = decodedValue{number: uint64(binary.LittleEndian.Uint32(payload[offset:]))}
			offset += 4
		case 14, 15:
			if offset+8 > len(payload) {
				return values
			}
			values[name] = decodedValue{number: binary.LittleEndian.Uint64(payload[offset:])}
			offset += 8
		case 16:
			if offset+8 > len(payload) {
				return values
			}
			values[name] = decodedValue{number: binary.LittleEndian.Uint64(payload[offset:])}
			offset += 8
		case 18:
			text, ok := readWideNull(payload, &offset)
			if !ok {
				return values
			}
			values[name] = decodedValue{text: text}
		default:
			return values
		}
	}
	return values
}

func uint64Value(values map[string]decodedValue, name string) uint64 {
	return values[strings.ToLower(name)].number
}

func stringValue(values map[string]decodedValue, name string) string {
	return values[strings.ToLower(name)].text
}

func resolveMethods(trace *Trace) {
	for i := range trace.Methods {
		if assemblyID := trace.Modules[trace.Methods[i].ModuleID]; assemblyID != 0 {
			trace.Methods[i].Assembly = trace.Assemblies[assemblyID]
		}
	}
	for sampleIndex := range trace.Samples {
		for frameIndex := range trace.Samples[sampleIndex].Frames {
			frame := &trace.Samples[sampleIndex].Frames[frameIndex]
			for _, method := range trace.Methods {
				if frame.IP < method.StartAddress || frame.IP >= method.StartAddress+method.Size {
					continue
				}
				frame.Method = method.Name
				frame.Type = method.Namespace
				frame.Assembly = method.Assembly
				break
			}
		}
	}
}

func isSampleEvent(provider, name string, payload []byte) bool {
	if strings.Contains(strings.ToLower(provider), "sampleprofiler") {
		return true
	}
	if strings.Contains(strings.ToLower(name), "sample") {
		return true
	}
	return bytes.Contains(bytes.ToLower(payload), []byte("sampleprofiler"))
}

func payloadLooksKnown(payload []byte) bool {
	for _, found := range readableStrings(payload) {
		if isEventName(found.value) || isProviderName(found.value) {
			return true
		}
	}
	return false
}

func isProviderName(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "dotnetruntime") || strings.Contains(value, "sampleprofiler")
}

func isEventName(value string) bool {
	lower := strings.ToLower(value)
	for _, part := range []string{"methodload", "methoddcend", "moduleload", "moduledcend", "assemblyload", "assemblydcend", "sample"} {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

type foundString struct {
	value  string
	offset int
}

func readableStrings(data []byte) []foundString {
	var out []foundString
	for offset := 0; offset < len(data); {
		if offset+4 <= len(data) && data[offset] >= 0x20 && data[offset] < 0x7f && data[offset+1] == 0 {
			start := offset
			var builder strings.Builder
			for offset+1 < len(data) && data[offset] >= 0x20 && data[offset] < 0x7f && data[offset+1] == 0 {
				builder.WriteByte(data[offset])
				offset += 2
			}
			if builder.Len() >= 2 {
				out = append(out, foundString{value: builder.String(), offset: start})
			}
			continue
		}
		if data[offset] >= 0x20 && data[offset] < 0x7f {
			start := offset
			for offset < len(data) && data[offset] >= 0x20 && data[offset] < 0x7f {
				offset++
			}
			if offset-start >= 2 {
				out = append(out, foundString{value: string(data[start:offset]), offset: start})
			}
			continue
		}
		offset++
	}
	return out
}

func isIdentifier(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if r == '.' || r == '_' || r == '`' || r == '+' || r == '<' || r == '>' || r == ':' || r == ',' || r == ' ' {
			continue
		}
		if r < '0' || r > '9' && r < 'A' || r > 'Z' && r < 'a' || r > 'z' {
			return false
		}
	}
	return value != ""
}

func readVarint(data []byte) (uint64, int) {
	var value uint64
	for i, b := range data {
		if i >= 10 {
			return 0, 0
		}
		value |= uint64(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return value, i + 1
		}
	}
	return 0, 0
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func uint32FromUint64(value uint64) uint32 {
	const maxUint32 = uint64(1<<32 - 1)
	if value > maxUint32 {
		return ^uint32(0)
	}
	return uint32(value)
}

func intFromUint64(value uint64) int {
	maxInt := uint64(^uint(0) >> 1)
	if value > maxInt {
		return int(maxInt)
	}
	return int(value)
}

func uint64FromInt(value int) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value)
}
