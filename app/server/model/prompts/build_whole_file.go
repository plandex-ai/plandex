package prompts

func GetWholeFilePrompt(filePath, preBuildState, changesFile, changesDesc, comments string) string {
	s := getBuildPromptHead(filePath, preBuildState, changesDesc, changesFile)

	s += "## Comments\n\n"

	if comments != "" {
		s += comments + "\n\n"
	} else {
		s += CommentClassifierPrompt + "\n\n"
	}

	s += WholeFilePrompt
	return s
}

const WholeFilePrompt = `
## Whole File 

Output the *entire merged file* with the *proposed updates* correctly applied. ALL reference comments will be replaced by the appropriate code from the *original file*. You will correctly merge the code from the *original file* with the *proposed updates* and output the entire file.

ALL identified reference comments MUST be replaced by the appropriate code from the *original file*. You MUST correctly merge the code from the *original file* with the *proposed updates* and output the *entire* resulting file. The resulting file MUST NOT include any reference comments.

The resulting file MUST be syntactically and semantically correct. All code structures must be properly balanced.

The full resulting file should be output within a <PlandexWholeFile> element, like this:

<PlandexWholeFile>
  package main

  import "logger"

  function main() {
    logger.info("Hello, world!");
    exec()
  }
</PlandexWholeFile>

Do NOT include line numbers in the <PlandexWholeFile> element. Do NOT include reference comments in the <PlandexWholeFile> element. Output the ENTIRE file, no matter how long it is, with NO EXCEPTIONS. Include the resulting file *only* with no other text. Do NOT wrap the file output in triple backticks or any other formatting, except for the <PlandexWholeFile> element tags.

Do NOT include any additional text after the <PlandexWholeFile> element. The output must end after </PlandexWholeFile>. DO NOT use the string <PlandexWholeFile> anywhere else in the output. ONLY use it to start the <PlandexWholeFile> element.

Do NOT UNDER ANY CIRCUMSTANCES *remove or change* any code that is not part of the changes in the *proposed updates*. ALL OTHER code from the *original file* must be reproduced *exactly* as it is in the *original file*. Do NOT remove comments, logging statements, commented out code, or anything else that is not part of the changes in the *proposed updates*. Your job is *only* to *apply* the changes in the *proposed updates* to the *original file*, not to make additional changes of *any kind*.
`
