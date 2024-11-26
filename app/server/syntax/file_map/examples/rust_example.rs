use std::{
    collections::{HashMap, HashSet},
    sync::{Arc, Mutex},
    time::{Duration, SystemTime},
};

use tokio::sync::mpsc;
use serde::{Deserialize, Serialize};

// Type alias
type Result<T> = std::result::Result<T, Box<dyn std::error::Error + Send + Sync>>;

// Constants
const MAX_RETRIES: u32 = 3;
const DEFAULT_TIMEOUT: Duration = Duration::from_secs(5);

// Custom error type
#[derive(Debug, thiserror::Error)]
pub enum ProcessError {
    #[error("Validation failed: {0}")]
    ValidationError(String),
    
    #[error("Processing failed: {0}")]
    ProcessingError(String),
    
    #[error(transparent)]
    Other(#[from] Box<dyn std::error::Error + Send + Sync>),
}

// Trait definition
#[async_trait::async_trait]
pub trait DataProcessor<T> {
    async fn process(&self, data: T) -> Result<T>;
    fn validate(&self, data: &T) -> bool;
}

// Struct with lifetime parameter and generic type
#[derive(Debug)]
pub struct ProcessorState<'a, T> {
    name: &'a str,
    data: T,
    created_at: SystemTime,
}

// Enum with different variants
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum Status {
    Pending,
    Active { started_at: SystemTime },
    Completed { result: String },
    Failed { error: String },
}

// Struct implementing trait
pub struct ItemProcessor {
    items: Arc<Mutex<HashSet<String>>>,
    status: Status,
    tx: mpsc::Sender<String>,
}

// Trait implementation
#[async_trait::async_trait]
impl DataProcessor<String> for ItemProcessor {
    async fn process(&self, data: String) -> Result<String> {
        if !self.validate(&data) {
            return Err(Box::new(ProcessError::ValidationError("Invalid data".into())));
        }
        Ok(data.to_uppercase())
    }

    fn validate(&self, data: &String) -> bool {
        !data.is_empty()
    }
}

// Struct with derive macros
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub name: String,
    pub max_items: usize,
    #[serde(default)]
    pub timeout: Option<Duration>,
}


// Implementation with associated types
pub trait Storage {
    type Item;
    type Error;

    fn store(&mut self, item: Self::Item) -> std::result::Result<(), Self::Error>;
    fn retrieve(&self, id: &str) -> std::result::Result<Option<Self::Item>, Self::Error>;
}

// Struct implementing trait with associated types
pub struct MemoryStorage {
    data: HashMap<String, Vec<u8>>,
}

impl Storage for MemoryStorage {
    type Item = Vec<u8>;
    type Error = std::io::Error;

    fn store(&mut self, item: Self::Item) -> std::result::Result<(), Self::Error> {
        self.data.insert(String::from("default"), item);
        Ok(())
    }

    fn retrieve(&self, id: &str) -> std::result::Result<Option<Self::Item>, Self::Error> {
        Ok(self.data.get(id).cloned())
    }
}

// Async main function
#[tokio::main]
async fn main() -> Result<()> {
    let (tx, mut rx) = mpsc::channel(100);
    let mut processor = ItemProcessor::new(tx);

    // Spawn background task
    tokio::spawn(async move {
        while let Some(item) = rx.recv().await {
            println!("Received: {}", item);
        }
    });

    // Process items
    processor.add_item("test".into()).await?;
    
    let config = Config {
        name: "test".into(),
        max_items: 100,
        timeout: Some(DEFAULT_TIMEOUT),
    };

    println!("Config: {:?}", config);
    
    Ok(())
}
