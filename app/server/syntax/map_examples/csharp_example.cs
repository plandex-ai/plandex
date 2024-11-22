using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using System.Linq;

// Namespace declaration
namespace ExampleApp
{
    // Interface definition
    public interface IProcessor<T>
    {
        Task<T> ProcessAsync(T input);
        bool Validate(T input);
    }

    // Enum definition
    public enum Status
    {
        Pending,
        Active,
        Completed,
        Failed
    }

    // Delegate declaration
    public delegate void StatusChangedEventHandler(Status oldStatus, Status newStatus);

    // Generic class implementing interface
    public class DataProcessor<T> : IProcessor<T> where T : class
    {
        // Event declaration
        public event StatusChangedEventHandler StatusChanged;

        // Auto-implemented property
        public Status CurrentStatus { get; private set; }

        // Static field
        private static readonly Dictionary<Type, int> _processedItems = new();

        // Constructor
        public DataProcessor()
        {
            CurrentStatus = Status.Pending;
        }

        // Async method implementation
        public async Task<T> ProcessAsync(T input)
        {
            var oldStatus = CurrentStatus;
            CurrentStatus = Status.Active;
            OnStatusChanged(oldStatus, CurrentStatus);

            await Task.Delay(100); // Simulate work

            if (_processedItems.ContainsKey(typeof(T)))
                _processedItems[typeof(T)]++;
            else
                _processedItems[typeof(T)] = 1;

            CurrentStatus = Status.Completed;
            OnStatusChanged(Status.Active, CurrentStatus);

            return input;
        }

        // Interface method implementation
        public bool Validate(T input) => input != null;

        // Protected virtual method
        protected virtual void OnStatusChanged(Status oldStatus, Status newStatus)
        {
            StatusChanged?.Invoke(oldStatus, newStatus);
        }

        // Static method
        public static int GetProcessedCount<TItem>() where TItem : class
        {
            return _processedItems.GetValueOrDefault(typeof(TItem));
        }
    }

    // Record type (C# 9.0+)
    public record Person(string Name, int Age)
    {
        // Property with validation
        public string Email { get; init; } = string.Empty;
    }

    // Extension method
    public static class StringExtensions
    {
        public static int WordCount(this string str)
        {
            return str.Split(new[] { ' ' }, StringSplitOptions.RemoveEmptyEntries).Length;
        }
    }

    // Main program class
    public class Program
    {
        public static async Task Main(string[] args)
        {
            var processor = new DataProcessor<Person>();
            processor.StatusChanged += (old, @new) => 
                Console.WriteLine($"Status changed from {old} to {@new}");

            var person = new Person("John Doe", 30) { Email = "john@example.com" };
            
            if (processor.Validate(person))
            {
                var result = await processor.ProcessAsync(person);
                Console.WriteLine($"Processed person: {result.Name}");
                Console.WriteLine($"Word count in name: {result.Name.WordCount()}");
            }

            Console.WriteLine($"Total processed persons: {DataProcessor<Person>.GetProcessedCount<Person>()}");
        }
    }
}
