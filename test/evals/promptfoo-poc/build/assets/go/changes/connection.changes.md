### Subtask 1:  Update `init` function to include error handling and retry logic for database connection.

```go
package database

import "database/sql"

var db *sql.DB

func init() {
  // Initialize database connection
  var err error
  for i := 0; i < 3; i++ {
      db, err = sql.Open(" postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
      if err != nil {
          log.Println("Error connecting to database: ", err)
          time.Sleep(5 * time.Second)
          continue
      }
      err = db.Ping()
      if err != nil {
          log.Println("Error connecting to database: ", err)
          time.Sleep(5 * time.Second)
          continue
      }
      break
  }
}
 
func GetConnection() *sql.DB {
  return db
}
```

### Subtask 2:  Add logging for successful connection or failure in the `init` method.

```go
package database

import (
    "database/sql"
    "log"
)

var db *sql.DB

func init() {
  // Initialize database connection

  var err error
  for i := 0; i < 3; i++ {
      db, err = sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
      if err != nil {
          log.Println("Error connecting to database: ", err)
          time.Sleep(5 * time.Second)
          continue
      }
      err = db.Ping()
      if err != nil {
          log.Println("Error connecting to database: ", err)
          time.Sleep(5 * time.Second)
          continue
      }
      log.Println("Connected to database")
      break
  }
}
 
func GetConnection() *sql.DB {
  return db
}
```

### Subtask 3:  Provide a new function `CloseConnection` to properly close the database connection.

```go
package database

import (
    "database/sql"
    "log"
)

var db *sql.DB

func init() {
  // Initialize database connection

  var err error
  for i := 0; i < 3; i++ {
      db, err = sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
      if err != nil {
          log.Println("Error connecting to database: ", err)
          time.Sleep(5 * time.Second)
          continue
      }
      err = db.Ping()
      if err != nil {
          log.Println("Error connecting to database: ", err)
          time.Sleep(5 * time.Second)
          continue
      }
      log.Println("Connected to database")
      break
  }
}
 
func GetConnection() *sql.DB {
  return db
}


func CloseConnection() {
  if db != nil {
      db.Close()
      log.Println("Database connection closed")
  }
}
```
