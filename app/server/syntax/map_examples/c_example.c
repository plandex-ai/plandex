#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Macro definitions
#define MAX_SIZE 100
#define SQUARE(x) ((x) * (x))

// Type definitions
typedef struct {
    char name[50];
    int age;
} Person;

typedef enum {
    MONDAY,
    TUESDAY,
    WEDNESDAY,
    THURSDAY,
    FRIDAY
} Weekday;

// Global variables
static const double PI = 3.14159;
int globalCounter = 0;

// Function declarations
void printPerson(const Person* p);
int factorial(int n);

// Union example
union Data {
    int i;
    float f;
    char str[20];
};

// Function pointer type
typedef int (*Operation)(int, int);

// Function implementations
int add(int a, int b) {
    return a + b;
}

int subtract(int a, int b) {
    return a - b;
}

void printPerson(const Person* p) {
    printf("Name: %s, Age: %d\n", p->name, p->age);
}

int factorial(int n) {
    if (n <= 1) return 1;
    return n * factorial(n - 1);
}

// Main function
int main() {
    // Local variable declarations
    Person person = {"John Doe", 30};
    union Data data;
    Operation op = add;
    
    // Using structs
    printPerson(&person);
    
    // Using unions
    data.i = 10;
    printf("data.i: %d\n", data.i);
    
    // Using function pointers
    printf("10 + 20 = %d\n", op(10, 20));
    op = subtract;
    printf("10 - 20 = %d\n", op(10, 20));
    
    // Using macros
    printf("Square of 5 is %d\n", SQUARE(5));
    
    // Using enums
    Weekday today = WEDNESDAY;
    printf("Day number: %d\n", today);
    
    // Using global variables
    globalCounter++;
    printf("Global counter: %d\n", globalCounter);
    
    return 0;
}
