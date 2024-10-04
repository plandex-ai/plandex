pdx-1: function fetchData() {
pdx-2: 	const apiURL = 'https://api.example.com/data';
pdx-3: 	fetch(apiURL)
pdx-4: 		.then((response) => {
pdx-5: 			if (!response.ok) {
pdx-6: 				throw new Error('Network response was not ok');
pdx-7: 			}
pdx-8: 			return response.json();
pdx-9: 		})
pdx-10: 		.then((data) => console.log(data))
pdx-11: 		.catch((error) => console.error('Error:', error));
pdx-12: // rest of the function...
