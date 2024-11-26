import Foundation

// Protocol definitions
protocol DataProcessor {
    associatedtype Input
    associatedtype Output
    
    func process(_ input: Input) async throws -> Output
    func validate(_ input: Input) -> Bool
}

// Error type
enum ProcessingError: LocalizedError {
    case invalidInput(String)
    case processingFailed(String)
    
    var errorDescription: String? {
        switch self {
        case .invalidInput(let reason): return "Invalid input: \(reason)"
        case .processingFailed(let reason): return "Processing failed: \(reason)"
        }
    }
}

// Property wrapper
@propertyWrapper
struct Validated<T> {
    private var value: T
    private let validator: (T) -> Bool
    
    var wrappedValue: T {
        get { value }
        set {
            guard validator(newValue) else {
                fatalError("Invalid value")
            }
            value = newValue
        }
    }
    
    init(wrappedValue: T, validator: @escaping (T) -> Bool) {
        guard validator(wrappedValue) else {
            fatalError("Invalid initial value")
        }
        self.value = wrappedValue
        self.validator = validator
    }
}

// Actor for thread-safe state management
actor ProcessingState {
    private(set) var processedCount: Int = 0
    private var status: Status = .pending
    
    enum Status {
        case pending
        case processing
        case completed
        case failed(Error)
    }
    
    func incrementCount() {
        processedCount += 1
    }
    
    func updateStatus(_ newStatus: Status) {
        status = newStatus
    }
}

// Generic struct with where clause
struct Queue<Element> where Element: Sendable {
    private var elements: [Element] = []
    private let lock = NSLock()
    
    mutating func enqueue(_ element: Element) {
        lock.lock()
        defer { lock.unlock() }
        elements.append(element)
    }
    
    mutating func dequeue() -> Element? {
        lock.lock()
        defer { lock.unlock() }
        return elements.isEmpty ? nil : elements.removeFirst()
    }
}

// Class inheritance and protocol conformance
class StringProcessor: DataProcessor {
    typealias Input = String
    typealias Output = String
    
    private let state = ProcessingState()
    
    @Validated(validator: { !$0.isEmpty })
    private var currentInput: String = "default"
    
    func process(_ input: String) async throws -> String {
        guard validate(input) else {
            throw ProcessingError.invalidInput("String is empty")
        }
        
        await state.updateStatus(.processing)
        
        // Simulate processing
        try await Task.sleep(nanoseconds: 1_000_000_000)
        let result = input.uppercased()
        
        await state.incrementCount()
        await state.updateStatus(.completed)
        
        return result
    }
    
    func validate(_ input: String) -> Bool {
        !input.isEmpty
    }
}

// Extension with async sequence
extension StringProcessor: AsyncSequence, AsyncIteratorProtocol {
    typealias Element = String
    
    func makeAsyncIterator() -> StringProcessor {
        self
    }
    
    func next() async throws -> String? {
        try await process(currentInput)
    }
}

// Result builders
@resultBuilder
struct ArrayBuilder<T> {
    static func buildBlock(_ components: T...) -> [T] {
        components
    }
}

// Function using result builder
func makeArray<T>(@ArrayBuilder<T> content: () -> [T]) -> [T] {
    content()
}

// Async main function demonstrating usage
@main
struct Example {
    static func main() async throws {
        let processor = StringProcessor()
        var queue = Queue<String>()
        
        // Using result builder
        let inputs = makeArray {
            "Hello"
            "World"
            "Swift"
        }
        
        // Process inputs
        for input in inputs {
            queue.enqueue(input)
        }
        
        // Process queue
        while let input = queue.dequeue() {
            do {
                let result = try await processor.process(input)
                print("Processed: \(result)")
            } catch {
                print("Error: \(error.localizedDescription)")
            }
        }
        
        // Using async sequence
        for try await result in processor.prefix(3) {
            print("Async sequence result: \(result)")
        }
    }
}
