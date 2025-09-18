package perspective

// BoardSpaceUnit refers to distances in units of the 'tiles' of a room. That
// is, an NxM room will be N tiles wide and M tiles long. This gives us a
// natural geometry to describe entity/door/furniture etc. locations.
type BoardSpaceUnit int
