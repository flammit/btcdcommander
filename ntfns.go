package btcdcommander

import (
	"encoding/json"
	"github.com/conformal/btcjson"
)

var (
	BtcdConnectedNtfnMethod    = "btcdconnected"
	BtcdDisconnectedNtfnMethod = "btcddisconnected"
)

func init() {
	btcjson.RegisterCustomCmdGenerator(BtcdConnectedNtfnMethod,
		func() btcjson.Cmd { return new(BtcdConnectedNtfn) })
	btcjson.RegisterCustomCmdGenerator(BtcdDisconnectedNtfnMethod,
		func() btcjson.Cmd { return new(BtcdDisconnectedNtfn) })
}

// BtcdConnectedNtfn is a type handling custom marshaling and
// unmarshaling of getaccount JSON RPC commands.
type BtcdConnectedNtfn struct {
}

// Enforce that BtcdConnectedCmd satisifies the Cmd interface.
var _ btcjson.Cmd = &BtcdConnectedNtfn{}

// Id satisfies the Cmd interface by returning the id of the command.
func (cmd *BtcdConnectedNtfn) Id() interface{} {
	return nil
}

// SetId allows one to modify the Id of a Cmd to help in relaying them.
func (cmd *BtcdConnectedNtfn) SetId(id interface{}) {
}

// Method satisfies the Cmd interface by returning the json method.
func (cmd *BtcdConnectedNtfn) Method() string {
	return BtcdConnectedNtfnMethod
}

// MarshalJSON returns the JSON encoding of cmd.  Part of the Cmd interface.
func (cmd *BtcdConnectedNtfn) MarshalJSON() ([]byte, error) {
	// Fill and marshal a RawCmd.
	return json.Marshal(btcjson.RawCmd{
		Jsonrpc: "1.0",
		Method:  BtcdConnectedNtfnMethod,
		Id:      nil,
		Params:  []interface{}{},
	})
}

// UnmarshalJSON unmarshals the JSON encoding of cmd into cmd.  Part of
// the Cmd interface.
func (cmd *BtcdConnectedNtfn) UnmarshalJSON(b []byte) error {
	return nil
}

// BtcdDisconnectedNtfn is a type handling custom marshaling and
// unmarshaling of getaccount JSON RPC commands.
type BtcdDisconnectedNtfn struct {
}

// Enforce that BtcdDisconnectedNtfn satisifies the Cmd interface.
var _ btcjson.Cmd = &BtcdDisconnectedNtfn{}

// Id satisfies the Cmd interface by returning the id of the command.
func (cmd *BtcdDisconnectedNtfn) Id() interface{} {
	return nil
}

// SetId allows one to modify the Id of a Cmd to help in relaying them.
func (cmd *BtcdDisconnectedNtfn) SetId(id interface{}) {
}

// Method satisfies the Cmd interface by returning the json method.
func (cmd *BtcdDisconnectedNtfn) Method() string {
	return BtcdDisconnectedNtfnMethod
}

// MarshalJSON returns the JSON encoding of cmd.  Part of the Cmd interface.
func (cmd *BtcdDisconnectedNtfn) MarshalJSON() ([]byte, error) {
	// Fill and marshal a RawCmd.
	return json.Marshal(btcjson.RawCmd{
		Jsonrpc: "1.0",
		Method:  BtcdDisconnectedNtfnMethod,
		Id:      nil,
		Params:  []interface{}{},
	})
}

// UnmarshalJSON unmarshals the JSON encoding of cmd into cmd.  Part of
// the Cmd interface.
func (cmd *BtcdDisconnectedNtfn) UnmarshalJSON(b []byte) error {
	return nil
}
