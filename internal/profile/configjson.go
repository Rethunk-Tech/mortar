package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/jsonc"
)

type jsonPair struct {
	Key   string
	Value jsonNode
}

type jsonNode struct {
	kind byte
	num  json.Number
	str  string
	flag bool
	arr  []jsonNode
	obj  []jsonPair
}

const (
	jsonNull = iota
	jsonBool
	jsonNum
	jsonStr
	jsonArr
	jsonObj
)

func rewriteConfigJSON(data []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(jsonc.Clean(data)))
	dec.UseNumber()
	node, err := decodeJSON(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("config.json has trailing data")
	}
	var buf bytes.Buffer
	if err := encodeJSON(&buf, node, 0); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func decodeJSON(dec *json.Decoder) (jsonNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return jsonNode{}, err
	}
	switch v := tok.(type) {
	case nil:
		return jsonNode{kind: jsonNull}, nil
	case bool:
		return jsonNode{kind: jsonBool, flag: v}, nil
	case json.Number:
		return jsonNode{kind: jsonNum, num: v}, nil
	case string:
		return jsonNode{kind: jsonStr, str: v}, nil
	case json.Delim:
		switch v {
		case '[':
			var arr []jsonNode
			for dec.More() {
				n, err := decodeJSON(dec)
				if err != nil {
					return jsonNode{}, err
				}
				arr = append(arr, n)
			}
			end, err := dec.Token()
			if err != nil {
				return jsonNode{}, err
			}
			if end != json.Delim(']') {
				return jsonNode{}, fmt.Errorf("expected ]")
			}
			return jsonNode{kind: jsonArr, arr: arr}, nil
		case '{':
			var obj []jsonPair
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return jsonNode{}, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return jsonNode{}, fmt.Errorf("expected object key")
				}
				val, err := decodeJSON(dec)
				if err != nil {
					return jsonNode{}, err
				}
				obj = append(obj, jsonPair{Key: key, Value: val})
			}
			end, err := dec.Token()
			if err != nil {
				return jsonNode{}, err
			}
			if end != json.Delim('}') {
				return jsonNode{}, fmt.Errorf("expected }")
			}
			return jsonNode{kind: jsonObj, obj: obj}, nil
		}
	}
	return jsonNode{}, fmt.Errorf("unexpected JSON token")
}

func encodeJSON(buf *bytes.Buffer, n jsonNode, depth int) error {
	switch n.kind {
	case jsonNull:
		buf.WriteString("null")
	case jsonBool:
		if n.flag {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case jsonNum:
		buf.WriteString(n.num.String())
	case jsonStr:
		enc, err := json.Marshal(n.str)
		if err != nil {
			return err
		}
		buf.Write(enc)
	case jsonArr:
		if len(n.arr) == 0 {
			buf.WriteString("[]")
			return nil
		}
		buf.WriteString("[\n")
		pad := indent(depth + 1)
		for i, item := range n.arr {
			buf.WriteString(pad)
			if err := encodeJSON(buf, item, depth+1); err != nil {
				return err
			}
			if i < len(n.arr)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(indent(depth))
		buf.WriteByte(']')
	case jsonObj:
		if len(n.obj) == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteString("{\n")
		pad := indent(depth + 1)
		for i, p := range n.obj {
			buf.WriteString(pad)
			key, err := json.Marshal(p.Key)
			if err != nil {
				return err
			}
			buf.Write(key)
			buf.WriteString(": ")
			if err := encodeJSON(buf, p.Value, depth+1); err != nil {
				return err
			}
			if i < len(n.obj)-1 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(indent(depth))
		buf.WriteByte('}')
	default:
		return fmt.Errorf("unknown JSON node")
	}
	return nil
}

func indent(depth int) string {
	return strings.Repeat("  ", depth)
}
