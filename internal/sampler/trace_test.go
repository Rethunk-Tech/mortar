package sampler

import (
	"os"
	"strings"
	"testing"
)

func TestParseSpinTrace(t *testing.T) {
	trace, err := ParseFile("testdata/spin.nettrace")
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.Samples) == 0 {
		t.Fatal("trace has no samples")
	}
	foundAlpha, foundBeta := false, false
	for _, method := range trace.Methods {
		foundAlpha = foundAlpha || strings.HasSuffix(method.Name, "Alpha")
		foundBeta = foundBeta || strings.HasSuffix(method.Name, "Beta")
	}
	if !foundAlpha || !foundBeta {
		t.Fatalf("methods did not include Alpha and Beta: %+v", trace.Methods)
	}
	foundSpin := false
	for _, assembly := range trace.Assemblies {
		foundSpin = foundSpin || strings.HasPrefix(assembly, "Spin")
	}
	if !foundSpin {
		t.Fatalf("assemblies did not include Spin: %+v", trace.Assemblies)
	}
	threadCounts := map[uint32]int{}
	for _, sample := range trace.Samples {
		for _, frame := range sample.Frames {
			if strings.HasSuffix(frame.Method, "Alpha") || strings.HasSuffix(frame.Method, "Beta") {
				threadCounts[sample.ThreadID]++
				break
			}
		}
	}
	var mainThread uint32
	mainCount := 0
	for thread, count := range threadCounts {
		if count > mainCount {
			mainThread, mainCount = thread, count
		}
	}
	if mainCount == 0 {
		t.Fatal("could not identify the fixture main thread")
	}
	var alphaNanos, betaNanos uint64
	for index := 0; index+1 < len(trace.Samples); index++ {
		sample := trace.Samples[index]
		if sample.ThreadID != mainThread || trace.Samples[index+1].Timestamp <= sample.Timestamp {
			continue
		}
		var method string
		if len(sample.Frames) > 0 {
			method = sample.Frames[0].Method
		}
		switch {
		case strings.HasSuffix(method, "Alpha"):
			alphaNanos += trace.Samples[index+1].Timestamp - sample.Timestamp
		case strings.HasSuffix(method, "Beta"):
			betaNanos += trace.Samples[index+1].Timestamp - sample.Timestamp
		}
	}
	if alphaNanos == 0 || betaNanos == 0 || alphaNanos < betaNanos*3/2 || alphaNanos > betaNanos*3 {
		t.Fatalf("unexpected Alpha/Beta sample spread: alpha=%d beta=%d", alphaNanos, betaNanos)
	}
	t.Logf("samples=%d methods=%d assemblies=%v", len(trace.Samples), len(trace.Methods), trace.Assemblies)
}

func TestParseRejectsNonNettrace(t *testing.T) {
	path := t.TempDir() + "/not-a-trace"
	if err := os.WriteFile(path, []byte("not a nettrace"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFile(path); err == nil {
		t.Fatal("ParseFile accepted a non-nettrace file")
	}
}

func TestParseCompressedEvent(t *testing.T) {
	state := v5EventState{}
	body := []byte{0x88, 3, 10, 2, 0xaa, 0xbb}
	event, used, ok := parseCompressedV5Event(body, &state)
	if !ok || used != len(body) {
		t.Fatalf("event parse = %v, used %d", ok, used)
	}
	if event.stackID != 3 || event.timestamp != 10 || string(event.payload) != "\xaa\xbb" {
		t.Fatalf("event = %+v", event)
	}
}

func TestParseMetadataRecord(t *testing.T) {
	payload := appendInt32ForTest(nil, 7)
	payload = appendWideForTest(payload, "Provider")
	payload = appendInt32ForTest(payload, 143)
	payload = appendWideForTest(payload, "MethodLoadVerbose")
	payload = appendUint64ForTest(payload, 0)
	payload = appendInt32ForTest(payload, 1)
	payload = appendInt32ForTest(payload, 5)
	payload = appendInt32ForTest(payload, 0)
	metadata := map[uint32]eventMetadata{}
	parseV5Metadata(payload, metadata)
	if metadata[7].provider != "Provider" || metadata[7].name != "MethodLoadVerbose" || metadata[7].eventID != 143 {
		t.Fatalf("metadata = %+v", metadata[7])
	}
}

func TestParseStackRecord(t *testing.T) {
	block := appendUint32ForTest(nil, 4)
	block = appendUint32ForTest(block, 1)
	block = appendUint32ForTest(block, 16)
	block = appendUint64ForTest(block, 0x1000)
	block = appendUint64ForTest(block, 0x2000)
	stacks := parseV5StackBlock(block, 8)
	if len(stacks) != 1 || len(stacks[0].frames) != 2 || stacks[0].frames[1] != 0x2000 {
		t.Fatalf("stacks = %+v", stacks)
	}
}

func appendInt32ForTest(data []byte, value int32) []byte {
	return appendUint32ForTest(data, uint32(value))
}

func appendUint32ForTest(data []byte, value uint32) []byte {
	return append(data, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))
}

func appendUint64ForTest(data []byte, value uint64) []byte {
	for index := 0; index < 8; index++ {
		data = append(data, byte(value>>uint(index*8)))
	}
	return data
}

func appendWideForTest(data []byte, value string) []byte {
	for _, r := range value {
		data = append(data, byte(r), byte(r>>8))
	}
	return append(data, 0, 0)
}
