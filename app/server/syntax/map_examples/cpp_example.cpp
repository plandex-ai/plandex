#include <iostream>
#include <memory>
#include <vector>
#include <string>
#include <functional>

// Template class
template<typename T>
class Container {
public:
    void add(T item) { items.push_back(item); }
    const std::vector<T>& getItems() const { return items; }
private:
    std::vector<T> items;
};

// Abstract base class
class Animal {
public:
    virtual ~Animal() = default;
    virtual void makeSound() const = 0;
protected:
    std::string name;
};

// Derived class with virtual inheritance
class Dog : virtual public Animal {
public:
    Dog(const std::string& dogName) { name = dogName; }
    void makeSound() const override {
        std::cout << name << " says: Woof!" << std::endl;
    }
};

// Namespace example
namespace Utils {
    // Function template
    template<typename T>
    T max(T a, T b) {
        return (a > b) ? a : b;
    }

    // Lambda function stored in variable
    const auto printer = [](const std::string& msg) {
        std::cout << "Message: " << msg << std::endl;
    };
}

// Smart pointer and move semantics example
class Resource {
public:
    Resource(const std::string& data) : data_(data) {
        std::cout << "Resource constructed" << std::endl;
    }
    ~Resource() {
        std::cout << "Resource destroyed" << std::endl;
    }
    Resource(Resource&& other) noexcept : data_(std::move(other.data_)) {}
    std::string getData() const { return data_; }
private:
    std::string data_;
};

// Static member variable and function
class Counter {
public:
    static int getCount() { return count; }
    Counter() { ++count; }
    ~Counter() { --count; }
private:
    static int count;
};
int Counter::count = 0;

// Friend function example
class Box {
    friend std::ostream& operator<<(std::ostream& os, const Box& box);
public:
    Box(int w, int h) : width(w), height(h) {}
private:
    int width;
    int height;
};

std::ostream& operator<<(std::ostream& os, const Box& box) {
    return os << "Box(" << box.width << "x" << box.height << ")";
}

// Main function
int main() {
    // Smart pointer usage
    auto resource = std::make_unique<Resource>("Hello");
    std::cout << resource->getData() << std::endl;

    // Template class usage
    Container<int> numbers;
    numbers.add(1);
    numbers.add(2);

    // Polymorphism
    std::unique_ptr<Animal> dog = std::make_unique<Dog>("Rex");
    dog->makeSound();

    // Template function
    std::cout << "Max: " << Utils::max(10, 20) << std::endl;

    // Lambda function
    Utils::printer("Hello from lambda!");

    // Counter static example
    Counter c1, c2;
    std::cout << "Count: " << Counter::getCount() << std::endl;

    // Friend function
    Box box(10, 20);
    std::cout << box << std::endl;

    return 0;
}
