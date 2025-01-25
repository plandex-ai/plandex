package prompts

const UpdateFormatPrompt = `
You ABSOLUTELY MUST *ONLY* USE the comment "// ... existing code ..." (or the equivalent with the appropriate comment symbol in another programming language) if you are *updating* an existing file. DO NOT use it when you are creating a new file. A new file has no existing code to refer to, so it must not include this kind of reference.

DO NOT UNDER ANY CIRCUMSTANCES use language other than "... existing code ..." in a reference comment. This is EXTREMELY IMPORTANT. You must use the appropriate comment symbol for the language you are using, followed by "... existing code ..." *exactly* (without the quotes).

When updating a file, you MUST NOT include large sections of the file that are not changing. Output ONLY code that is changing and code that is necessary to understand the changes, the code structure, and where the changes should be applied. Example:

---

// ... existing code ...

function fooBar() {
  // ... existing code ...

  updateState();
}

// ... existing code ...

---

ALWAYS show the full structure of where a change should be applied. For example, if you are adding a function to an existing class, do it like this:

---
// ... existing code ...

class FooBar {
  // ... existing code ...

  updateState() {
    doSomething();
  }
}
---

DO NOT leave out the class definition. This applies to other code structures like functions, loops, and conditionals as well. You MUST make it unambiguously clear where the change is being applied by including all relevant code structure.

Below, if the 'update' function is being added to an existing class, you MUST NOT leave out the code structure like this:

---
// ... existing code ...

  update() {
    doSomething();
  }

// ... existing code ...
---

You ABSOLUTELY MUST include the full code structure like this:

---
// ... existing code ...

class FooBar {
  // ... existing code ...

  update() {
    doSomething();
  }
}
---

ALWAYS use the above format when updating a file. You MUST NEVER UNDER ANY CIRCUMSTANCES leave out an "... existing code ..." reference for a section of code that is *not* changing and is not reproduce in the code block in order to demonstrate the structure of the code and where the change will occur.

If you are updating a file type that doesn't use comments (like JSON or plain text), you *MUST still use* '// ... existing code ...' to denote where the reference should be placed. Do NOT omit references for sections of code that are not changing regardless of the file type. Remember, this *ONLY* applies to files that don't use comments. For ALL OTHER file types, you MUST use the correct comment symbol for the language and the section of code where the reference should be placed.

For example, in a JSON file:

---

{
  // ... existing code ...

  "foo": "bar",

  "baz": {
    // ... existing code ...

    "arr": [
      // ... existing code ...
      "val"
    ]
  },

  // ... existing code ...
}
---

You MUST NOT omit references in JSON files or similar file types. You MUST NOT leave out "// ... existing code ..." references for sections of code that are not changing, and you MUST use these references to make the structure of the code unambiguously clear.

Even if you are only updating a single property or value, you MUST use the appropriate references where needed to make it clear exactlywhere the change should be applied.

If you have a JSON file like:

---
{                                                                         
  "name": "vscode-plandex",                                  
  "contributes": {                                                        
    "languages": [{                                                       
      "id": "plandex",
    }],
    "commands": [
      {
        "command": "plandex.tellPlandex",
      }
    ],
    "keybindings": [{
      "command": "plandex.showFilePicker",
    }]
  },
  "scripts": {
    "compile": "webpack",
  },
}
---

And you are adding a new key to the 'contributes' object, you MUST NOT output a code block like:

---

{
  "contributes": {
    "languages": [{
      "id": "plandex",
    }],
    "grammars": [
      {
        "language": "plandex",
      }
    ]
  }
}

---

The problem with the above is that it leaves out *multiple* reference comments that *MUST* be present. It is EXTREMELY IMPORTANT that you include these references.

You also MUST NOT output a code block like:

---

{
  // ... existing code ...

  "contributes":{
    "languages": [{
      "id": "plandex",
    }],
    "grammars": [
      {
        "language": "plandex",
      }
    ]
  }
}

---

This ONLY includes a single reference comment for the code that isn't changing *before* the change. It *forgets* the code that isn't changing *after* the change, as well the remaining properties of the 'contributes' object.
                 
Here's the CORRECT way to output the code block for this change:

---

{
  // ... existing code ...

  "contributes": {
    "languages": [{
      "id": "plandex",
    }],
    "grammars": [
      {
        "language": "plandex",
      }
    ]

    // ... existing code ...
  },

  // ... existing code ...
}
---

You MUST NOT omit references for code that is not changing—this applies to EVERY level of the structural hierarchy. No matter how deep the nesting, every level MUST be accounted for with references if it includes code that is not included in the code block and is not changing.

You MUST ONLY use the exact comment "// ... existing code ..." (with the appropriate comment symbol for the programming language) to denote where the reference should be placed.

You MUST NOT use any other form of reference comment. ONLY use "// ... existing code ...".

When reproducing lines of code from the *original file*, you ABSOLUTELY MUST *exactly match* the indentation of the code being referenced. Do NOT alter the indentation of the code being referenced in any way. If the original file uses tabs for indentation, you MUST use tabs for indentation. If the original file uses spaces for indentation, you MUST use spaces for indentation. When you are reproducing a line, you MUST use the exact same number of spaces or tabs for indentation as the original file.

You MUST NOT output multiple references with no changes in between them. DO NOT UNDER ANY CIRCUMSTANCES DO THIS:

---
function fooBar() error {
  log.Println("fooBar")

  // ... existing code ...

  // ... existing code ...

  return nil
}
---

It must instead be:

---
function fooBar() error {
  log.Println("fooBar")

  // ... existing code ...

  return nil
}
---

You MUST ensure that references are clear and can be unambiguously located in the file in terms of both position and structure/depth of nesting. You MUST NOT use references in a way that makes their exact location in the file ambiguous. It must be possible from the surrounding code to unambiguously and deterministically locate the exact position and depth of nesting of the code that is being referenced. Include as much surrounding code as necessary to achieve this (and no more).

For example, if the original file looks like this:

---
const a = [
  8,
  9,
  10,
  11,
  12,
  13,
  14,
  15,
]
---

you MUST NOT do this:

---
const a = [
  // ... existing code ...
  1,
  5,	
  7,
  // ... existing code ...
]
---

Because it is not unambiguously clear where in the array the new code should be inserted. It could be inserted between any pair of existing elements. The reference comment does not make it clear which, so it is ambiguous. 

The correct way to do it is:

---
const a = [
  // ... existing code ...
  10,
  1,
  5,
  7,
  11,
  // ... existing code ...
]
---

In the above example, the lines with '10' and '11' and included on either side of the new code to make it unambiguously clear exactly where the new code should be inserted.

When using reference comments, you MUST include trailing commas (or similar syntax) where necessary to ensure that when the reference is replace with the new code, ALL the code is perfectly syntactically correct and no comma or other necessary syntax is omitted.

You MUST NOT do this:

---
const a = [
  1,
  5
  // ... existing code ...
]
---

Because it leaves out a necessary trailing comman after the '5'. Instead do this:

---
const a = [
  1,
  5,
  // ... existing code ...
]
---

Reference comments MUST ALWAYS be on their *OWN LINES*. You MUST NEVER include a reference comment on the same line as code.

You MUST NOT do this:

---
const a = [1, 2, /* ... existing code ... */, 4, 5]
---

Instead, rewrite the entire line to include the new code without using a reference comment:

---
const a = [1, 2, 11, 15, 14, 4, 5]
---

You MUST NOT extra newlines around a reference comment unless they are also present in the original file. You ABSOLUTELY MUST be precise about matching newlines with corresponding code in the original file.

If the original file looks like this:

---
package main

import (
  "fmt"
  "os"
)

func main() {
  fmt.Println("Hello, World!")
  exec()
  measure()
  os.Exit(0)
}
---

DO NOT output superfluous newlines before or after reference comments like this:

---

// ... existing code ...

func main() {
  fmt.Println("Hello, World!")
  prepareData()

  // ... existing code ...

}

---

Instead, do this:

---
// ... existing code ...

func main() {
  fmt.Println("Hello, World!")
  prepareData()
  // ... existing code ...
}
---

Note the lack of superfluous newlines before and after the reference comment. There is a newline included between the first '// ... existing code ...' and the 'func main()' line because this newline is present in the original file. There is no newline *before* the first '// ... existing code ...' reference comment because the original file does not have a newline before that comment. Similarly, there is no newline before *or* after the second '// ... existing code ...' reference comment because the original file does not have newlines before or after the code that is being referenced. Newlines are SIGNIFICANT—you must strive to maintain consistent formatting between the original file and the changes in the code block.

*

If code is being removed from a file and not replaced with new code, the removal MUST ALWAYS WITHOUT EXCEPTION be shown in a labelled code block according to your instructions. Use the comment "// Plandex: removed code" (with the appropriate comment symbol for the programming language) to denote the removal. You MUST ALWAYS use this exact comment for any code that is removed and not replaced with new code. DO NOT USE ANY OTHER COMMENT FOR CODE REMOVAL.
    
Do NOT use any other formatting apart from a labelled code block with the comment "// Plandex: removed code" to denote code removal.

Example of code being removed and not replaced with new code:

---
function fooBar() {
  log.Println("called fooBar")
  // Plandex: removed code
}
---

As with reference comments, code removal comments MUST ALWAYS:
  - Be on their own line. They must not be on the same line as any other code.
  - Be on the same line as the code being removed
  - Be surrounded by enough context so that the location and nesting depth of the code being removed is obvious and unambiguous.

Also like reference comments, you MUST NOT use multiple code removal comments in a row without any code in between them.

You MUST NOT do this:

---
function fooBar() {
  // Plandex: removed code
  // Plandex: removed code
  exec()
}
---

Instead, do this:

---
function fooBar() {	
  // Plandex: removed code
  exec()
}
---

You MUST NOT use reference comments and removal comments together in an ambiguous way. Do NOT do this:

---
function fooBar() {
  log.Println("called fooBar")
  // Plandex: removed code
  // ... existing code ...
}
---

Above, there is no way to know deterministically which code should be removed. Instead, include context that makes it clear and unambiguous which code should be removed:

---
function fooBar() {
  log.Println("called fooBar")
  // Plandex: removed code
  exec()
  // ... existing code ...
}
---

By including the 'exec()' line from the original file, it becomes clear and unambiguous that all code between the 'log.Println("called fooBar")' line and the 'exec()' line is being removed.

*

When *replacing* code from the original file with *new code*, you MUST make it unambiguously clear exactly which code is being replaced by including surrounding context. Include as much surrounding context as necessary to achieve this (and no more).

If the original file looks like this:

---
class FooBar {	
  func baz() {
    log.Println("baz")
  }

  func bar() {
    log.Println("bar")
    sendMessage("bar")
    reportSentMessage()
  }
  
  func qux() {
    log.Println("qux")
  }

  func axon() {
    log.Println("axon")
    escapeFromBar()
    runAway()
  }

  func tango() {
    log.Println("tango")
  }
}
---

and you are replacing the 'qux()' method with a different method, you MUST include enough context so that it is clear and unambiguous which method is being replaced. Do NOT do this:

---
class FooBar {
  // ... existing code ...

  func updatedQux() {
    log.Println("updatedQux")
  }

  // ... existing code ...
}
---

The code above is ambiguous because it could also be *inserting* the 'updatedQux()' method in addition to the 'qux()' method rather than replacing the 'qux()' method. Instead, include enough context so that it is clear and unambiguous which method is being replaced, like this:

---
class FooBar {
  // ... existing code ...

  func bar() {
    // ... existing code ...
  }

  func updatedQux() {
    log.Println("updatedQux")
  }

  func axon() {
    // ... existing code ...
  }
  
  // ... existing code ...
}
---

By including the context before and after the 'updatedQux()'—the 'bar' and 'axon' method signatures—it becomes clear and unambiguous that the 'qux()' method is being *replaced* with the 'updatedQux()' method.

*

When using an "... existing code ..." comment, you must ensure that the lines around the comment which locate the comment in the code exactly the match the lines in the original file and do not change it in subtle ways. For example, if the original file looks like this:

---
{
  "key1": [{
    "subkey1": "value1",
    "subkey2": "value2"
  }],
  "key2": "value2"
}
---

DO NOT output a code block like this:

---
{
  "key1": [
    // ... existing code ...
  ],
  "key2": "updatedValue2"
}
---

The problem is that the line '"key1": [{' has been changed to '"key1": [' and the line '}],' has been changed to '],' which makes it difficult to locate these lines in the original file. Instead, do this:

---
{
  "key1": [{
    // ... existing code ...
  }],
  "key2": "updatedValue2"
}
---

Note that the lines around the "... existing code ..." comment exactly match the lines in the original file.

*

When outputting a code block for a change, unless the change begins at the *start* of the file, you ABSOLUTELY MUST include an "... existing code ..." comment prior to the change to account for all the code before the change. Similarly, unless the change goes to the *end* of the file, you ABSOLUTE MUST include an "... existing code ..." comment after the change to account for all the code after the change. It is EXTREMELY IMPORTANT that you include these references and do no leave them out under any circumstances.

For example, if the original file looks like this:

---
package main

import "fmt"

func main() {
  fmt.Println("Hello, World!")
}

func fooBar() {
  fmt.Println("fooBar")
}
---

DO NOT output a code block like this:

---
func main() {
  fmt.Println("Hello, World!")
  fooBar()
}
---

The problem is that the change doesn't begin at the start of the file, and doesn't go to the end of the file, but "... existing code ..." comments are missing from both before and after the change. Instead, do this:

---
// ... existing code ...

func main() {
  fmt.Println("Hello, World!")
  fooBar()
}

// ... existing code ...
---

Now the code before and after the change is accounted for.

Unless you are fully overwriting the entire file, you ABSOLUTELY MUST ALWAYS include at least one "... existing code ..." comment before or after the change to account for all the code before or after the change.

*

When outputting a change to a file, like adding a new function, you MUST NOT include only the new function without including *anchors* from the original file to locate the position of the new code unambiguously. For example, if the original file looks like this:

---
function someFunction() {
  console.log("someFunction")
  const res = await fetch("https://example.com")
  processResponse(res)
  return res
}

function processResponse(res) {
  console.log("processing response")
  callSomeOtherFunction(res)
  return res
}

function yetAnotherFunction() {
  console.log("yetAnotherFunction")
}

function callSomething() {
  console.log("callSomething")
  await logSomething()
  return "something"
}
---

DO NOT output a code block like this:

---
// ... existing code ...

function newFunction() {
  console.log("newFunction")
  const res = await callSomething()
  return res
}

// ... existing code ...
---

The problem is that surrounding context from the original file was not included to clearly indicate *exactly* where the new function is being added in the file. Instead, do this:

---
// ... existing code ...

function processResponse(res) {
  // ... existing code ...
}

function newFunction() {
  console.log("newFunction")
  const res = await callSomething()
  return res
}

// ... existing code ...
---

By including the 'processResponse' function signature from the original code as an *anchor*, the location of the new code can be *unambiguously* located in the original file. It is clear now that the new function is being added immediately after the 'processResponse' function.

It's EXTREMELY IMPORTANT that every code block that is *updating* an existing file includes at least one anchor that maps the lines from the original file to the lines in the code block so that the changes can be unambiguously located in the original file, and applied correctly.

Even if it's unimportant where in the original file the new code should be added and it could be added anywhere, you still *must decide* *exactly* where in the original file the new code should be added and include one or more *anchors* to make the insertion point clear and unambiguous. Do NOT leave out anchors for a file update under any circumstances.

*

When inserting new code between two existing blocks of code in the original file, you MUST include "... existing code ..." comments correctly in order to avoid overwriting sections of existing code. For example, if the original file looks like this:

---

func main() {
  console.log("main")
}

func fooBar() {
  console.log("fooBar")
}

func baz() {
  console.log("baz")
}

func qux() {
  console.log("qux")
}

func quix() {
  console.log("quix")
}

func qwoo() {
  console.log("qwoo")
}

func last() {
  console.log("last")
}

---

DO NOT output a code block like this to demonstrate that new code will be inserted somewhere between the 'fooBar' and 'last' functions:

---
// ... existing code ...

func fooBar() {
  console.log("fooBar")
}

func newCode() {
  console.log("newCode")
}

func last() {
  console.log("last")
}
---

If you want to demonstrate that a new function will be inserted somewhere between the 'fooBar' and 'last' functions, you MUST include "... existing code ..." comments correctly in order to avoid overwriting sections of existing code. Instead, do this to show exactly where the new function will be inserted:

---

// ... existing code ...

func baz() {
  // ... existing code ...
}

func newCode() {
  console.log("newCode")
}

func qux() {
  // ... existing code ...
}

// ... existing code ...


Or this to show that the new function will be inserted *somehwere* between the 'fooBar' and 'last' functions:

---

// ... existing code ...

func fooBar() {
  console.log("fooBar")
}

// ... existing code ...

func newCode() {
  console.log("newCode")
}

// ... existing code ...

func last() {
  console.log("last")
}

---

Either way, you MUST NOT leave out the "... existing code ..." comments for ANY existing code that will remain in the file after the change is applied.

*

When including code from the original file to that is not changing and is intended to be used as an *anchor* to locate the insertion point of the new code, you ABSOLUTELY MUST NOT EVER change the order of the code in the original file. The order of the code in the original file MUST be preserved exactly as it is in the original file unless the proposed change is specifically changing the order of this code.

If you are making multiple changes to the same file in a single code block, you MUST adhere to the order of the original file as closely as possible.

If the original file is:

---
func buck() {
  console.log("buck")
}

func qux() {
  console.log("qux")
}

func fooBar() {
  console.log("fooBar")
}

func baz() {
  console.log("baz")
}

func yup() {
  console.log("yup")
}
---

DO NOT output a code block like this to demonstrate that new code will be inserted between the 'fooBar' and 'baz' functions:

---
// ... existing code ...

func baz() {
  console.log("baz-updated")
}

// ... existing code ...

func qux() {
  console.log("qux-updated")
}

// ... existing code ...

---

The problem is that the order of the 'baz' and 'qux' functions has been changed in the proposed changes unnecessarily. Instead, do this:

---
// ... existing code ...

func qux() {
  console.log("qux-updated")
}

// ... existing code ...

func baz() {
  console.log("baz-updated")
}

// ... existing code ...
---

Now the order of the 'baz' and 'qux' functions is preserved exactly as it is in the original file.

*

When writing an "... existing code ..." comment, you MUST use the correct comment symbol for the programming language. For example, if you are writing a plan in Python, Ruby, or Bash, you MUST use '# ... existing code ...' instead of '// ... existing code ...'. If you're writing HTML, you MUST use '<!-- ... existing code ... -->'. If you're writing jsx, tsx, svelte, or another language where the correct comment symbol(s) depend on where in the code you are, use the appropriate comment symbol(s) for where that comment is placed in the file. If you're in a javascript block of a jsx file, use '// ... existing code ...'. If you're in a markup block of a jsx file, use '{/* ... existing code ... */}'.
`

const UpdateFormatAdditionalExamples = `
Here are some important examples of INCORRECT vs CORRECT file updates:

Example 1 - Adding a new route:

❌ INCORRECT - Replacing instead of inserting:
<PlandexBlock lang="go">
// ... existing code ...

r.HandleFunc(prefix+"/api/users", handlers.ListUsersHandler).Methods("GET")

r.HandleFunc(prefix+"/api/config", handlers.GetConfigHandler).Methods("GET")

// ... existing code ...
</PlandexBlock>
This is wrong because it doesn't show enough context to know what surrounding routes were preserved.

✅ CORRECT - Proper insertion with context:
<PlandexBlock lang="go">
// ... existing code ...

r.HandleFunc(prefix+"/api/users", handlers.ListUsersHandler).Methods("GET")
r.HandleFunc(prefix+"/api/teams", handlers.ListTeamsHandler).Methods("GET")

r.HandleFunc(prefix+"/api/config", handlers.GetConfigHandler).Methods("GET")

r.HandleFunc(prefix+"/api/settings", handlers.GetSettingsHandler).Methods("GET")
r.HandleFunc(prefix+"/api/status", handlers.GetStatusHandler).Methods("GET")

// ... existing code ...
</PlandexBlock>

Example 2 - Adding a method to a class:

❌ INCORRECT - Ambiguous insertion:
<PlandexBlock lang="go">
class UserService {
  // ... existing code ...
  
  async createUser(data) {
    // new method
  }
  
  // ... existing code ...
}
</PlandexBlock>
This is wrong because it doesn't show where exactly the new method should go.

✅ CORRECT - Clear insertion point:
<PlandexBlock lang="go">
class UserService {
  // ... existing code ...
  
  async getUser(id) {
    return await this.db.users.findOne(id)
  }
  
  async createUser(data) {
    return await this.db.users.create(data)
  }
  
  async updateUser(id, data) {
    return await this.db.users.update(id, data)
  }
  
  // ... existing code ...
}
</PlandexBlock>

Example 3 - Adding a configuration section:

❌ INCORRECT - Lost context:
<PlandexBlock lang="json">
{
  "database": {
    "host": "localhost",
    "port": 5432
  },
  "newFeature": {
    "enabled": true,
    "timeout": 30
  }
}
</PlandexBlock>
This is wrong because it dropped existing configuration sections.

✅ CORRECT - Preserved context:
<PlandexBlock lang="json">
{
  // ... existing code ...
  
  "database": {
    "host": "localhost",
    "port": 5432,
    "username": "admin"
  },
  
  "newFeature": {
    "enabled": true,
    "timeout": 30
  },
  
  "logging": {
    "level": "info",
    "file": "app.log"
  }
  
  // ... existing code ...
}
</PlandexBlock>

Key principles demonstrated in these examples:
1. Always show the surrounding context that will be preserved
2. Make insertion points unambiguous by showing adjacent code
3. Never remove existing functionality unless explicitly instructed to do so
4. Use "... existing code ..." comments properly to indicate preserved sections
5. Show enough context to understand the code structure
`
