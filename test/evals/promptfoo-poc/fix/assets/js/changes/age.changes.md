### Subtask 1:  Add error handling to ensure 'birthYear' is not in the future.

```js
function calculateAge(birthYear) {
	const currentYear = new Date().getFullYear();
	if (birthYear > currentYear) {
		console.error("Birth year cannot be in the future.");
	}
}

console.log(calculateAge(1990));
```

### Subtask 2:  Ensure the function prints a message instead of returning the age.

```js
function calculateAge(birthYear) {
	const currentYear = new Date().getFullYear();
	if (birthYear > currentYear) {
		console.error("Birth year cannot be in the future.");
	}
	console.log(currentYear - birthYear);
}

console.log(calculateAge(1990));
```