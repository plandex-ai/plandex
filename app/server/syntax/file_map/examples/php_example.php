<?php

declare(strict_types=1);

namespace Example;

use DateTime;
use Exception;
use InvalidArgumentException;
use JsonSerializable;
use Psr\Log\LoggerInterface;

// Interface definition
interface DataProcessor
{
    public function process(mixed $data): mixed;
    public function validate(mixed $data): bool;
}

// Trait definition
trait Loggable
{
    protected ?LoggerInterface $logger = null;

    public function setLogger(LoggerInterface $logger): void
    {
        $this->logger = $logger;
    }

    protected function log(string $message, array $context = []): void
    {
        $this->logger?->info($message, $context);
    }
}

// Abstract class
abstract class Entity implements JsonSerializable
{
    protected DateTime $createdAt;
    protected ?DateTime $updatedAt = null;

    public function __construct()
    {
        $this->createdAt = new DateTime();
    }

    abstract public function validate(): bool;

    public function jsonSerialize(): mixed
    {
        return [
            'createdAt' => $this->createdAt->format('c'),
            'updatedAt' => $this->updatedAt?->format('c'),
        ];
    }
}

// Enum definition (PHP 8.1+)
enum Status: string
{
    case PENDING = 'pending';
    case ACTIVE = 'active';
    case COMPLETED = 'completed';
    case FAILED = 'failed';

    public function label(): string
    {
        return match($this) {
            self::PENDING => 'Pending',
            self::ACTIVE => 'Active',
            self::COMPLETED => 'Completed',
            self::FAILED => 'Failed',
        };
    }
}

// Class implementing interface and using trait
class User extends Entity implements DataProcessor
{
    use Loggable;

    private static int $instanceCount = 0;

    public function __construct(
        private string $name,
        private string $email,
        private Status $status = Status::PENDING,
        private array $metadata = []
    ) {
        parent::__construct();
        self::$instanceCount++;
    }

    public static function getInstanceCount(): int
    {
        return self::$instanceCount;
    }

    // Property getter with validation
    public function getEmail(): string
    {
        return $this->email;
    }

    // Property setter with validation
    public function setEmail(string $email): void
    {
        if (!filter_var($email, FILTER_VALIDATE_EMAIL)) {
            throw new InvalidArgumentException('Invalid email format');
        }
        $this->email = $email;
        $this->updatedAt = new DateTime();
    }

    // Magic method implementation
    public function __get(string $name)
    {
        return $this->metadata[$name] ?? null;
    }

    public function __set(string $name, mixed $value): void
    {
        $this->metadata[$name] = $value;
    }

    // Interface method implementations
    public function process(mixed $data): mixed
    {
        if (!is_array($data)) {
            throw new InvalidArgumentException('Data must be an array');
        }

        $this->log('Processing user data', ['user' => $this->name]);

        foreach ($data as $key => $value) {
            $this->metadata[$key] = $value;
        }

        $this->status = Status::COMPLETED;
        return $this;
    }

    public function validate(mixed $data): bool
    {
        return is_array($data) && !empty($data);
    }

    // Abstract method implementation
    public function validate(): bool
    {
        return !empty($this->name) && !empty($this->email);
    }

    // Method using arrow functions (PHP 7.4+)
    public function getMetadataValues(): array
    {
        return array_map(
            fn($value) => is_array($value) ? json_encode($value) : (string)$value,
            $this->metadata
        );
    }

    // Implementation of JsonSerializable
    public function jsonSerialize(): mixed
    {
        return [
            ...parent::jsonSerialize(),
            'name' => $this->name,
            'email' => $this->email,
            'status' => $this->status->value,
            'metadata' => $this->metadata,
        ];
    }
}

// Custom exception
class ProcessingException extends Exception
{
    public function __construct(
        string $message = "",
        private ?string $errorCode = null,
        int $code = 0,
        ?Throwable $previous = null
    ) {
        parent::__construct($message, $code, $previous);
    }

    public function getErrorCode(): ?string
    {
        return $this->errorCode;
    }
}

// Anonymous class usage
$validator = new class {
    public function validateUser(User $user): bool
    {
        return $user->validate();
    }
};

// Example usage
try {
    $user = new User("John Doe", "john@example.com");
    $user->process(['role' => 'admin', 'preferences' => ['theme' => 'dark']]);
    
    // Using magic methods
    $user->customField = 'custom value';
    echo $user->customField;
    
    // JSON serialization
    echo json_encode($user, JSON_PRETTY_PRINT);
    
} catch (Exception $e) {
    echo "Error: " . $e->getMessage();
}
