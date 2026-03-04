package testdata

func processRequest(x int) int {
	return x * 2
}

func validateInput(x int) bool {
	return x > 0
}

func handler(x int) (int, bool) {
	//pgogen:hot weight=10000
	result := processRequest(x)

	//pgogen:hot
	valid := validateInput(x)

	return result, valid
}
