package licensecheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// LoadInput parses the checklist payload. Duplicate JSON object keys are
// rejected so that component ids and license rules cannot be shadowed.
func LoadInput(r io.Reader) (Input, error) {
	var in Input

	raw, err := io.ReadAll(r)
	if err != nil {
		return in, fmt.Errorf("read input: %w", err)
	}
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))

	if err := rejectDuplicateKeys(bytes.NewReader(raw)); err != nil {
		return in, err
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&in); err != nil {
		return in, fmt.Errorf("parse input: %w", err)
	}
	if dec.More() {
		return in, fmt.Errorf("parse input: unexpected trailing values")
	}

	if err := validate(&in); err != nil {
		return in, err
	}
	return in, nil
}

func rejectDuplicateKeys(r io.Reader) error {
	dec := json.NewDecoder(r)
	type frame struct {
		isArray bool
		keys    map[string]struct{}
	}
	var stack []frame

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("parse input: %w", err)
		}

		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{':
				stack = append(stack, frame{keys: map[string]struct{}{}})
			case '[':
				stack = append(stack, frame{isArray: true})
			case '}', ']':
				if len(stack) == 0 {
					return fmt.Errorf("parse input: unbalanced JSON")
				}
				stack = stack[:len(stack)-1]
			}
		case string:
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				if !top.isArray {
					// Inside an object a string token is always a key.
					if _, ok := top.keys[v]; ok {
						return fmt.Errorf("parse input: duplicate key %q", v)
					}
					top.keys[v] = struct{}{}
				}
			}
		}
	}
	if len(stack) != 0 {
		return fmt.Errorf("parse input: unbalanced JSON")
	}
	return nil
}

func validate(in *Input) error {
	components := make(map[string]*Component, len(in.Components))
	for i := range in.Components {
		c := &in.Components[i]
		if c.ID == "" {
			return fmt.Errorf("component #%d has an empty id", i+1)
		}
		if _, dup := components[c.ID]; dup {
			return fmt.Errorf("duplicate component id %q", c.ID)
		}
		components[c.ID] = c
	}

	rules := make(map[string]struct{}, len(in.Rules))
	for _, rl := range in.Rules {
		if rl.License == "" {
			return fmt.Errorf("rule has an empty license")
		}
		if _, dup := rules[rl.License]; dup {
			return fmt.Errorf("duplicate license rule %q", rl.License)
		}
		rules[rl.License] = struct{}{}
	}

	for _, c := range in.Components {
		for _, dep := range c.DependsOn {
			if _, ok := components[dep]; !ok {
				return fmt.Errorf("component %q depends on unknown component %q", c.ID, dep)
			}
		}
	}

	seenRoots := map[string]struct{}{}
	for _, root := range in.Roots {
		if _, ok := components[root]; !ok {
			return fmt.Errorf("delivery root %q does not exist", root)
		}
		if _, dup := seenRoots[root]; dup {
			return fmt.Errorf("duplicate delivery root %q", root)
		}
		seenRoots[root] = struct{}{}
	}
	return nil
}
