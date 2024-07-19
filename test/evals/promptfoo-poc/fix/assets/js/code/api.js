pdx-1: function fetchData() {
pdx-2: 	const apiURL = 'https://api.example.com/data';
pdx-3: 	fetch(apiURL)
pdx-4: 		.then((response) => response.json())
pdx-5: 		.then((data) => console.log(data))
pdx-6: 		.catch((error) => console.error('Error:', error));
pdx-7: }
