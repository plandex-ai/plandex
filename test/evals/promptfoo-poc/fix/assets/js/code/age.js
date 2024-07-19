//JavaScript function to calculate age

function calculateAge(birthYear) {
	const currentYear = new Date().getFullYear();
	return currentYear - birthYear;
}

console.log(calculateAge(1990));