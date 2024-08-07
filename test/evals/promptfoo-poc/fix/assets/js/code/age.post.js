pdx-1: function calculateAge(birthYear) {
pdx-2: 	const currentYear = new Date().getFullYear();
pdx-3: 	if (birthYear > currentYear) {
pdx-4: 		console.error("Birth year cannot be in the future.");
pdx-5: 	} else {
pdx-6: 	console.log(currentYear - birthYear);
pdx-7: }
pdx-8: 
pdx-9: console.log(calculateAge(1990));
