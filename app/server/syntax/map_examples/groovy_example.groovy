#!/usr/bin/env groovy

// Trait definition
trait Loggable {
    def log(String message) {
        println "[${new Date()}] $message"
    }
}

// Abstract class
abstract class Vehicle {
    String make
    String model
    Integer year
    
    abstract void start()
    abstract void stop()
}

// Class with trait implementation
class Car extends Vehicle implements Loggable {
    // Properties with type definitions
    private BigDecimal price
    protected Boolean running = false
    
    // Static fields
    static final String MANUFACTURER = "Generic Motors"
    static def carCount = 0
    
    // Constructor with default parameters
Car(String make = 'Unknown', String model = 'Generic', Integer year = 2024) {
    this.make = make
    this.model = model
    this.year = year
    carCount++
}
    
    // Getter with @Lazy annotation
    @Lazy
    String fullName = "$make $model ($year)"
    
    // Method implementation with closure parameter
    void start() {
        running = true
        log "Starting ${fullName}"
        withLock {
            // Some synchronized code
            println "Engine started"
        }
    }
    
    void stop() {
        running = false
        log "Stopping ${fullName}"
    }
    
    // Operator overloading
    def plus(Car other) {
        new Car(
make: "${this.make}-${other.make}",
model: "${this.model}-${other.model}",
year: Math.max(this.year, other.year)
        )
    }
    
    // Property with custom getter/setter
    private BigDecimal _price
    void setPrice(BigDecimal price) {
        if (price < 0) throw new IllegalArgumentException("Price cannot be negative")
        this._price = price
    }
    BigDecimal getPrice() { _price }
}

// Enum definition
enum Status {
ACTIVE('A'),
INACTIVE('I'),
PENDING('P')
    
    final String code
    
    private Status(String code) {
        this.code = code
    }
    
    String toString() { code }
}

// Category class
class StringExtensions {
    static String truncate(String self, Integer length) {
        self.size() <= length ? self : self[0..<length] + "..."
    }
}

// Builder pattern using closure
class EmailBuilder {
    private def email = [:]
    
    EmailBuilder to(String recipient) {
        email.to = recipient
        this
    }
    
    EmailBuilder from(String sender) {
        email.from = sender
        this
    }
    
    EmailBuilder subject(String subject) {
        email.subject = subject
        this
    }
    
    EmailBuilder body(@DelegatesTo(strategy=Closure.DELEGATE_FIRST) Closure body) {
        def builder = new StringBuilder()
        body.delegate = builder
        body.resolveStrategy = Closure.DELEGATE_FIRST
        body()
        email.body = builder.toString()
        this
    }
    
    Map build() {
        email.clone() as Map
    }
}

// Main script execution
def main() {
    use(StringExtensions) {
        def description = "This is a very long description that needs truncating"
        println description.truncate(20)
    }
    
    def car = new Car(make: "Tesla", model: "Model S")
    car.start()
def email = new EmailBuilder()
    .to("recipient@example.com")
    .from("sender@example.com")
    .subject("Test Email")
    .body {
        append "Hello,\n"
        append "This is a test email.\n"
        append "Regards"
    }
    .build()
    .build()
    
    println email
}

if (this.metaClass.getMetaProperty('main')) {
    main()
}
    main()
}
}


