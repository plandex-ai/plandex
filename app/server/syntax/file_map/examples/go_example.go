package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Interface definition
type DataProcessor interface {
	Process(ctx context.Context, data interface{}) error
	Validate(data interface{}) bool
}

// Custom error type
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// Struct with embedded type and tags
type User struct {
	sync.Mutex
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Type alias and constants
type UserID = int64

const (
	MaxRetries   = 3
	DefaultLimit = 100
)

// single line const
const singleLineConst string = "single line const"

// Global variables
var (
	defaultTimeout = time.Second * 30
	processor      DataProcessor
)

// single line var
var singleLineVar string = "single line var"

// Generic type
type Result[T any] struct {
	Data    T
	Error   error
	Retries int
}

// Implementation of DataProcessor
type UserProcessor struct {
	cache map[UserID]*User
	mu    sync.RWMutex
}

func NewUserProcessor() *UserProcessor {
	return &UserProcessor{
		cache: make(map[UserID]*User),
	}
}

func (p *UserProcessor) Process(ctx context.Context, data interface{}) error {
	user, ok := data.(*User)
	if !ok {
		return fmt.Errorf("invalid data type: expected *User")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.cache[user.ID] = user
	return nil
}

func (p *UserProcessor) Validate(data interface{}) bool {
	user, ok := data.(*User)
	return ok && user.Name != "" && user.Email != ""
}

// Channel operations
func processUsers(ctx context.Context, users <-chan *User) <-chan *Result[*User] {
	results := make(chan *Result[*User])

	go func() {
		defer close(results)
		for user := range users {
			select {
			case <-ctx.Done():
				return
			case results <- &Result[*User]{Data: user}:
			}
		}
	}()

	return results
}

// Function with multiple return values and named returns
func createUser(name, email string) (user *User, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	user = &User{
		Name:      name,
		Email:     email,
		CreatedAt: time.Now(),
	}

	if !processor.Validate(user) {
		return nil, &ValidationError{Field: "user", Message: "invalid user data"}
	}

	return user, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	processor = NewUserProcessor()

	users := make(chan *User)
	results := processUsers(ctx, users)

	// Anonymous struct
	config := struct {
		Workers int
		Buffer  int
	}{
		Workers: 3,
		Buffer:  10,
	}

	var wg sync.WaitGroup
	wg.Add(config.Workers)

	for i := 0; i < config.Workers; i++ {
		go func(id int) {
			defer wg.Done()
			for result := range results {
				if result.Error != nil {
					log.Printf("Worker %d: Error processing user: %v", id, result.Error)
					continue
				}
				log.Printf("Worker %d: Processed user: %+v", id, result.Data)
			}
		}(i)
	}

	// Cleanup
	close(users)
	wg.Wait()
}
