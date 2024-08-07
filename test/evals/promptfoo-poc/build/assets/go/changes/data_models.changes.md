### Subtask 1:  Add a `CreatedAt` timestamp field to both `User` and `Post` structs.

```go
package models 

type User struct {
    ID        string
    Username  string
    Email     string
    CreatedAt time.Time
}

type Post struct {
    ID      string
    Content string
    Author  string
    CreatedAt time.Time
}
```

### Subtask 2:  Add a new struct `Comment` with fields `ID`, `Content`, `Author`, and `CreatedAt`.

```go
package models

type Comment struct {
    ID        string
    Content   string
    Author    string
    CreatedAt time.Time
}

type User struct {
    ID        string
    Username  string
    Email     string
    CreatedAt time.Time
}

type Post struct {
    ID      string
    Content string
    Author  string
    CreatedAt time.Time
}
```

### Subtask 3:  Update `Post` to include a slice of `Comment` references.

```go
package models

type Post struct {
    ID        string
    Content   string
    Author    string
    CreatedAt time.Time
    Comments  []*Comment
}

type Comment struct {
    ID        string
    Content   string
    Author    string
    CreatedAt time.Time
}

type User struct {
    ID        string
    Username  string
    Email     string
    CreatedAt time.Time
}
```
