function fetchData() {
	const apiURL = 'https://api.example.com/data';
	fetch(apiURL)
		.then((response) => response.json())
		.then((data) => console.log(data))
		.catch((error) => console.error('Error:', error));
}