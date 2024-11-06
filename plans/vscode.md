I want to build a VSCode extension that can do the following:

It should be active when editing any file with a `.pdx` extension.

`.pdx` file should have `.mdx` syntax highlighting and be treated the same as an `.mdx` file by the mdx extension.

It should track some state related to the .pdx file being edited.

It should track them through frontmatter keys at the top of the file:

- the name of the plan (if it exists) and its id
- who created the plan in the format `[name]<email> | [timestamp]`
- who last updated the plan in the same format
- plan collaborators (in the same format as the creator)
- the status of the plan
- it should use YAML (and YAML frontmatter syntax/syntax highlighting) for the frontmatter keys

- it should have `<Contexts></Contexts>` tag at the top (after the frontmatter keys) that lists the plan's context.

- it should then have a `<Prompts></Prompts>` section for the user's prompt

Don't worry about getting this data from the API yet—just use placeholder text.

-

When I type the `@` key, I want to get a list of files from the workspace (relative to the location of the .pdx file) that I can fuzzy search.

When a file is selected, unless the Plandex terminal is already active/visible, it should bring up a terminal tab in the same directory as the .pdx file.

It should run the command `plandex load [path to the selected file]` in the terminal.

-

At the top of the editor, I want a footer action bar with a `Tell Plandex` button—it should have a green 'play' icon.

When the `Tell Plandex` button is clicked, it should bring up a terminal tab in the same directory as the .pdx file, unless the Plandex terminal is already active/visible.

In the terminal, it should run `plandex new` then `plandex tell -f [location of the .pdx file being edited]`

-

Create all necessary files and folders in the `vscode` folder. Create scripts for installing dependencies, running the extension, etc. Again, prefix ALL paths with `vscode/`—you must write all files into this directory.

Write tests for core functionality.

-

Use TypeScript and create the relevant types. Don't use `any` and be strict about types.

For the webpack config file, use typescript as well.
