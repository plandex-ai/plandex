// Package declaration
package example

// Imports
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.*
import java.time.LocalDateTime
import kotlin.properties.Delegates

// Interface with generic type parameter
interface DataProcessor<T> {
    suspend fun process(data: T): Result<T>
    fun validate(data: T): Boolean
}

// Sealed class for representing states
sealed class ProcessingState<out T> {
    object Loading : ProcessingState<Nothing>()
    data class Success<T>(val data: T) : ProcessingState<T>()
    data class Error(val exception: Throwable) : ProcessingState<Nothing>()
}

// Data class with default parameters
data class User(
    val id: String,
    val name: String,
    val email: String,
    val roles: Set<Role> = emptySet(),
    val createdAt: LocalDateTime = LocalDateTime.now()
)

// Enum class with properties and function
enum class Role(val permission: Int) {
    ADMIN(0xFF) {
        override fun toString() = "Administrator"
    },
    USER(0x0F) {
        override fun toString() = "Regular User"
    },
    GUEST(0x00) {
        override fun toString() = "Guest User"
    }
}

// Object declaration (Singleton)
object Configuration {
    const val API_VERSION = "1.0.0"
    val defaultTimeout = 5000L
    
    fun getConfig(key: String) = config[key]
    
    private val config = mutableMapOf<String, Any>()
}

// Class with companion object and delegation
class UserProcessor : DataProcessor<User> {
    // Companion object with factory method
    companion object {
        fun create(): UserProcessor = UserProcessor()
    }
    
    // Property delegation
    private var processingCount: Int by Delegates.observable(0) { _, old, new ->
        println("Processing count changed from $old to $new")
    }
    
    // Property with custom getter
    val isActive: Boolean
        get() = processingCount > 0
    
    // Suspending function implementation
    override suspend fun process(data: User): Result<User> = runCatching {
        processingCount++
        validateEmail(data.email)
        data
    }.also {
        processingCount--
    }
    
    // Regular function implementation
    override fun validate(data: User): Boolean =
        data.email.isNotBlank() && data.name.isNotBlank()
    
    // Extension function
    private fun String.isValidEmail(): Boolean =
        matches(Regex("^[A-Za-z0-9+_.-]+@(.+)$"))
    
    // Inline function with reified type parameter
    inline fun <reified T> logType() =
        println("Processing type: ${T::class.simpleName}")
    
    // Private function using extension function
    private fun validateEmail(email: String) {
        require(email.isValidEmail()) { "Invalid email format" }
    }
}

// Higher-order function with function type parameter
fun <T> withRetry(
    times: Int = 3,
    action: suspend () -> T
): suspend () -> T = {
    var lastException: Exception? = null
    repeat(times) { attempt ->
        try {
            return@withRetry action()
        } catch (e: Exception) {
            lastException = e
            println("Attempt ${attempt + 1} failed: ${e.message}")
        }
    }
    throw lastException ?: IllegalStateException("All attempts failed")
}

// Coroutine scope extension
fun CoroutineScope.processUsers(users: List<User>): Flow<ProcessingState<User>> = flow {
    val processor = UserProcessor.create()
    
    emit(ProcessingState.Loading)
    
    users.forEach { user ->
        processor.process(user)
            .onSuccess { emit(ProcessingState.Success(it)) }
            .onFailure { emit(ProcessingState.Error(it)) }
    }
}

// Extension property
val User.displayName: String
    get() = "$name (${email})"

// Main function demonstrating usage
suspend fun main() = coroutineScope {
    val users = listOf(
        User("1", "John Doe", "john@example.com"),
        User("2", "Jane Smith", "jane@example.com", setOf(Role.ADMIN))
    )
    
    launch {
        processUsers(users)
            .collect { state ->
                when (state) {
                    is ProcessingState.Loading -> println("Processing started")
                    is ProcessingState.Success -> println("Processed: ${state.data.displayName}")
                    is ProcessingState.Error -> println("Error: ${state.exception.message}")
                }
            }
    }
}
