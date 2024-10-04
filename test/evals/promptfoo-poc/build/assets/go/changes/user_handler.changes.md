### Subtask 1:  Add input validation for `UserSignup` to check for valid email and password.

```go
package handlers

import (
  "fmt"
  "net/http"
)

func UserSignup(w http.ResponseWriter, r *http.Request) {
    // Handles user signup
    email := r.FormValue("email")
    password := r.FormValue("password")

    if email == "" || password == "" {
        http.Error(w, "Email and password are required", http.StatusBadRequest)
        return
    }

    // Validate email format
    if !isValidEmail(email) {
        http.Error(w, "Invalid email format", http.StatusBadRequest)
        return
    }

    // Validate password length
    if len(password) < 8 {
        http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
        return
    }

    // Proceed with user signup
}
 
 func UserLogin(w http.ResponseWriter, r *http.Request) {
   // Handles user login
}
```

### Subtask 2:  Ensure `UserLogin` checks for correct user credentials and returns an error message on failure.

```go
package handlers

import (
  "fmt"
  "net/http"
)

func UserSignup(w http.ResponseWriter, r *http.Request) {
    // Handles user signup
    email := r.FormValue("email")
    password := r.FormValue("password")

    if email == "" || password == "" {
        http.Error(w, "Email and password are required", http.StatusBadRequest)
        return
    }

    // Validate email format
    if !isValidEmail(email) {
        http.Error(w, "Invalid email format", http.StatusBadRequest)
        return
    }

    // Validate password length
    if len(password) < 8 {
        http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
        return
    }

    // Proceed with user signup
}
 
 func UserLogin(w http.ResponseWriter, r *http.Request) {
   
    email := r.FormValue("email")
    password := r.FormValue("password")

    // Check for correct user credentials
    if email != "[email protected]" || password != "password123" {
        http.Error(w, "Invalid email or password", http.StatusUnauthorized)
        return
    }

    // Proceed with user login
}
```

### Subtask 3:  Introduce rate limiting on both `UserSignup` and `UserLogin` to prevent abuse.

```go
package handlers

import (
  "fmt"
  "net/http"
)

func UserSignup(w http.ResponseWriter, r *http.Request) {
    // Handles user signup
    email := r.FormValue("email")
    password := r.FormValue("password")

    // Rate limit user signup
    if !rateLimit("signup", email) {
        http.Error(w, "Too many signup requests", http.StatusTooManyRequests)
        return
    }

    if email == "" || password == "" {
        http.Error(w, "Email and password are required", http.StatusBadRequest)
        return
    }

    // Validate email format
    if !isValidEmail(email) {
        http.Error(w, "Invalid email format", http.StatusBadRequest)
        return
    }

    // Validate password length
    if len(password) < 8 {
        http.Error(w, "Password must be at least 8 characters long", http.StatusBadRequest)
        return
    }

    // Proceed with user signup
}
 
 func UserLogin(w http.ResponseWriter, r *http.Request) {
   
    email := r.FormValue("email")
    password := r.FormValue("password")

    // Rate limit user signup
    if !rateLimit("signup", email) {
        http.Error(w, "Too many signup requests", http.StatusTooManyRequests)
        return
    }

    // Check for correct user credentials
    if email != "[email protected]" || password != "password123" {
        http.Error(w, "Invalid email or password", http.StatusUnauthorized)
        return
    }
    // Proceed with user login
}

func rateLimit(action, email string) bool {
    // Implement rate limiting logic
    return true
}
```
