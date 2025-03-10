package prompts

const ChangeExplanationPrompt = `
### Action Explanation Format

#### 1. Updating an existing file in context

Prior to any code block that is *updating* an existing file in context, you MUST explain the change in the following format EXACTLY:

---
**Updating ` + "`[file path]`" + `**  
Type: [type]  
Summary: [brief description, symbols/sections being changed]
Replace: [lines to replace/remove]
Context: [describe surrounding code that helps locate the change unambiguously]
Preserve: [symbols/structures/sections to preserve when overwriting entire file]
---

OR if multiple changes are being made to the same file in a single subtask and a single code block, list each change independently like this:

---
**Updating ` + "`[file path]`" + `**  
Change 1.
Type: [type]
Summary: [brief description, symbols/sections being changed]
Replace: [lines to replace/remove]
Context: [describe surrounding code that helps locate the change unambiguously]

Change 2.
Type: [type]
Summary: [brief description, symbols/sections being changed]
Replace: [lines to replace/remove]
Context: [describe surrounding code that helps locate the change unambiguously]

... and so on for each change
---

Include a line break after the initial '**Updating ` + "`[file path]`" + `**' line as well as each of the following fields. Use the exact same spacing and formatting as shown in the above format and in the examples further down.

The Type field MUST be exactly one of these values: 'add', 'prepend', 'append', 'replace', 'remove', or 'overwrite'.

- add 
  - For inserting new code within the file *only*
  - Only use if NO existing code is being changed or removed - otherwise use 'replace' or 'overwrite'
  - If inserting code at the start of the file, use 'prepend' instead
  - If inserting code at the end of the file, use 'append' instead
- prepend 
  - For inserting new code at the start of the file *only*
  - Only use if NO existing code is being changed or removed - otherwise use 'replace' or 'overwrite'
- append 
  - For inserting new code at the end of the file *only*
  - Only use if NO existing code is being changed or removed - otherwise use 'replace' or 'overwrite'
- replace 
  - For replacing existing code within the file *only*
  - Only use if existing code is being replaced by new code. If new code is being added but none is being replaced, use 'add', 'append', or 'prepend' instead
  - If the entire file is being replaced, use 'overwrite' instead
  - If existing code is being removed and nothing new is being added, use 'remove' instead
- remove 
  - For removing existing code within the file *only*
  - Only use if existing code is being removed. If new code is being added but none is being removed, use 'add', 'append', or 'prepend' instead
  - If code is being removed and replaced with new code, use 'replace' instead
- overwrite 
  - For replacing the entire file *only*
  - Only use if the *entire file* is being replaced. If new code is being added but none is being replaced or removed, use 'add', 'append', or 'prepend' instead.


For each Type, follow these validation rules:

- For 'add':
   - Summary MUST briefly describe the new code being added and where it will be inserted
   - Context MUST describe the surrounding code structures that help locate where the new code will be inserted. The context MUST be *OUTSIDE* of the lines that are being added so that it 'anchors' the exact location of the change in the original file.
   - Preserve field must be omitted
   - Remove field must be omitted
  - In the code block, include the anchors identified in the 'Context' field, collapsed with a reference comment if they span more than a few lines, that are immediately before and after the new code being added. Do NOT include large sections of code from the original file that are not being modified when using 'add'; include enough surrounding code to unambiguously locate the change in the original file, and no more.
  - In the code block, DO NOT UNDER ANY CIRCUMSTANCES reproduce the entire original file with the new code added—that's not what 'add' is for. If you're reproducing the entire original file, use 'overwrite' instead.

- For 'prepend':
   - Summary MUST briefly describe the new code being prepended to the start of the file
   - Context MUST identify the first *existing* code structure in the original file (which will NOT be modified) that the new code will be added before
   - Preserve field must be omitted
   - Remove field must be omitted
   - Code block MUST include JUST the first existing code structure in the original file (which will NOT be modified), collapsed with a reference comment if it spans more than a few lines, immediately followed by the new code being prepended. Do NOT include large sections of code from the original file that are not being modified when using 'prepend'.
   - In the code block, DO NOT UNDER ANY CIRCUMSTANCES reproduce the entire original file with the new code prepended—that's not what 'prepend' is for. If you're reproducing the entire original file, use 'overwrite' instead.

- For 'append':
   - Summary MUST briefly describe the new code being appended to the end of the file
   - Context MUST identify the last *existing* code structure in the original file (which will NOT be modified) that the new code will be added after
   - Preserve field must be omitted
   - Remove field must be omitted
   - Code block MUST include JUST the last existing code structure in the original file (which will NOT be modified), collapsed with a reference comment if it spans more than a few lines, immediately followed by the new code being appended. Do NOT include large sections of code from the original file that are not being modified when using 'append'.
   - In the code block, DO NOT UNDER ANY CIRCUMSTANCES reproduce the entire original file with the new code appended—that's not what 'append' is for. If you're reproducing the entire original file, use 'overwrite' instead.

- For 'replace':
   - Summary MUST briefly describe the change
   - Remove field MUST list lines in the original file that are being replaced. Use the exact format: 'lines [startLineNumber]-[endLineNumber]' — e.g. 'lines 10-20' or for a single line, 'line [lineNumber]' — e.g. 'line 10' — DO NOT use any other format, or describe the lines in any other way.
   - Context MUST describe the surrounding code structures that help locate what is being replaced. Context MUST be *OUTSIDE* of the lines that are being replaced so that it 'anchors' the exact location of the change in the original file.
   - Preserve field must be omitted
   - In the code block, include the anchors identified in the 'Context' field, collapsed with a reference comment if they span more than a few lines, that are immediately before and after the lines being replaced. Do NOT include large sections of code from the original file that are not being modified when using 'replace'; include enough surrounding code to unambiguously locate the change in the original file, and no more.
   - Do NOT UNDER ANY CIRCUMSTANCES reproduce the entire original file with the new code added—that's not what 'replace' is for. If you're reproducing the entire original file, use 'overwrite' instead.

- For 'remove':
   - Summary MUST briefly describe the change
   - Remove field MUST list lines in the original file that are being removed. Use the exact format: 'lines [startLineNumber]-[endLineNumber]' — e.g. 'lines 10-20' or for a single line, 'line [lineNumber]' — e.g. 'line 10' — DO NOT use any other format, or describe the lines in any other way.
   - Context MUST describe the surrounding code structures that help locate what is being removed. Context MUST be *OUTSIDE* of the lines that are being removed so that it 'anchors' the exact location of the change in the original file.
   - Preserve field must be omitted
   - In the code block, include the anchors identified in the 'Context' field, collapsed with a reference comment if they span more than a few lines, that are immediately before and after the lines being removed. Do NOT include large sections of code from the original file that are not being modified when using 'remove'; include enough surrounding code to unambiguously locate the change in the original file, and no more.
   - Do NOT UNDER ANY CIRCUMSTANCES reproduce the entire original file with the removed code omitted—that's not what 'remove' is for. If you're reproducing the entire original file, use 'overwrite' instead.

- For 'overwrite':
   - Summary MUST briefly describe the change and list the specific symbols/sections being changed or replaced
   - Context field must be omitted
   - Preserve MUST *exhaustively* list all symbols/sections in the original file that should be included in the final result. Do *NOT* say that you are 'preserving nothing' because you are overwriting the entire file—the point what, if anything, will be *kept the same* from the original file, even though you are overwriting the whole file. Only say that you're preserving nothing if *nothing* will be kept the same from the original file and the new file will be completely new. The point of this field is to ensure that the final result is a *complete* and *correct* replacement of the original file, and that no important code is omitted.
   - Changes with 'overwrite' MUST NOT be combined with other changes in the same code block. An 'overwrite' change MUST be the ONLY change for the code block.

In the Context, Summary, Remove, and Preserve fields, when listing code symbols, list them in a comma-separated list and surround them with backticks. For example, ` + "`foo`,`someFunc`, `someVar`" + `

IMPORTANT: when listing code symbols or structures in the Context, Summary, and Preserve fields, you MUST include the name of the symbol or structure only, *not* the full signature (e.g. don't include the function parameters or return type for a function—just the function name; don't include the type or the 'var/let/const' keywords for a variable—just the variable name, and so on). DO NOT UNDER ANY CIRCUMSTANCES include full function signatures when listing functions. Include *only* the function name.

For example, instead of ` + "`func (state *activeTellStreamState) genPlanDescription() (*db.ConvoMessageDescription, error)`" + `, you should use ` + "`genPlanDescription`" + `. Instead of ` + "`var foo int`" + `, you should use ` + "`foo`" + `.

CRITICAL: The Context field MUST include symbols/structures that are NOT being modified in any way. They must be completely outside of and untouched by the change. They serve as anchors to locate where the change should occur in the file. The purpose is to clearly demonstrate which context immediately *surrounds* the change so that it can be included in the code block that updates the file.

	INCORRECT - symbols in Context are part of the change:
	Summary: Replace implementations of ` + "`foo`, `bar`, and `baz`" + `
  Replace: lines 105-200
	Context: Located between ` + "`foo`" + ` and ` + "`baz`" + `  # Wrong - these are being changed!

	CORRECT - symbols in Context are outside the change:
	Summary: Replace implementations of ` + "`foo`, `bar`, and `baz`" + `
  Replace: lines 105-200
	Context: Located between ` + "`setup`" + ` and ` + "`cleanup`" + ` functions  # Correct - these aren't being changed

Again, the point of the Context field is to identify *anchors* that exist completely *outside* of the bounds of the change in the original file. The Context field is NOT used to identify code that is being *modified* or *replaced* as part of the change, but rather the code immediately *surrounding* the change.

The symbols/structure you mention in the Context field MUST ALSO be *immediately adjacent* to the change in the original file. Do NOT use symbols or structures that are further away from the change and have other code between them and the change.

ALWAYS surround the symbols/structures you mention in the Context field with backticks. Do NOT leave them out.

Furthermore, every symbol/structure you mention in the Context field ABSOLUTELY MUST be included in the code block that updates the file. Do NOT UNDER ANY CIRCUMSTANCES omit any of these symbols/structures from the code block. Use reference comments to avoid repeating code that is not changing.

Keep the explanation as succinct as possible while still following all of the above rules.

You ABSOLUTELY MUST use this template EXACTLY as described above. DO NOT CHANGE THE FORMATTING OR WORDING IN ANY WAY! DO NOT OMIT ANY FIELDS FROM THE EXPLANATION AS DESCRIBED ABOVE.

Example explanations:

**Updating ` + "`server/api/client.go`" + `**
Type: add
Summary: Add new ` + "`doRequest`" + ` method to ` + "`Client`" + ` struct after the constructor method
Context: Located between ` + "`NewClient`" + ` constructor and ` + "`getUser`" + ` method

**Updating ` + "`server/types/api.go`" + `**
Type: replace
Summary: Replace implementation of ` + "`extractName`" + ` function with new version using ` + "`xml.Decoder`" + `
Replace: lines 8-15
Context: Located between ` + "`validateName`" + ` and ` + "`formatName`" + ` functions

**Updating ` + "`cli/cmd/update.go`" + `**
Type: overwrite
Summary: Replace implementations of ` + "`updateCmd`" + `, ` + "`runUpdate`" + `, and ` + "`validateUpdate`" + ` functions with new versions
Preserve: ` + "`updateFlags`" + ` struct and ` + "`defaultTimeout`" + ` constant

**Updating ` + "`server/config/init.go`" + `**
Type: prepend
Summary: Add new ` + "`validateConfig`" + ` function at start of file
Context: Will be placed before the ` + "`init`" + ` function
 
**Updating ` + "`server/models/user.go`" + `**
Type: append  
Summary: Add new ` + "`cleanupUserData`" + ` function at end of file
Context: Will be placed after the ` + "`validateUser`" + ` function

**Updating ` + "`server/handlers/auth.go`" + `**
Type: remove
Summary: Remove unused ` + "`validateLegacyTokens`" + ` function and its helper ` + "`checkTokenFormat`" + `
Replace: lines 25-85
Context: Located between ` + "`parseAuthHeader`" + ` and ` + "`validateJWT`" + ` functions

*

If multiple changes are being made to the same file in a single subtask, you MUST ALWAYS combine them into a SINGLE code block. Do NOT use multiple code blocks for multiple changes to the same file.

When writing the explanation for multiple changes that will be included in a single code block, list each change independently like this:

**Updating  + "server/handlers/auth.go" + **
Change 1. 
  Type: remove
  Summary: Remove unused ` + "`validateLegacyTokens`" + ` function and its helper ` + "`checkTokenFormat`" + `
  Replace: lines 25-85
  Context: Located between ` + "`parseAuthHeader`" + ` and ` + "`validateJWT`" + ` functions

Change 2.
  Type: append
  Summary: Append just-removed ` + "`checkTokenFormat`" + ` function to the end of the file
  Replace: lines 8-15
  Context: The last code structure is ` + "`finalizeAuth`" + ` function
  
When outputting a compound explanation in the above format, it is CRITICAL that you still only output a SINGLE code block. Do NOT output multiple code blocks.

*

Again, ALL code structures/symbols that are mentioned in the Context field MUST be included as *anchors* in the code block that updates the file. If you are inserting new code between [structure 1] and [structure 2], then you MUST include both [structure 1] and [structure 2] as anchors in the code block that updates the file. Include *anchors* from the Context field so that the change is clearly positioned in the file between sections of code that are *not* being modified.

At the same time, you MUST NOT reproduce large sections of code from the original file that are not changing. You MUST use reference comments "// ... existing code ..." to avoid reproducing large sections of code from the original file that are not changing.

If you are using functions that are not being modified as anchors, then include the function signatures and closing braces, but use a reference comment for the function bodies. Here is an example:

If you are using functions that are not being modified as anchors, then include the function signatures and closing braces, but use a reference comment for the function bodies. Here is an example:

If your change description is:

**Updating ` + "`server/api/users.go`" + `**  
Type: replace
Summary: Replace implementation of ` + "`validateUser`" + ` function to add role and permission validation
Replace: lines 10-20
Context: Located between ` + "`parseUser`" + ` and ` + "`updateUser`" + ` functions

Then your code block MUST look like:

---
// ... existing code ...

func (api *API) parseUser(input []byte) (*User, error) {
    // ... existing code ...
}

func (api *API) validateUser(user *User) error {
    // Validate basic fields
    if user.ID == "" {
        return errors.New("user ID is required")
    }
    if user.Email == "" {
        return errors.New("email is required")
    }

    // New validation for roles
    if len(user.Roles) == 0 {
        return errors.New("user must have at least one role")
    }
    for _, role := range user.Roles {
        if !isValidRole(role) {
            return fmt.Errorf("invalid role: %s", role)
        }
    }

    // New validation for permissions
    for _, permission := range user.Permissions {
        if !isValidPermission(permission) {
            return fmt.Errorf("invalid permission: %s", permission)
        }
    }
    
    return nil
}

func (api *API) updateUser(user *User) error {
    // ... existing code ...
}

// ... existing code ...
---

Notice how:
- The anchor functions 'parseUser' and 'updateUser' are included with their full signatures
- Their bodies are replaced with '// ... existing code ...' since they aren't being modified
- The new 'validateUser' implementation is included in full since it's the actual change
- The file starts and ends with '// ... existing code ...' comments since this change is in the middle of the file
- There's a comment indicating we're replacing the existing implementation

*

❌ INCORRECT - Context symbols missing from code block:
**Updating ` + "`sound.py`" + `**
Type: add
Summary: Add ` + "`debug_status`" + ` method to ` + "`Engine`" + ` class
Context: Located in the ` + "`Engine`" + ` class, right after the ` + "`__init__`" + ` method and right before the ` + "`cleanup`" + ` method

- sound.py:
<PlandexBlock lang="python" path="sound.py">
# ... existing code ...

def debug_status(self):
    """Print debug information about the sound engine state."""
    print("Sound engine debug info")
    
# ... existing code ...
</PlandexBlock>

✅ CORRECT - Context symbols included in code block:
**Updating ` + "`sound.py`" + `**
Type: add
Summary: Add ` + "`debug_status`" + ` method to ` + "`Engine`" + ` class
Context: Located in the ` + "`Engine`" + ` class, after the ` + "`cleanup`" + ` method

- sound.py:
<PlandexBlock lang="python" path="sound.py">
# ... existing code ...

class Engine:
  def __init__(self):
    # ... existing code ...

  def debug_status(self):
      """Print debug information about the sound engine state."""
      print("Sound engine debug info")

  def cleanup(self):
    # ... existing code ...
    
# ... existing code ...
</PlandexBlock>

*

As you can see, in the correct example, every symbol/structure mentioned in the Context field is included in the code block, unambiguously locating the change.

*

If a file is being *updated* and the above explanation does *not* indicate that the file is being *overwritten* or that the change is being prepended to the *start* of the file, then the code block ABSOLUTELY ALWAYS MUST begin with an "... existing code ..." comment to account for all the code before the change. It is EXTREMELY IMPORTANT that you include this comment when it is needed—it must not be omitted.

If a file is being *updated* and the above explanation does *not* indicate that the file is being *overwritten* or that the change is being appended to the *end* of the file, then the code block ABSOLUTELY ALWAYS MUST end with an "... existing code ..." comment to account for all the code after the change. It is EXTREMELY IMPORTANT that you include this comment when it is needed—it must not be omitted.

Again, unless a file is being fully ovewritten, or the change either starts at the *absolute start* of the file or ends at the *absolute end* of the file, IT IS ABSOLUTELY CRITICAL that the file both BEGINS with an "... existing code ..." comment and ENDS with an "... existing code ..." comment.

If a file must begin with an "... existing code ..." comment according to the above rules, then there MUST NOT be any code before the initial "... existing code ..." comment.

If a file must end with an "... existing code ..." comment according to the above rules, then there MUST NOT be any code after the final "... existing code ..." comment.

Again, if the change *does not* end at the *absolute end* of the file, then the LAST LINE of the code block MUST be an "... existing code ..." comment. Ending the code block like this:

---
// ... existing code ...

func (a *Api) NewMethod() {
  callExistingMethod()
}

func (a *Api) LoadContext(planId, branch string, req                      
  shared.LoadContextRequest) (*shared.LoadContextResponse, *shared.ApiError) {
  // ... existing code ...                                                  
}
---

is NOT CORRECT, because the last line is not an "... existing code ..." comment—it is rather the '}' closing bracket of the function. Instead, it must be:

---
// ... existing code ...

func (a *Api) NewMethod() {
  callExistingMethod()
}

func (a *Api) LoadContext(planId, branch string, req                      
  shared.LoadContextRequest) (*shared.LoadContextResponse, *shared.ApiError) {
  // ... existing code ...                                                  
}

// ... existing code ...
---

Now the final line is an "... existing code ..." comment, which is correct.

*

If the explanation states that it will overwrite the entire file, then the code block that updates the file MUST include the ENTIRE file *with no reference or removal comments* and no necessary code omitted. Include *all* code from both the original file and the intended change merged together correctly. Do NOT omit any code from the original file unless the specific intention of the task is to replace or remove that code. Ensure that all symbols/sections mentioned in the 'Preserve' field are included in the code block that updates the file. *MAKE THE CODE BLOCK AS LONG AS NECESSARY TO INCLUDE THE **ENTIRE** FILE.* If the file is too long to fit within a single code block or a single response, *do not* use the 'overwrite' type. Use another type to make a more specific change.

Do NOT overwrite the entire file for very large files that cannot fit within a single response.

*

If the explanation includes a 'Preserve' field, be absolutely certain that the corresponding code block does *not* remove or replace any of the code listed in the 'Preserve' field.

---

Example of an explanation that includes multiple changes to the same file, with a *single* code block:

**Updating  + "server/handlers/auth.go" + **
Change 1. 
  Type: remove
  Summary: Remove  + "validateLegacyTokens" +  and  + "checkTokenFormat" +  (original file lines 25-35).
  Context: Located between  + "parseAuthHeader" +  and  + "validateJWT" +  functions
Change 2.
  Type: append
  Summary: Append a new  + "checkTokenFormatV2" +  function at the end of the file
  Context: The last code structure is  + "finalizeAuth" +  function

- server/handlers/auth.go:
<PlandexBlock lang="go" path="server/handlers/auth.go">
// ... existing code ...

func parseAuthHeader() { 
  // ... existing code ... 
}

// Plandex: removed code

func validateJWT() { 
  // ... existing code ... 
}

func finalizeAuth() { 
  // ... existing code ... 
}

func checkTokenFormatV2(header string) bool {
  // new code for updated token checking
  return header != ""
}

// ... existing code ...
</PlandexBlock>

*

Remember, when outputting a compound explanation in the above format, it is CRITICAL that you still only output a SINGLE code block.

❌ INCORRECT - Including too much of the file with append

**Updating ` + "`server/models/user.go`" + `**
Type: append
Summary: Add new ` + "`validateUserEmail`" + ` function at the end of file
Context: Will be placed after the ` + "`isAdmin`" + ` function

- server/models/user.go:
<PlandexBlock lang="go" path="server/models/user.go">
package models

import (
  "errors"
  "strings"
)

type User struct {
  ID    string
  Name  string
  Email string
  Role  string
}

func NewUser(name, email string) *User {
  return &User{
      Name:  name,
      Email: email,
  }
}

func (u *User) isAdmin() bool {
  return u.Role == "admin"
}

func (u *User) validateUserEmail() error {
  if u.Email == "" {
      return errors.New("email cannot be empty")
  }
  if !strings.Contains(u.Email, "@") {
      return errors.New("invalid email format")
  }
  return nil
}
</PlandexBlock>

✅ CORRECT - Proper append example

**Updating ` + "`server/models/user.go`" + `**
Type: append
Summary: Add new ` + "`validateUserEmail`" + ` function at the end of file
Context: Will be placed after the ` + "`isAdmin`" + ` function

- server/models/user.go:
<PlandexBlock lang="go" path="server/models/user.go">
// ... existing code ...

func (u *User) isAdmin() bool {
  // ... existing code ...
}

func (u *User) validateUserEmail() error {
  if u.Email == "" {
      return errors.New("email cannot be empty")
  }
  if !strings.Contains(u.Email, "@") {
      return errors.New("invalid email format")
  }
  return nil
}
</PlandexBlock>

❌ INCORRECT - Reproducing too much of the file with prepend

**Updating ` + "`server/handlers/users.go`" + `**
Type: prepend
Summary: Add imports and package declaration at the beginning of the file
Context: Will be placed before the ` + "`UserHandler`" + ` struct definition

- server/handlers/users.go:
<PlandexBlock lang="go" path="server/handlers/users.go">
package handlers

import (
  "encoding/json"
  "net/http"
  "github.com/example/app/models"
  "github.com/example/app/utils"
)

type UserHandler struct {
  UserService *models.UserService
}

func NewUserHandler(service *models.UserService) *UserHandler {
  return &UserHandler{
      UserService: service,
  }
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
  // ... existing code ...
}
</PlandexBlock>

✅ CORRECT - Proper prepend example

**Updating ` + "`server/handlers/users.go`" + `**
Type: prepend
Summary: Add imports and package declaration at the beginning of the file
Context: Will be placed before the ` + "`UserHandler`" + ` struct definition

- server/handlers/users.go:
<PlandexBlock lang="go" path="server/handlers/users.go">
package handlers

import (
  "encoding/json"
  "net/http"
  "github.com/example/app/models"
  "github.com/example/app/utils"
)

type UserHandler struct {
  // ... existing code ...
}
</PlandexBlock>

❌ INCORRECT - Using overwrite when replace would be better

**Updating ` + "`server/config/defaults.go`" + `**
Type: overwrite
Summary: Update the ` + "`NewDefaultConfig`" + ` function to change default timeout
Preserve: ` + "`ConfigVersion`" + ` constant, ` + "`DefaultConfig`" + ` struct

- server/config/defaults.go:
<PlandexBlock lang="go" path="server/config/defaults.go">
package config

const ConfigVersion = "1.0.0"

type DefaultConfig struct {
    Port        int
    Host        string
    LogLevel    string
    MaxConn     int
    Timeout     int
    EnableCache bool
}

func NewDefaultConfig() *DefaultConfig {
    return &DefaultConfig{
        Port:        8080,
        Host:        "localhost",
        LogLevel:    "info",
        MaxConn:     100,
        Timeout:     60, // Changed from 30 to 60
        EnableCache: true,
    }
}
</PlandexBlock>

✅ CORRECT - Using replace instead of overwrite for a small change

**Updating ` + "`server/config/defaults.go`" + `**
Type: replace
Summary: Update the ` + "`NewDefaultConfig`" + ` function to change default timeout
Replace: lines 15-24
Context: Located between ` + "`DefaultConfig`" + ` struct definition and end of file

- server/config/defaults.go:
<PlandexBlock lang="go" path="server/config/defaults.go">
// ... existing code ...

type DefaultConfig struct {
    Port        int
    Host        string
    LogLevel    string
    MaxConn     int
    Timeout     int
    EnableCache bool
}

func NewDefaultConfig() *DefaultConfig {
    return &DefaultConfig{
        Port:        8080,
        Host:        "localhost",
        LogLevel:    "info",
        MaxConn:     100,
        Timeout:     60, // Changed from 30 to 60
        EnableCache: true,
    }
}

// ... existing code ...
</PlandexBlock>

✅ CORRECT - Proper use of overwrite

**Updating ` + "`server/config/defaults.go`" + `**
Type: overwrite
Summary: Replace entire file with new implementation of ` + "`DefaultConfig`" + ` and add new ` + "`ValidateConfig`" + ` function
Preserve: ` + "`ConfigVersion`" + ` constant

- server/config/defaults.go:
<PlandexBlock lang="go" path="server/config/defaults.go">
package config

const ConfigVersion = "1.0.0"

type DefaultConfig struct {
  Port        int
  Host        string
  LogLevel    string
  MaxConn     int
  Timeout     int
  EnableCache bool
}

func NewDefaultConfig() *DefaultConfig {
  return &DefaultConfig{
      Port:        8080,
      Host:        "localhost",
      LogLevel:    "info",
      MaxConn:     100,
      Timeout:     30,
      EnableCache: true,
  }
}

func ValidateConfig(cfg *DefaultConfig) error {
  if cfg.Port <= 0 {
      return errors.New("port must be positive")
  }
  if cfg.Host == "" {
      return errors.New("host cannot be empty")
  }
  return nil
}
</PlandexBlock>

❌ INCORRECT - Vague Context that doesn't specify exact location

**Updating ` + "`server/api/auth.go`" + `**
Type: add
Summary: Add new ` + "`validateToken`" + ` helper function
Context: Located in the auth package

- server/api/auth.go:
<PlandexBlock lang="go" path="server/api/auth.go">
package auth

import (
  "errors"
  "strings"
  "time"
)

func validateToken(token string) (bool, error) {
  if token == "" {
      return false, errors.New("token cannot be empty")
  }
  parts := strings.Split(token, ".")
  if len(parts) != 3 {
      return false, errors.New("invalid token format")
  }
  return true, nil
}
</PlandexBlock>

✅ CORRECT - Proper use of Context field with anchors

**Updating ` + "`server/api/auth.go`" + `**
Type: add
Summary: Add new ` + "`validateToken`" + ` helper function after the imports
Context: Located between the import statements and the ` + "`AuthHandler`" + ` struct definition

- server/api/auth.go:
<PlandexBlock lang="go" path="server/api/auth.go">
// ... existing code ...

import (
  "errors"
  "strings"
  "time"
)

func validateToken(token string) (bool, error) {
  if token == "" {
      return false, errors.New("token cannot be empty")
  }
  parts := strings.Split(token, ".")
  if len(parts) != 3 {
      return false, errors.New("invalid token format")
  }
  return true, nil
}

type AuthHandler struct {
  // ... existing code ...
}

// ... existing code ...
</PlandexBlock>

❌ INCORRECT - Multiple code blocks for changes to the same file

**Updating ` + "`server/handlers/users.go`" + `**
Type: add
Summary: Add new ` + "`validateUserInput`" + ` helper function
Context: Located between the import statements and the ` + "`UserHandler`" + ` struct definition

- server/handlers/users.go:
<PlandexBlock lang="go" path="server/handlers/users.go">
// ... existing code ...

import (
  "encoding/json"
  "errors"
  "net/http"
  "github.com/example/app/models"
)

func validateUserInput(user *models.User) error {
  if user.Name == "" {
      return errors.New("name cannot be empty")
  }
  if user.Email == "" {
      return errors.New("email cannot be empty")
  }
  return nil
}

type UserHandler struct {
  // ... existing code ...
}

// ... existing code ...
</PlandexBlock>

**Updating ` + "`server/handlers/users.go`" + `**
Type: replace
Summary: Update ` + "`CreateUser`" + ` method to use the new validation function
Replace: lines 25-35
Context: Located between the ` + "`UserHandler`" + ` struct definition and the ` + "`GetUser`" + ` method

- server/handlers/users.go:
<PlandexBlock lang="go" path="server/handlers/users.go">
// ... existing code ...

type UserHandler struct {
  // ... existing code ...
}
  
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
  var user models.User
  if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
  }
  
  if err := validateUserInput(&user); err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
  }
  
  if err := h.UserService.Create(&user); err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }
  
  w.WriteHeader(http.StatusCreated)
  json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
  // ... existing code ...
}

// ... existing code ...
</PlandexBlock>

✅ CORRECT - Multiple changes to the same file with a single code block

**Updating ` + "`server/handlers/users.go`" + `**
Change 1.
  Type: add
  Summary: Add new ` + "`validateUserInput`" + ` helper function
  Context: Located between the import statements and the ` + "`UserHandler`" + ` struct definition

Change 2.
  Type: replace
  Summary: Update ` + "`CreateUser`" + ` method to use the new validation function
  Replace: lines 25-35
  Context: Located between the ` + "`UserHandler`" + ` struct definition and the ` + "`GetUser`" + ` method

- server/handlers/users.go:
<PlandexBlock lang="go" path="server/handlers/users.go">
// ... existing code ...

import (
  "encoding/json"
  "errors"
  "net/http"
  "github.com/example/app/models"
)

func validateUserInput(user *models.User) error {
  if user.Name == "" {
      return errors.New("name cannot be empty")
  }
  if user.Email == "" {
      return errors.New("email cannot be empty")
  }
  return nil
}

type UserHandler struct {
  UserService *models.UserService
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
  var user models.User
  if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
  }
  
  if err := validateUserInput(&user); err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
  }
  
  if err := h.UserService.Create(&user); err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
  }
  
  w.WriteHeader(http.StatusCreated)
  json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
  // ... existing code ...
}

// ... existing code ...
</PlandexBlock>

---

#### 2. Creating a new file

Prior to any code block that is *creating a new file*, you MUST explain the change in the following format EXACTLY:

---
**Creating ` + "`[file path]`" + `**  
Type: new file  
Summary: [brief description of the new file]
---

Include a line break after the initial '**Creating ` + "`[file path]`" + `**' line as well as each of the following fields. Use the exact same spacing and formatting as shown in the above format and in the examples further down.

The Type field MUST be exactly 'new file'.
The Summary field MUST briefly describe the new file and its purpose.

Do NOT include the 'Context' or 'Preserve' fields when creating a new file. Just the 'Type' and 'Summary' fields are required.

You ABSOLUTELY MUST use this template EXACTLY as described above.

Example explanation for a *new file*:

**Creating ` + "`server/handlers/auth.go`" + `**
Type: new file
Summary: Add new ` + "`auth`" + ` handler in the ` + "`server/handlers`" + ` directory

- server/handlers/auth.go:
<PlandexBlock lang="go" path="server/handlers/auth.go">
package handlers

func (api *API) authHandler(w http.ResponseWriter, r *http.Request) {
  authHeader := r.Header.Get("Authorization")
  if authHeader == "" {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
  }

  valid := validateAuthHeader(authHeader)
  if !valid {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
  }

  session, err := api.sessionStore.Get(r, "session")
  if err != nil {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
  }

  response := &http.Response{
    StatusCode: http.StatusOK,
    Body:       io.NopCloser(strings.NewReader("OK")),
  }

  json.NewEncoder(w).Encode(response)
}
</PlandexBlock>

*

For new files: 
  - You MUST ALWAYS include the *entire file* in the code block. Do not omit any code from the file.
  -  Do NOT use placeholder code or comments like '// implement authentication here' to indicate that the file is incomplete. Implement *all* functionality.
  - Do NOT use reference comments like '// ... existing code ...'. Those are only used for updating existing files and *never* when creating new files.
  - Include the *entire file* in the code block.

`
