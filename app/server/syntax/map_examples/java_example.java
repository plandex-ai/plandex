package example;

import java.util.*;
import java.util.concurrent.*;
import java.util.function.*;
import java.util.stream.*;
import java.time.*;

// Generic interface with type bounds
interface DataProcessor<T extends Comparable<T>> {
    CompletableFuture<T> processAsync(T input);
    boolean validate(T input);
}

// Enum with methods and fields
enum Status {
    PENDING("P"),
    ACTIVE("A"),
    COMPLETED("C"),
    FAILED("F");

    private final String code;

    Status(String code) {
        this.code = code;
    }

    public String getCode() {
        return code;
    }
}

// Abstract class with generic type
abstract class BaseEntity<ID> {
    protected ID id;
    protected LocalDateTime createdAt;
    protected LocalDateTime updatedAt;

    public abstract void validate();
}

// Record type (Java 16+)
record UserDTO(
    String name,
    String email,
    Set<String> roles
) {}

// Annotation definition
@interface Audited {
    String value() default "";
    boolean required() default true;
}

// Main class with various Java features
public class Example extends BaseEntity<UUID> implements DataProcessor<String> {
    // Static fields and initialization block
    private static final int MAX_RETRIES = 3;
    private static final Map<String, Integer> CACHE;
    
    static {
        CACHE = new ConcurrentHashMap<>();
    }

    // Instance fields with different access modifiers
    private final Queue<String> queue;
    protected Status status;
    @Audited
    public String name;

    // Constructor with builder pattern
    private Example(Builder builder) {
        this.queue = new LinkedBlockingQueue<>();
        this.status = Status.PENDING;
        this.name = builder.name;
    }

    // Builder static class
    public static class Builder {
        private String name;

        public Builder name(String name) {
            this.name = name;
            return this;
        }

        public Example build() {
            return new Example(this);
        }
    }

    // Interface implementation
    @Override
    public CompletableFuture<String> processAsync(String input) {
        return CompletableFuture.supplyAsync(() -> {
            try {
                queue.offer(input);
                return input.toUpperCase();
            } catch (Exception e) {
                throw new CompletionException(e);
            }
        });
    }

    @Override
    public boolean validate(String input) {
        return input != null && !input.isEmpty();
    }

    // Abstract method implementation
    @Override
    public void validate() {
        if (name == null || name.isEmpty()) {
            throw new IllegalStateException("Name is required");
        }
    }

    // Generic method with wildcards
    public <T extends Comparable<? super T>> List<T> sort(Collection<T> items) {
        return items.stream()
                   .sorted()
                   .collect(Collectors.toList());
    }

    // Method with functional interfaces
    public void processItems(
        List<String> items,
        Predicate<String> filter,
        Consumer<String> processor
    ) {
        items.stream()
             .filter(filter)
             .forEach(processor);
    }

    // Exception class
    public static class ProcessingException extends RuntimeException {
        public ProcessingException(String message) {
            super(message);
        }
    }

    // Main method demonstrating usage
    public static void main(String[] args) {
        var example = new Builder()
            .name("Test Example")
            .build();

        // Lambda and method reference usage
        List<String> items = Arrays.asList("a", "b", "c");
        example.processItems(
            items,
            String::isEmpty,
            System.out::println
        );

        // Stream API usage
        Map<Status, Long> statusCounts = items.stream()
            .map(s -> Status.PENDING)
            .collect(Collectors.groupingBy(
                status -> status,
                Collectors.counting()
            ));

        // CompletableFuture with exception handling
        example.processAsync("test")
              .thenApply(String::toLowerCase)
              .exceptionally(throwable -> {
                  System.err.println("Error: " + throwable.getMessage());
                  return "";
              });

        // Try-with-resources and Optional usage
        try (var scanner = new Scanner(System.in)) {
            Optional.of(scanner.nextLine())
                    .filter(example::validate)
                    .ifPresent(example::processAsync);
        }
    }
}
