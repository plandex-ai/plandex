### Subtask 1:  Update the `handler` function to log the request method and URI

```go
package server

import (
 "log"
 "net/http"
)

func init() { log.Println("Server starting...") }

func handler(w http.ResponseWriter, r *http.Request) { 
    w.Write([]byte("Hello, world!")) 
    log.Printf("Request: %s %s", r.Method, r.RequestURI)
}
```

### Subtask 2: Add a new endpoint for `/api` that returns `API is live`

```go
package server

import (
 "log"
 "net/http"
)

func init() { log.Println("Server starting...") }

func handler(w http.ResponseWriter, r *http.Request) { 
    w.Write([]byte("Hello, world!")) 
    log.Printf("Request: %s %s", r.Method, r.RequestURI)
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("API is live"))
}
```

### Subtask 3: Ensure logging of server startup includes timestamp

```go
package server

import (
 "log"
 "net/http"
 "time"
)

func init() { log.Printf("Server starting at %s...", time.Now()) }

func handler(w http.ResponseWriter, r *http.Request) { 
    w.Write([]byte("Hello, world!")) 
    log.Printf("Request: %s %s", r.Method, r.RequestURI)
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("API is live"))
}
```
