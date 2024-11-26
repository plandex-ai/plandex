// ES Module imports
import { EventEmitter } from 'events';
import { promisify } from 'util';

// Global constants
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT = 5000;

// Symbol for private properties
const privateState = Symbol('privateState');

// Class using ES6+ features
class DataProcessor extends EventEmitter {
    // Private class field
    #cache = new Map();
    
    // Static class field
    static version = '1.0.0';
    
    // Constructor with parameter destructuring
    constructor({ maxRetries = MAX_RETRIES, timeout = DEFAULT_TIMEOUT } = {}) {
        super();
        this[privateState] = { maxRetries, timeout };
    }
    
    // Async method with error handling
    async processData(data) {
        try {
            const result = await this.#validateAndTransform(data);
            this.emit('processed', result);
            return result;
        } catch (error) {
            this.emit('error', error);
            throw error;
        }
    }
    
    // Private method
    async #validateAndTransform(data) {
        if (!data) throw new Error('Data is required');
        return { ...data, timestamp: Date.now() };
    }
    
    // Generator method
    *iterateCache() {
        for (const [key, value] of this.#cache) {
            yield { key, value };
        }
    }
}


// Decorator function (stage 3 proposal)
function deprecated(target, context) {
    if (context.kind === 'method') {
        const originalMethod = target;
        return function(...args) {
            console.warn(`Warning: ${context.name} is deprecated`);
            return originalMethod.apply(this, args);
        };
    }
}

// Proxy example
const handler = {
    get(target, prop) {
        return prop in target ? target[prop] : 'Property not found';
    }
};

const proxy = new Proxy({}, handler);

// Promise-based utility function
const delay = ms => new Promise(resolve => setTimeout(resolve, ms));

// Async generator function
async function* generateSequence(start, end) {
    for (let i = start; i <= end; i++) {
        await delay(100);
        yield i;
    }
}

// Higher-order function
const memoize = (fn) => {
    const cache = new Map();
    return (...args) => {
        const key = JSON.stringify(args);
        if (cache.has(key)) return cache.get(key);
        const result = fn.apply(this, args);
        cache.set(key, result);
        return result;
    };
};

// Custom error class
class ValidationError extends Error {
    constructor(message, field) {
        super(message);
        this.name = 'ValidationError';
        this.field = field;
    }
}

// Object with getter/setter
const config = {
    _theme: 'light',
    get theme() {
        return this._theme;
    },
    set theme(value) {
        if (!['light', 'dark'].includes(value)) {
            throw new ValidationError('Invalid theme', 'theme');
        }
        this._theme = value;
    }
};

// Array methods and destructuring
const processItems = (items) => {
    const [first, ...rest] = items;
    return rest
        .filter(item => item != null)
        .map(item => ({ ...item, processed: true }))
        .reduce((acc, curr) => {
            acc[curr.id] = curr;
            return acc;
        }, {});
};

// Async/await with Promise.all
const fetchData = async (urls) => {
    try {
        const responses = await Promise.all(
            urls.map(url => fetch(url).then(res => res.json()))
        );
        return responses;
    } catch (error) {
        console.error('Failed to fetch data:', error);
        throw error;
    }
};

// Export statement
export {
    DataProcessor,
    ValidationError,
    processItems,
    fetchData,
    delay,
    memoize,
    config
};
