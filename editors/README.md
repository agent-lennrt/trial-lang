# Editor support

This directory contains a TextMate grammar, editor configuration, and setup
notes for the triallang language server.

## Syntax highlighting

[`trial.tmLanguage.json`](trial.tmLanguage.json) is a TextMate grammar
for `.trial` filings (and it tolerates `.deposition` files). It works
anywhere TextMate grammars do:

- **VS Code**: drop a minimal extension folder into
  `~/.vscode/extensions/triallang/` containing the grammar and this
  `package.json`:

  ```json
  {
    "name": "triallang",
    "displayName": "triallang",
    "version": "2.0.0",
    "engines": { "vscode": "^1.0.0" },
    "contributes": {
      "languages": [{
        "id": "trial",
        "extensions": [".trial", ".deposition"],
        "configuration": "./language-configuration.json"
      }],
      "grammars": [{
        "language": "trial",
        "scopeName": "source.trial",
        "path": "./trial.tmLanguage.json"
      }]
    }
  }
  ```

  with [`language-configuration.json`](language-configuration.json)
  beside it (comments and bracket behavior).

- **GitHub rendering**: Linguist ships grammars for registered languages only.
  In a repository that embeds triallang files, add
  `*.trial linguist-language=Text` to that repository's `.gitattributes` if
  they should be excluded from language statistics.

A tree-sitter grammar remains on the roadmap. The TextMate grammar provides
highlighting, and the LSP provides diagnostics, hover text, and completion.

## The language server: `trial counsel`

`trial counsel` speaks LSP 3.x on stdio. Diagnostics use the same Gregor lexer,
parser, and code generator as the CLI. The server also provides hover text that
cites the reference manual and completion for required phrases.

The client must send full-document updates with exactly one text change.
An explicitly empty string clears a document; missing or null text leaves the
last accepted document unchanged. Positions use the LSP default
[UTF-16 encoding](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position).
Columns exclude CRLF line endings, and a character offset past a line clamps
to that line's end.

- **Neovim** (0.10+):

  ```lua
  vim.filetype.add({ extension = { trial = "trial" } })
  vim.api.nvim_create_autocmd("FileType", {
    pattern = "trial",
    callback = function()
      vim.lsp.start({ name = "trial-counsel", cmd = { "trial", "counsel" } })
    end,
  })
  ```

- **VS Code / others**: point any generic LSP client at the command
  `trial counsel` for the language id `trial`.

By default, filings that `INCORPORATE BY REFERENCE` receive syntax-only
diagnostics. Use `trial counsel --canon` to compile documents whose imports all
come from the bundled canon. In Neovim, change the command above to
`cmd = { "trial", "counsel", "--canon" }`.

Canon mode resolves transitive imports and catches errors such as a missing
article, a wrong number of arguments, or a duplicate office. A custom or unknown
import keeps the whole document on syntax-only checks. No workspace files or
broker statutes are fetched, and checked documents never execute or retain
temporary cases between edits.

Some imported compilation errors appear at document start with an explicit
bundled-import message. The server uses a precise source range only when it can
independently establish that the error belongs to the open document. This avoids
highlighting a canon file's line as though it came from your source.

## Docket view: `trial watch`

`trial watch` is a live dashboard of cases, their current attention, and
consumer lag. Use `--once` for one snapshot or `--interval` to set the refresh
period.
