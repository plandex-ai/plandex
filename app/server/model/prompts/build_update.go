package prompts

import (
	"fmt"

	"github.com/plandex/plandex/shared"
)

func GetUpdatedBuildPrompt(filePath, preBuildState, changesFile, changesDesc string) string {
	preBuildStateWithLineNums := shared.AddLineNums(preBuildState)
	changesWithLineNums := shared.AddLineNums(changesFile)

	s := UpdatedBuildPrompt + "\n\n" + getPreBuildStatePrompt(filePath, preBuildStateWithLineNums) + "\n\n"

	s += fmt.Sprintf("Proposed updates:\n%s\n```\n%s\n```", changesDesc, changesWithLineNums)

	return s
}

var UpdatedBuildPrompt = `
	You are an AI that takes a file and a set of proposed changes to that file and determines exactly how the changes should be applied in order to produce a valid result. To do this you will produce a map of the resulting file, showing which sections remain unchanged from the original file and which sections will be updated by the proposed changes.
	
	In order to do this, you will go through the following process:

	**1.** Identify all *anchors* in the proposed changes and list them all. An anchor a line or group of lines from the proposed changes that can be unambigulously matched with a line or group of lines in the pre-change file. Anchors must be exact matches between the proposed changes and the original file.
	
	For each anchor:
	 - Briefly identify it with a description
	 - Output the starting line number prefixed by 'pdx-' in the proposed changes. If the anchor spans multiple lines, also output the ending line number prefixed by 'pdx-'.
	 - Output the starting line number in the original file that corresponds to the anchor's starting line number in the proposed changes. If the anchor spans multiple lines, also output the ending line number in the original file prefixed by 'pdx-' that corresponds to the anchor's ending line number in the proposed changes.
	 
	Example:

	---
	## Anchors

	1. Signature and opening brace of the 'update' function
  Changes pdx-5 to pdx-6 > Original pdx-456 to pdx-457

	2. 'Commit db transaction' comment
  Changes pdx-10 > Original pdx-456

	3. Closing brace of the 'update' function
  Changes pdx-7 > Original pdx-458

  ---

	**2.** Identify all *references* in the proposed changes. A reference is a comment like: 
	
	- "// ... existing code..."
	- "# Existing code..."
	- "/* ... */"
	- "// Rest of the function..."
	- "<!-- rest of div tag -->"
	- "// ... rest of function ..."
	- "// rest of component..."
	- "# other methods..."
	- "// ... rest of init code..."
	- "// rest of the class..."
	- "// other properties"
	- "// other methods"
	
	These are examples of reference comments, but there are many other variations a reference could take. Identify all reference comments based on the context of the change. Use your judgement to determine what is a reference. A reference must clearly be intented to reference code in the original file for the purpose of making it clear where a change should be applied.

	For each reference:
	  - Briefly identify it with a description 
		- Output the line number of the reference prefixed by 'pdx-'
		- Output the starting line number prefixed by 'pdx-' in the original file that the reference is referring to. If the reference spans multiple lines, also output the ending line number prefixed by 'pdx-' in the original file that the reference is referring to.
	
	Ensure no *anchors* are included in the list of references. The referenced section of the original file must not overlap with any lines that are present in the proposed updates. For example, if there's a reference immediately after an opening brace in the proposed changes, the line with the opening brace must *NOT* be included in the referenced section of the original file. Similarly if there's a reference immediately before a closing brace in the proposed changes, the line with the closing brace must *NOT* be included in the referenced section of the original file since this would produce overlap.

	Example:

	---
	## References

	1. "// ... existing imports..." reference to import statements in original file
  Changes pdx-3 > Original pdx-1 to pdx-10

	2. "// ... rest of the function..." reference to code after the addition of the conditional statment in the 'update' function
  Changes pdx-10 > Original pdx-456 to pdx-460

	---

	**3.** Identify insertions in the proposed changes. An insertion is code from the proposed changes that should be added to the original file *and doesn't replace any existing code in the original file*. Insertions are for *new code*. They are *not* for changes to existing code. They are also *not* for deletions of existing code.
	
	For each insertion:
	  - Briefly identify it with a description
		- Output the starting line of the code that will be insert in the proposed changes prefixed by 'pdx-'. If the insertion spans multiple lines, also output the ending line of the code that will be inserted in the proposed changes prefixed by 'pdx-'.
		- Output the line number in the original file that the new code should be inserted *after* prefixed by 'pdx-'. If the new code should be prepended to the original file, output 'pdx-0'. If the new code should be appended to the original file, output 'pdx-n' where n is the last line of the original file.
	
	Example:

	---
	## Insertions

	1. New imports
  Changes pdx-10 to pdx-15 > Original pdx-0

	2. New validation check
  Changes pdx-30 to pdx-35 > Original pdx-323
	
	---
	
  **4.** Identify replacements in the proposed changes. A replacement is code from the proposed changes that should replace code in the original file. Replacements are for *changes* to existing code that must overwrite existing code in the original file. They are *not* for new code that does not replace any existing code in the original file. They are also *not* for deletions of existing code.

	For each replacement:
	  - Briefly identify it with a description
		- Output the starting line of the code that will be replaced in the proposed changes prefixed by 'pdx-'. If the replacement spans multiple lines, also output the ending line of the code that will be replaced in the proposed changes prefixed by 'pdx-'.
		- Output the starting line number in the original file that the replacement is replacing. If the replacement spans multiple lines, also output the ending line number in the original file prefixed by 'pdx-' that the replacement is replacing.
	
	Example:

	---
	## Replacements

	1. Update the db transaction commit
  Changes pdx-10 to pdx-15 > Original pdx-456 to pdx-461

	2. Improve the validation check
  Changes pdx-30 to pdx-40 > Original pdx-323 to pdx-328
	
	---
	
	**5.** Identify deletions in the proposed changes. A deletion is code that should be removed from the original file. They are marked by comments like:
	- "// Remove the validation check"
	- "# Remove the 'update' function"
	- "<!-- Remove the header -->"
	- "// Remove the footer"
	
	Deletions are for *deletions* of existing code. They are *not* for changes to existing code. They are also *not* for new code that does not replace any existing code in the original file.
	
	Use *deletions* for these kinds of comments rather than replacements so that the deletion comment does not end up in the result.

	For each deletion:
	  - Briefly identify it with a description
		- Output the line number of the deletion in the proposed changes prefixed by 'pdx-'
		- Output the starting line number in the original file that the deletion is deleting. If the deletion spans multiple lines, also output the ending line number in the original file prefixed by 'pdx-' that the deletion is deleting.
	
	Example:

	---
	## Deletions

	1. Remove the old validation check
  Changes pdx-30 > Original pdx-323 to pdx-328

	2. Remove the 'update' function
  Changes pdx-10 > Original pdx-456 to pdx-461
	
	---
	
	**6.** Now that you've identified all the anchors, references, insertions, replacements, and deletions in the proposed changes, you will output a map of the ENTIRE resulting file as a series of sections that refer to either the original file or the proposed changes. Take into account anchors, references, insertions, replacements, and deletions when building the map.
	
	Use xml tags for the sections with these attributes:
	- 'type': 'original' or 'proposed'
	- 'description': a brief description of the section. Be clear, precise, and unambiguous about where the section begins and ends.
	- 'anchor': optional anchor number from step 1. Do NOT repeat an anchor number in multiple sections—each anchor should be unique.
	- 'reference': optional reference number from step 2. Do NOT repeat a reference number in multiple sections—each reference should be unique.
	- 'insertion': optional insertion number from step 3. Do NOT repeat an insertion number in multiple sections—each insertion should be unique.
	- 'replacement': optional replacement number from step 4. Do NOT repeat a replacement number in multiple sections—each replacement should be unique.
	- 'start': the starting line number in the original file or proposed changes
	- 'end': the ending line number in the original file or proposed changes. If the section is a single line, the start and end should be the same.
	- 'newlineBefore': optional boolean. Set to 'true' if a newline should be added *before* the section to preserve the original file's formatting. Omit this if the section already includes all necessary newlines.
	- 'newlineAfter': optional boolean. Set to 'true' if a newline should be added *after* the section to preserve the original file's formatting. Omit this if the section already includes all necessary newlines.

	Follow these rules for the map:
	- Always give anchors, references, inserts, replacements, and deletions their *own sections* in the map. Do not include multiple anchors, references, inserts, replacements, and deletions in a single section.
	- Anchors should always have type "proposed". Even though they are present in both the original file and the proposed changes, always include them as "proposed" sections in the map for consistency. Anchors that are added as "proposed" sections must *NOT* be included in the map as "original" sections.
	- References should always have type "original".
	- Insertions should always have type "proposed".
	- Replacements should always have type "proposed".
	- Lines that have been *deleted* in the 'deletions' list must *NOT* be included in the map.
	- Lines that have been *replaces* in the 'replacements' list must have a corresponding section in the map with type "proposed", and the lines from the original file that have been replaced must *NOT* be included in the map.
	- Lines that have been *inserted* in the 'insertions' list must have a corresponding section in the map with type "proposed". 
	- Every anchor, reference, insertion, and replacement *must* have a corresponding section in the map. 'deletion' sections must *NOT* be included in the map.
	- Do not include the same line of code in multiple sections. Each line should be assigned to *only one* section.
	- Be precise about section boundaries and DO NOT incorrectly duplicate or omit a single line of code.
	- The map, when assembled from all the sections *must* include the *complete* update file with all changes correctly applied, no change omitted, and no references incorrectly included.
	- When producing the map, do not change or omit any lines from the original file unless they are explicitly delete or replaced. This includes comments, whitespace, and newlines.
	- You MUST NOT omit any sections from the original file that were not *explictly* deleted or replaced in the proposed changes.
	- You MUST ensure newlines are preserved from both the original file and the proposed changes. Do NOT add or remove newlines unless it is explicitly intended by the proposed changes. Even if there is a large block of newlines in the original file, you MUST NOT omit them unless they are explicitly deleted in the proposed changes.
	- The start and end of each section *must* be the same line if the section is a single line.
	- The start and end of each section *must* be a line that exists in the original file or proposed changes.
	- The end line of each section must always be equal to or greater than the start line.

	Example:

	---
  ## Result map

	<result>
		<section type="original" description="unchanged import statements prior to the newly added imports" start="pdx-1" end="pdx-10" />
		<section type="proposed" description="new imports" insertion="1" start="pdx-2" end="pdx-10" />
		<section type="original" description="unchanged code before the 'update' function" start="pdx-11" end="pdx-300" newlineBefore="true" />
		<section type="proposed" description="function signature and opening brace of the 'update' function" start="pdx-301" end="pdx-302" anchor="1" newlineAfter="true" />
		<section type="proposed" description="new code added to the 'update' function" insertion="2" start="pdx-1" end="pdx-5" />
		<section type="proposed" description="changes to the loop body in the 'update' function" replacement="1" start="pdx-6" end="pdx-10" />
		<section type="original" description="rest of the unchanged 'update' function" reference="1" start="pdx-11" end="pdx-300" />
		<section type="proposed" description="closing brace of the 'update' function" start="pdx-301" end="pdx-302" anchor="2" newlineBefore="true" />
		<section type="original" description="unchanged code following the 'update' function" reference="2" start="pdx-303" end="pdx-620" />
	</result>
	---

	Do not include any additional text in your final output. Only output the process sections and the result map.
`
