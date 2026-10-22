package claims

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// jsonError describes a problem of the JSON document of a claim file. It
// always carries the line of the problem.
type jsonError struct {
	// line is the line of the document where the problem was found.
	line int
	// message explains the problem and how to recover from it.
	message string
}

func (e *jsonError) Error() string {
	return fmt.Sprintf("line %d: %s", e.line, e.message)
}

// jsonPositions holds the position of every element of a claim file that the
// validation needs to name in its messages.
type jsonPositions struct {
	// fields holds the line of every key of the top level object.
	fields map[string]int
	// claims holds the position of every claim of the "claims" array.
	claims []jsonClaimPosition
}

// jsonClaimPosition holds the line of a claim and the line of every key of
// the claim.
type jsonClaimPosition struct {
	// line is the line where the claim starts.
	line int
	// keys holds the line of every key of the claim.
	keys map[string]int
}

// scanJSON reads the structure of a claim file and records the line of every
// key of the document and of every claim.
//
// The scan is separate from the validation on purpose: the validation needs
// to name the line of the problem, and the standard library does not report
// the position of a semantic problem.
func scanJSON(data []byte) (*jsonPositions, error) {
	positions := &jsonPositions{fields: map[string]int{}}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return nil, &jsonError{line: lineOf(data, offsetOf(err)), message: fmt.Sprintf("the claim file is not valid JSON: %v", err)}
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return nil, &jsonError{line: 1, message: "the claim file must be a JSON object"}
	}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, &jsonError{line: lineOf(data, decoder.InputOffset()), message: fmt.Sprintf("the claim file is not valid JSON: %v", err)}
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, &jsonError{line: lineOf(data, decoder.InputOffset()), message: "the claim file must be a JSON object"}
		}
		keyLine := lineOf(data, decoder.InputOffset())
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, &jsonError{line: keyLine, message: fmt.Sprintf("the claim file is not valid JSON: %v", err)}
		}
		valueOffset := decoder.InputOffset() - int64(len(raw))
		positions.fields[key] = keyLine
		if key == "claims" {
			claims, err := scanClaims(data, valueOffset, raw)
			if err != nil {
				return nil, err
			}
			positions.claims = claims
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, &jsonError{line: lineOf(data, decoder.InputOffset()), message: fmt.Sprintf("the claim file is not valid JSON: %v", err)}
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, &jsonError{line: lineOf(data, decoder.InputOffset()), message: "the claim file must hold a single JSON object"}
	}
	return positions, nil
}

// scanClaims records the position of every claim of the "claims" array.
func scanClaims(data []byte, valueOffset int64, raw []byte) ([]jsonClaimPosition, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return nil, &jsonError{line: lineOf(data, valueOffset), message: fmt.Sprintf("the claim file is not valid JSON: %v", err)}
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '[' {
		return nil, &jsonError{line: lineOf(data, valueOffset), message: "the field \"claims\" must be a list of claims"}
	}
	var claims []jsonClaimPosition
	for decoder.More() {
		elementOffset := decoder.InputOffset()
		var element json.RawMessage
		if err := decoder.Decode(&element); err != nil {
			return nil, &jsonError{line: lineOf(data, valueOffset+elementOffset), message: fmt.Sprintf("the claim file is not valid JSON: %v", err)}
		}
		absolute := valueOffset + skipToElement(raw, elementOffset)
		line := lineOf(data, absolute)
		keys, err := scanKeys(element)
		if err != nil {
			return nil, &jsonError{line: line, message: "every claim must be a JSON object"}
		}
		position := jsonClaimPosition{line: line, keys: map[string]int{}}
		for key, relative := range keys {
			position.keys[key] = line + relative - 1
		}
		claims = append(claims, position)
	}
	return claims, nil
}

// scanKeys records the line of every key of a JSON object, relative to the
// first line of the object.
func scanKeys(object []byte) (map[string]int, error) {
	keys := map[string]int{}
	decoder := json.NewDecoder(bytes.NewReader(object))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return nil, errors.New("the value is not a JSON object")
	}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, errors.New("the key of the object is not a text value")
		}
		line := lineOf(object, decoder.InputOffset())
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		keys[key] = line
	}
	return keys, nil
}

// lineOf returns the line of a document where a byte offset points.
func lineOf(data []byte, offset int64) int {
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(data)) {
		offset = int64(len(data))
	}
	return 1 + bytes.Count(data[:offset], []byte("\n"))
}

// skipSpaces returns the offset of the first byte that is not a space.
func skipSpaces(data []byte, offset int64) int64 {
	for offset < int64(len(data)) {
		switch data[offset] {
		case ' ', '\t', '\r', '\n':
			offset++
			continue
		}
		break
	}
	return offset
}

// skipToElement returns the offset of the next element of an array.
//
// The decoder stops right after the element it read, so the separator of the
// array and the spaces around it have to be skipped before the element is
// located.
func skipToElement(data []byte, offset int64) int64 {
	offset = skipSpaces(data, offset)
	if offset < int64(len(data)) && data[offset] == ',' {
		offset = skipSpaces(data, offset+1)
	}
	return offset
}

// offsetOf returns the byte offset of a decoding error, when the error
// carries one.
func offsetOf(err error) int64 {
	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return syntaxError.Offset
	}
	return 0
}
