#!/usr/bin/env groovy

// Trait definition
trait Loggable {
    def log(String message) {
        println "[${new Date()}] $message"
    }
}

// Abstract class definition
abstract class Vehicle {
    String make
    String model
    Integer year
    
    abstract void start()
    abstract void stop()
}

// Class implementation - fixed inheritance syntax
class Car extends Vehicle implements Loggable {
    // Properties with type definitions
    private BigDecimal price
    protected Boolean running = false
    
    // Static fields with proper type declarations
    static final String MANUFACTURER = "Generic Motors"
    static int carCount = 0
    
    // Constructor - fixed parameter initialization
    Car(String make = 'Unknown', String model = 'Generic', Integer year = 2024) {
        this.make = make
        this.model = model
        this.year = year
        carCount++
    }
    
    // Lazy property evaluation
    @Lazy String fullName = "$make $model ($year)"
    
    // Method implementation with synchronized block
    void start() {
        running = true
        log("Starting ${fullName}")
        synchronized(this) {
            println("Engine started")
        }
    }
    
    void stop() {
        running = false
        log("Stopping ${fullName}")
    }
    
    // Operator overloading - fixed map construction
    def plus(Car other) {
        return new Car(
            make: "${this.make}-${other.make}",
            model: "${this.model}-${other.model}",
            year: Math.max(this.year, other.year)
        )
    }
    
    // Property accessors
    private BigDecimal _price
    
    void setPrice(BigDecimal price) {
        if (price < 0) {
            throw new IllegalArgumentException("Price cannot be negative")
        }
        this._price = price
    }
    
    BigDecimal getPrice() {
        return _price
    }
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
    
    @Override
    String toString() {
        return code
    }
}

// Category class with explicit return
class StringExtensions {
    static String truncate(String self, Integer length) {
        return self.size() <= length ? self : self[0..<length] + "..."
    }
}

// Builder pattern class with proper return statements
class EmailBuilder {
    private Map email = [:]
    
    EmailBuilder to(String recipient) {
        email.to = recipient
        return this
    }
    
    EmailBuilder from(String sender) {
        email.from = sender
        return this
    }
    
    EmailBuilder subject(String subject) {
        email.subject = subject
        return this
    }
    
    EmailBuilder body(@DelegatesTo(StringBuilder) Closure body) {
        def builder = new StringBuilder()
        body.delegate = builder
        body.resolveStrategy = Closure.DELEGATE_FIRST
        body()
        email.body = builder.toString()
        return this
    }
    
    Map build() {
        return email.clone() as Map
    }
}

// Main execution with proper closure syntax
def main() {
    use(StringExtensions) {
        def description = "This is a very long description that needs truncating"
        println(description.truncate(20))
    }
    
    def car = new Car(make: "Tesla", model: "Model S")
    car.start()
    
    def email = new EmailBuilder()
        .to("recipient@example.com")
        .from("sender@example.com")
        .subject("Test Email")
        .body {
            append("Hello\n")
            append("This is a test email.\n")
            append("Regards")
        }
        .build()
    
    println(email)
}

// Script execution with proper binding check
if (this.binding.hasVariable('main')) {
    main()
}