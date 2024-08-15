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