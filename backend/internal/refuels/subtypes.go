package refuels

// subtypesByCarFuel lists the refuel subtypes each kind of car accepts: an
// electric car records charges, every other car fuel grades. `other` fits
// both.
var subtypesByCarFuel = map[bool]map[string]bool{
	true:  {"home": true, "ac": true, "dc": true, "other": true},
	false: {"normal": true, "special": true, "other": true},
}

// FitsCarFuel reports whether a refuel subtype is allowed for a car with
// the given fuel.
func FitsCarFuel(carFuel, subtype string) bool {
	return subtypesByCarFuel[carFuel == "electric"][subtype]
}
