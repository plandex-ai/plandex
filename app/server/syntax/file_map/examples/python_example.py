#!/usr/bin/env python3
from __future__ import annotations

import asyncio
import dataclasses
import enum
from abc import ABC, abstractmethod
from datetime import datetime
from functools import wraps
from typing import (
    Any, AsyncIterator, Callable, ClassVar, Dict, Generic, 
    List, Optional, Protocol, TypeVar, Union
)

# Type variable definitions
T = TypeVar('T')
K = TypeVar('K')
V = TypeVar('V')

# Protocol definition
class Processable(Protocol):
    def process(self) -> None: ...
    def validate(self) -> bool: ...

# Enum definition
class Status(enum.Enum):
    PENDING = "pending"
    ACTIVE = "active"
    COMPLETED = "completed"
    FAILED = "failed"

    def __str__(self) -> str:
        return self.value

# Dataclass with frozen and slots options
@dataclasses.dataclass(frozen=True, slots=True)
class UserCredentials:
    username: str
    email: str
    created_at: datetime = dataclasses.field(default_factory=datetime.now)

# Abstract base class
class BaseProcessor(ABC, Generic[T]):
    def __init__(self) -> None:
        self._items: List[T] = []
        self._processed_count: int = 0

    @abstractmethod
    async def process_item(self, item: T) -> None:
        pass

    @property
    def processed_count(self) -> int:
        return self._processed_count

# Decorator definition
def log_execution(func: Callable) -> Callable:
    @wraps(func)
    async def wrapper(*args: Any, **kwargs: Any) -> Any:
        print(f"Executing {func.__name__}")
        try:
            result = await func(*args, **kwargs)
            print(f"Completed {func.__name__}")
            return result
        except Exception as e:
            print(f"Error in {func.__name__}: {e}")
            raise
    return wrapper

# Class implementing abstract base class and protocol
class DataProcessor(BaseProcessor[UserCredentials], Processable):
    # Class variable
    DEFAULT_BATCH_SIZE: ClassVar[int] = 100

    def __init__(self, batch_size: Optional[int] = None) -> None:
        super().__init__()
        self.batch_size = batch_size or self.DEFAULT_BATCH_SIZE
        self._status = Status.PENDING

    # Property with getter and setter
    @property
    def status(self) -> Status:
        return self._status

    @status.setter
    def status(self, value: Status) -> None:
        if not isinstance(value, Status):
            raise ValueError("Status must be a Status enum value")
        self._status = value

    # Context manager methods
    async def __aenter__(self) -> DataProcessor:
        self.status = Status.ACTIVE
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        self.status = Status.COMPLETED if exc_type is None else Status.FAILED

    # Generator method
    async def process_batch(self) -> AsyncIterator[List[UserCredentials]]:
        for i in range(0, len(self._items), self.batch_size):
            batch = self._items[i:i + self.batch_size]
            yield batch
            await asyncio.sleep(0.1)

    # Implementation of abstract method
    @log_execution
    async def process_item(self, item: UserCredentials) -> None:
        if not self.validate():
            raise ValueError("Processor is not in a valid state")
        self._items.append(item)
        self._processed_count += 1

    # Implementation of protocol method
    def process(self) -> None:
        if not self._items:
            raise ValueError("No items to process")
        self.status = Status.ACTIVE

    def validate(self) -> bool:
        return self.status != Status.FAILED

# Custom exception
class ProcessingError(Exception):
    def __init__(self, message: str, item: Any) -> None:
        self.item = item
        super().__init__(f"Error processing {item}: {message}")

# Async main function
async def main() -> None:
    async with DataProcessor(batch_size=10) as processor:
        # Create test data
        user = UserCredentials(
            username="test_user",
            email="test@example.com"
        )

        try:
            await processor.process_item(user)
            
            async for batch in processor.process_batch():
                print(f"Processing batch of {len(batch)} items")
                
        except ProcessingError as e:
            print(f"Processing failed: {e}")

if __name__ == "__main__":
    asyncio.run(main())
