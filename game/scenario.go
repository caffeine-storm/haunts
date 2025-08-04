package game

type Scenario struct {
	// TODO(tmckee): this should be a []byte provider; right now it's a
	// filesystem path :(
	Script    string
	HouseName string
}
