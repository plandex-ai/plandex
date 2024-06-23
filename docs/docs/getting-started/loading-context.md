# Loading Context

After creating a plan, load any relevant files, directories, URLs, or other data into the plan context.

## Loading Files and Directories

To load files or directories into context, use the `plandex load` command:

```bash
plandex load file1.ts file2.ts
plandex load src -r  # Load a directory recursively
```

## Loading URLs

You can also load the content of a URL into context:

```bash
plandex load https://example.com/docs
```

## Loading Notes

To add a note to the context, use the `-n` flag:

```bash
plandex load -n "This is a note."
```

## Loading Images

Plandex supports loading images (PNG, JPEG, non-animated GIF, and WebP formats):

```bash
plandex load image.png
```

## Using the `pdx` Alias

For convenience, you can use the `pdx` alias instead of typing `plandex` for every command:

```bash
pdx load file1.ts file2.ts
```
