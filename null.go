package fat

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
)

var (
	_ sql.Scanner   = (*NullUint128)(nil)
	_ sql.Scanner   = (*NullInt128)(nil)
	_ sql.Scanner   = (*NullUint256)(nil)
	_ sql.Scanner   = (*NullInt256)(nil)
	_ driver.Valuer = NullUint128{}
	_ driver.Valuer = NullInt128{}
	_ driver.Valuer = NullUint256{}
	_ driver.Valuer = NullInt256{}

	_ json.Marshaler   = NullUint128{}
	_ json.Marshaler   = NullInt128{}
	_ json.Marshaler   = NullUint256{}
	_ json.Marshaler   = NullInt256{}
	_ json.Unmarshaler = (*NullUint128)(nil)
	_ json.Unmarshaler = (*NullInt128)(nil)
	_ json.Unmarshaler = (*NullUint256)(nil)
	_ json.Unmarshaler = (*NullInt256)(nil)
)

// NullUint128 is a Uint128 that may be NULL, as sql.NullInt64
type NullUint128 struct {
	Uint128 Uint128
	Valid   bool // Valid is true if Uint128 is not NULL
}

// Scan reads NULL as zero and not Valid, the rest as Uint128.Scan does
func (n *NullUint128) Scan(src any) error {
	if src == nil {
		*n = NullUint128{}
		return nil
	}
	err := n.Uint128.Scan(src)
	n.Valid = err == nil
	return err
}

// Value is NULL when not Valid, else as Uint128.Value
func (n NullUint128) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Uint128.Dec(), nil
}

// MarshalJSON is null when not Valid, else as Uint128.MarshalJSON
func (n NullUint128) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Uint128.MarshalJSON()
}

// UnmarshalJSON reads null as zero and not Valid, the rest as Uint128 does.
// An error leaves n as is, as encoding/json does, unlike Scan
func (n *NullUint128) UnmarshalJSON(text []byte) error {
	if string(text) == "null" {
		*n = NullUint128{}
		return nil
	}
	if err := n.Uint128.UnmarshalJSON(text); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

// NullInt128 is an Int128 that may be NULL, as sql.NullInt64
type NullInt128 struct {
	Int128 Int128
	Valid  bool // Valid is true if Int128 is not NULL
}

// Scan reads NULL as zero and not Valid, the rest as Int128.Scan does
func (n *NullInt128) Scan(src any) error {
	if src == nil {
		*n = NullInt128{}
		return nil
	}
	err := n.Int128.Scan(src)
	n.Valid = err == nil
	return err
}

// Value is NULL when not Valid, else as Int128.Value
func (n NullInt128) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int128.Dec(), nil
}

// MarshalJSON is null when not Valid, else as Int128.MarshalJSON
func (n NullInt128) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Int128.MarshalJSON()
}

// UnmarshalJSON reads null as zero and not Valid, the rest as Int128 does.
// An error leaves n as is, as encoding/json does, unlike Scan
func (n *NullInt128) UnmarshalJSON(text []byte) error {
	if string(text) == "null" {
		*n = NullInt128{}
		return nil
	}
	if err := n.Int128.UnmarshalJSON(text); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

// NullUint256 is a Uint256 that may be NULL, as sql.NullInt64
type NullUint256 struct {
	Uint256 Uint256
	Valid   bool // Valid is true if Uint256 is not NULL
}

// Scan reads NULL as zero and not Valid, the rest as Uint256.Scan does
func (n *NullUint256) Scan(src any) error {
	if src == nil {
		*n = NullUint256{}
		return nil
	}
	err := n.Uint256.Scan(src)
	n.Valid = err == nil
	return err
}

// Value is NULL when not Valid, else as Uint256.Value
func (n NullUint256) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Uint256.Dec(), nil
}

// MarshalJSON is null when not Valid, else as Uint256.MarshalJSON
func (n NullUint256) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Uint256.MarshalJSON()
}

// UnmarshalJSON reads null as zero and not Valid, the rest as Uint256 does.
// An error leaves n as is, as encoding/json does, unlike Scan
func (n *NullUint256) UnmarshalJSON(text []byte) error {
	if string(text) == "null" {
		*n = NullUint256{}
		return nil
	}
	if err := n.Uint256.UnmarshalJSON(text); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

// NullInt256 is an Int256 that may be NULL, as sql.NullInt64
type NullInt256 struct {
	Int256 Int256
	Valid  bool // Valid is true if Int256 is not NULL
}

// Scan reads NULL as zero and not Valid, the rest as Int256.Scan does
func (n *NullInt256) Scan(src any) error {
	if src == nil {
		*n = NullInt256{}
		return nil
	}
	err := n.Int256.Scan(src)
	n.Valid = err == nil
	return err
}

// Value is NULL when not Valid, else as Int256.Value
func (n NullInt256) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int256.Dec(), nil
}

// MarshalJSON is null when not Valid, else as Int256.MarshalJSON
func (n NullInt256) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Int256.MarshalJSON()
}

// UnmarshalJSON reads null as zero and not Valid, the rest as Int256 does.
// An error leaves n as is, as encoding/json does, unlike Scan
func (n *NullInt256) UnmarshalJSON(text []byte) error {
	if string(text) == "null" {
		*n = NullInt256{}
		return nil
	}
	if err := n.Int256.UnmarshalJSON(text); err != nil {
		return err
	}
	n.Valid = true
	return nil
}
