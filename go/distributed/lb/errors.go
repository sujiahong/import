package lb

import "errors"

var ErrNoInstances = errors.New("no instances available")
var ErrNoHealthyInstances = errors.New("no healthy instances available")
