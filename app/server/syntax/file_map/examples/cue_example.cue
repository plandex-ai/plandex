// Package definition
package example

// Import statements
import (
    "strings"
    "time"
)

// Schema definitions
#Person: {
    name:     string
    age:      int & >=0 & <=120
    email?:   string & =~"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
    address?: #Address
}

#Address: {
    street:  string
    city:    string
    country: string
    zip:     string & =~"^[0-9]{5}$"
}

// Default values and constraints
#DefaultPerson: #Person & {
    name: string | *"John Doe"
    age:  int | *30
}

// List type definition
#Team: {
    name:    string
    members: [...#Person]
    leader:  #Person
}

// Computed fields
#Employee: #Person & {
    role:    string
    salary:  float
    taxRate: float

    // Computed field
    netSalary: salary * (1 - taxRate)
}

// Concrete values
exampleTeam: #Team & {
    name: "Engineering"
    members: [
        {
            name: "Alice Smith"
            age:  28
            email: "alice@example.com"
        },
        {
            name: "Bob Jones"
            age:  35
            email: "bob@example.com"
        }
    ]
    leader: {
        name:  "Carol Wilson"
        age:   40
        email: "carol@example.com"
    }
}

// Configuration with references and templates
#Config: {
    environment: "development" | "staging" | "production"
    database: {
        host:     string
        port:     int & >0 & <65536
        username: string
        password: string
    }
    features: [string]: bool
}

// Template usage
productionConfig: #Config & {
    environment: "production"
    database: {
        host:     "db.example.com"
        port:     5432
        username: "admin"
        password: "secret"
    }
    features: {
        "feature1": true
        "feature2": false
    }
}
