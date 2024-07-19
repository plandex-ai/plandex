function calculateAge(birthYear) {
	const currentYear = new Date().getFullYear();
	if (birthYear > currentYear) {
		console.error("Birth year cannot be in the future.");
	} else {
	console.log(currentYear - birthYear);
}

console.log(calculateAge(1990));