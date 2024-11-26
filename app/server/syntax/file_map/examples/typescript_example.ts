// Type imports
import type { Request, Response, NextFunction } from 'express';

// Interface definitions
interface DataProcessor<T> {
    process(data: T): Promise<T>;
    validate(data: T): boolean;
}

// Type aliases
type Result<T> = {
    data: T;
    error?: string;
    metadata: Record<string, unknown>;
};

// Enum with string values
enum Status {
    PENDING = 'pending',
    ACTIVE = 'active',
    COMPLETED = 'completed',
    FAILED = 'failed'
}

// Union type
type ValidationResult = 
    | { valid: true; data: unknown }
    | { valid: false; errors: string[] };

// Intersection type
type AdminUser = User & {
    permissions: string[];
    role: 'admin';
};

// Mapped type
type Readonly<T> = {
    readonly [P in keyof T]: T[P];
};

// Utility type
type Partial<T> = {
    [P in keyof T]?: T[P];
};

// Class with decorators

// Decorator factory
function validate(target: any, propertyKey: string, descriptor: PropertyDescriptor) {
    const originalMethod = descriptor.value;
    descriptor.value = function(...args: any[]) {
        if (this.validate(args[0])) {
            return originalMethod.apply(this, args);
        }
        throw new Error('Validation failed');
    };
    return descriptor;
}

@logger
class User {
    @required
    private name: string;

    @email
    private email: string;

    @format('YYYY-MM-DD')
    private createdAt: Date;

    constructor(name: string, email: string) {
        this.name = name;
        this.email = email;
        this.createdAt = new Date();
    }

    // Method decorator
    @validate
    public updateEmail(newEmail: string): void {
        this.email = newEmail;
    }

    // Getter with type guard
    public get isAdmin(): this is AdminUser {
        return 'role' in this && (this as AdminUser).role === 'admin';
    }
}


// Abstract class
abstract class BaseProcessor<T> implements DataProcessor<T> {
    protected status: Status = Status.PENDING;

    abstract process(data: T): Promise<T>;

    validate(data: T): boolean {
        return data !== null && data !== undefined;
    }
}

// Generic class extending abstract class
class StringProcessor extends BaseProcessor<string> {
    async process(data: string): Promise<string> {
        this.status = Status.ACTIVE;
        const result = await this.transform(data);
        this.status = Status.COMPLETED;
        return result;
    }

    private async transform(data: string): Promise<string> {
        return data.toUpperCase();
    }
}

// Function overloads
function process(data: string): Promise<string>;
function process(data: number): Promise<number>;
function process(data: string | number): Promise<string | number> {
    return Promise.resolve(data);
}

// Generic function with constraints
async function validateData<T extends { id: string }>(
    data: T
): Promise<ValidationResult> {
    if (!data.id) {
        return { valid: false, errors: ['ID is required'] };
    }
    return { valid: true, data };
}

// Higher-order function
function withRetry<T>(
    fn: () => Promise<T>,
    retries: number = 3
): () => Promise<T> {
    return async () => {
        let lastError: Error | undefined;
        
        for (let i = 0; i < retries; i++) {
            try {
                return await fn();
            } catch (error) {
                lastError = error as Error;
            }
        }
        
        throw lastError;
    };
}

// Middleware function type
type Middleware = (
    req: Request,
    res: Response,
    next: NextFunction
) => Promise<void>;

// Utility functions with type inference
const createUser = <T extends User>(data: Partial<T>): T => {
    return { ...data } as T;
};

// Async generator function
async function* generateSequence(
    start: number,
    end: number
): AsyncGenerator<number> {
    for (let i = start; i <= end; i++) {
        await new Promise(resolve => setTimeout(resolve, 100));
        yield i;
    }
}

// Example usage
async function main() {
    const processor = new StringProcessor();
    const result = await processor.process('hello');
    
    const user = createUser<AdminUser>({
        name: 'Admin',
        email: 'admin@example.com',
        permissions: ['read', 'write'],
        role: 'admin'
    });
    
    const retryableProcess = withRetry(async () => {
        return await processor.process('retry me');
    });
    
    for await (const num of generateSequence(1, 5)) {
        console.log(num);
    }
}
