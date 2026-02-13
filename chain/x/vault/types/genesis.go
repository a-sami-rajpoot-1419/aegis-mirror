package types

// GenesisState defines the vault module's genesis state
type GenesisState struct{}

// NewGenesisState creates a new GenesisState object
func NewGenesisState() *GenesisState {
	return &GenesisState{}
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() *GenesisState {
	return NewGenesisState()
}
