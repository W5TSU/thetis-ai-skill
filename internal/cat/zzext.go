package cat

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// LookupZZField returns the registered wire shape for a ZZxx extended CAT
// command (case-insensitive), or false if it isn't in zzFields — either
// because it's outside the commands confirmed to fit a plain
// bool/unsigned/signed/action shape (see zzfields.go's package doc), or
// because it's TX-capable and deliberately excluded (ZZTX, ZZTU — see
// SetTuneCAT and the existing PTT command for their gated equivalents).
func LookupZZField(code string) (ZZField, bool) {
	code = strings.ToUpper(code)
	for _, f := range zzFields {
		if f.Code == code {
			return f, true
		}
	}
	return ZZField{}, false
}

// ListZZFields returns every registered ZZxx field, sorted by code.
func ListZZFields() []ZZField {
	out := make([]ZZField, len(zzFields))
	copy(out, zzFields)
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// SetZZ sets a registered ZZxx field to value, encoding it per the field's
// Kind and validating against its known range if any (both confirmed
// against the field's actual CATCommands.cs handler body, not guessed from
// its struct entry alone). Returns an error for ZZAction fields (use SendZZ
// instead) or codes not in the registry.
func (c *Client) SetZZ(code string, value int) error {
	f, ok := LookupZZField(code)
	if !ok {
		return fmt.Errorf("cat: %q is not a registered ZZ field (see PROTOCOLS.md, or fall back to 'cat set %s <raw>' if you've confirmed the wire format yourself)", code, code)
	}
	if f.HasRange && (value < f.Min || value > f.Max) {
		return fmt.Errorf("cat: %s value %d out of range [%d, %d]", f.Code, value, f.Min, f.Max)
	}
	switch f.Kind {
	case ZZBool:
		if value != 0 && value != 1 {
			return fmt.Errorf("cat: %s is a bool field, want 0 or 1", f.Code)
		}
		return c.Set(f.Code, strconv.Itoa(value))
	case ZZUnsigned:
		if value < 0 {
			return fmt.Errorf("cat: %s is unsigned, got negative value %d", f.Code, value)
		}
		return c.Set(f.Code, fmt.Sprintf("%0*d", f.Width, value))
	case ZZSigned:
		sign := "+"
		abs := value
		if value < 0 {
			sign = "-"
			abs = -value
		}
		return c.Set(f.Code, sign+fmt.Sprintf("%0*d", f.Width-1, abs))
	default:
		return fmt.Errorf("cat: %s takes no value — use SendZZ instead", f.Code)
	}
}

// SendZZ sends a registered ZZAction (no-parameter) ZZxx command — e.g. band/
// step/VFO nudge actions like ZZBA (RX2 down one band).
func (c *Client) SendZZ(code string) error {
	f, ok := LookupZZField(code)
	if !ok {
		return fmt.Errorf("cat: %q is not a registered ZZ field", code)
	}
	if f.Kind != ZZAction {
		return fmt.Errorf("cat: %s takes a value — use SetZZ instead", f.Code)
	}
	return c.Send(f.Code)
}

// GetZZ reads a registered ZZxx field and decodes it per its known Kind:
// bool fields as 0/1, unsigned/signed numeric fields as their integer
// value. ZZAction fields have no value to decode — query them with the raw
// 'cat query' passthrough instead, which returns Thetis's reply unparsed.
func (c *Client) GetZZ(code string) (int, error) {
	f, ok := LookupZZField(code)
	if !ok {
		return 0, fmt.Errorf("cat: %q is not a registered ZZ field", code)
	}
	if f.Kind == ZZAction {
		return 0, fmt.Errorf("cat: %s has no value to read — it's an action, not a field", f.Code)
	}
	reply, err := c.Query(f.Code)
	if err != nil {
		return 0, err
	}
	reply = strings.TrimSpace(reply)
	switch f.Kind {
	case ZZBool:
		switch reply {
		case "0":
			return 0, nil
		case "1":
			return 1, nil
		default:
			return 0, fmt.Errorf("cat: %s reply %q: want 0 or 1", f.Code, reply)
		}
	case ZZUnsigned:
		v, err := strconv.Atoi(reply)
		if err != nil {
			return 0, fmt.Errorf("cat: %s reply %q: %w", f.Code, reply, err)
		}
		return v, nil
	case ZZSigned:
		if len(reply) < 1 {
			return 0, fmt.Errorf("cat: %s reply %q: too short for a signed field", f.Code, reply)
		}
		sign, digits := reply[0:1], reply[1:]
		v, err := strconv.Atoi(digits)
		if err != nil {
			return 0, fmt.Errorf("cat: %s reply %q: %w", f.Code, reply, err)
		}
		if sign == "-" {
			v = -v
		}
		return v, nil
	default:
		return 0, fmt.Errorf("cat: %s has no known decode for kind %v", f.Code, f.Kind)
	}
}
