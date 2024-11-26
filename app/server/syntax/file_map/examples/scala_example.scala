package example

import scala.concurrent.{Future, ExecutionContext}
import scala.util.{Try, Success, Failure}
import scala.collection.mutable
import scala.annotation.tailrec

// Type alias
type Result[T] = Either[String, T]

// Trait with type parameter
trait DataProcessor[T] {
  def process(data: T): Result[T]
  def validate(data: T): Boolean
}

// Case class with type parameters
case class ProcessingState[T](
  data: T,
  status: ProcessingState.Status,
  timestamp: Long = System.currentTimeMillis()
)

// Companion object with sealed trait
object ProcessingState {
  sealed trait Status
  case object Pending extends Status
  case object Active extends Status
  case object Completed extends Status
  case class Failed(error: String) extends Status
}

// Abstract class
abstract class BaseProcessor[T] extends DataProcessor[T] {
  protected val logger = new Logger(getClass.getName)
  
  // Abstract method
  protected def transform(data: T): T
  
  // Concrete implementation using abstract method
  override def process(data: T): Result[T] = {
    if (!validate(data)) {
      Left("Invalid data")
    } else {
      Try(transform(data)).toEither.left.map(_.getMessage)
    }
  }
}

// Class with type bounds
class NumberProcessor[T <: Number] extends BaseProcessor[T] {
  override def validate(data: T): Boolean = data != null
  
  protected def transform(data: T): T = data
}

// Case class with default and optional parameters
case class User(
  id: String,
  name: String,
  email: String,
  roles: Set[String] = Set.empty,
  metadata: Map[String, String] = Map.empty
)

// Object for utility functions
object Utils {
  def withRetry[T](times: Int)(f: => T): Try[T] = {
    @tailrec
    def attempt(remaining: Int, lastError: Option[Throwable]): Try[T] = {
      if (remaining == 0) {
        Failure(lastError.getOrElse(new RuntimeException("Max retries exceeded")))
      } else {
        Try(f) match {
          case success @ Success(_) => success
          case Failure(error) => attempt(remaining - 1, Some(error))
        }
      }
    }
    attempt(times, None)
  }
}

// Implicit class for extensions
object Implicits {
  implicit class StringOps(val s: String) extends AnyVal {
    def isValidEmail: Boolean = s.matches(".+@.+\\..+")
  }
}

// Trait with self type annotation
trait Logging {
  self: Logger =>
  
  def debug(message: => String): Unit = log("DEBUG", message)
  def info(message: => String): Unit = log("INFO", message)
  def error(message: => String): Unit = log("ERROR", message)
}

// Class using self type trait
class Logger(name: String) extends Logging {
  def log(level: String, message: => String): Unit = {
    println(s"[$level] $name: $message")
  }
}

// Class with type constructor and higher-kinded type
trait Monad[F[_]] {
  def pure[A](a: A): F[A]
  def flatMap[A, B](fa: F[A])(f: A => F[B]): F[B]
  
  def map[A, B](fa: F[A])(f: A => B): F[B] =
    flatMap(fa)(a => pure(f(a)))
}

// Implicit conversions
object Conversions {
  implicit def stringToUser(s: String): User = {
    val parts = s.split(":")
    User(parts(0), parts(1), parts(2))
  }
}

// Trait with path-dependent type
trait Container {
  type Content
  def content: Content
  def transform(f: Content => Content): Container
}

// Class implementing path-dependent type
class Box[T](initial: T) extends Container {
  type Content = T
  def content: T = initial
  def transform(f: T => T): Box[T] = new Box(f(initial))
}

// Main object with application
object Main extends App {
  import Implicits._
  import ExecutionContext.Implicits.global
  
  val processor = new NumberProcessor[java.lang.Integer]
  
  def processAsync[T](data: T)(implicit ec: ExecutionContext): Future[Result[T]] = {
    Future {
      Thread.sleep(100) // Simulate work
      Right(data)
    }
  }
  
  // Pattern matching
  def handleResult[T](result: Result[T]): Unit = result match {
    case Right(value) => println(s"Success: $value")
    case Left(error) => println(s"Error: $error")
  }
  
  // For comprehension
  val computation = for {
    a <- Future(1)
    b <- Future(2)
    c <- Future(a + b)
  } yield c
  
  // Partial function
  val handler: PartialFunction[Throwable, Unit] = {
    case e: IllegalArgumentException => println(s"Invalid argument: ${e.getMessage}")
    case e: Exception => println(s"Other error: ${e.getMessage}")
  }
  
  // Using implicit conversion
  val user: User = "1:John Doe:john@example.com"
  
  // Using type class
  val box = new Box(42)
  val transformed = box.transform(_ * 2)
  
  println(s"Transformed value: ${transformed.content}")
}
