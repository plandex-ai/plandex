pdx-1: //JavaScript function to calculate age
pdx-2: 
pdx-3: function calculateAge(birthYear) {
pdx-4: 	const currentYear = new Date().getFullYear();
pdx-5: 	return currentYear - birthYear;
pdx-6: }
pdx-7: 
pdx-8: console.log(calculateAge(1990));
