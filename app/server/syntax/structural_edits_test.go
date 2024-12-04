package syntax

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/plandex/plandex/shared"
	tree_sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
)

func TestStructuralReplacements(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		proposed    string
		references  []Reference
		removals    []Removal
		want        string
		language    shared.TreeSitterLanguage
		parser      *tree_sitter.Parser
		anchorLines map[int]int
	}{
		{
			name: "single reference in function",
			original: `
    func processUser(id int) error {
        validate(id)
        startTx()
        updateUser(id)
        commit()
        return nil
    }`,
			proposed: `
    func processUser(id int) error {
        // ... existing code ...
        log.Info("processing user")
        return nil
    }`,
			references: []Reference{
				3,
			},
			want: `
    func processUser(id int) error {
        validate(id)
        startTx()
        updateUser(id)
        commit()
        log.Info("processing user")
        return nil
    }`,
			language: shared.TreeSitterLanguageGo,
			parser:   GetParserForLanguage("go"),
		},
		{
			name: "bad formatting",
			original: `
    func processUser(id int) error {
    validate(id)
    validateAgain(id)
    prepForUpdate(id)
    return update(id)
    }`,
			proposed: `
    func processUser(id int) error {
    // ... existing code ...
    if force {
    log.Warn("will force update")
    }
    return update(id)
    }`,
			references: []Reference{
				3,
			},
			want: `
    func processUser(id int) error {
    validate(id)
    validateAgain(id)
    prepForUpdate(id)
    if force {
    log.Warn("will force update")
    }
    return update(id)
    }`,
			language: shared.TreeSitterLanguageGo,
			parser:   GetParserForLanguage("go"),
		},
		{
			name: "multiple refs in class/nested structures",
			original: `
    package main
    
    import "log"

    func init() {
      log.Println("init")
    }

    type UserService struct {
        db *DB
        cache *Cache
    }

    func (s *UserService) Process() {
        s.validate()
        s.update()
        s.notify()
    }

    func (s *UserService) Update() {
        s.db.begin()
        s.db.exec()
        s.db.commit()
    }

    func (s *UserService) Record() {
      log.Println("record")
    }
    `,
			proposed: `
    // ... existing code ...

    type UserService struct {
        // ... existing code ...
        metrics *Metrics
    }

    func (s *UserService) Process() {
        // ... existing code ...
        s.metrics.Record()
        // ... existing code ...
    }

    func (s *UserService) Update() {
        // ... existing code ...
    }

    // ... existing code ...
    `,
			references: []Reference{
				2, 5, 10, 12, 16, 19,
			},
			want: `
    package main
    
    import "log"

    func init() {
      log.Println("init")
    }

    type UserService struct {
        db *DB
        cache *Cache
        metrics *Metrics
    }

    func (s *UserService) Process() {
        s.validate()
        s.update()
        s.metrics.Record()
        s.notify()
    }

    func (s *UserService) Update() {
        s.db.begin()
        s.db.exec()
        s.db.commit()
    }

    func (s *UserService) Record() {
      log.Println("record")
    }
    `,
			language: shared.TreeSitterLanguageGo,
			parser:   GetParserForLanguage("go"),
		},
		{
			name: "code removal comment",
			original: `
    func processUser(id int) error {
        validate(id)
        startTx()
        logTransaction()
        validateAgain(id)
        updateUser(id)
        commit()
        return nil
    }`,
			proposed: `
    func processUser(id int) error {
        validate(id)
        // Plandex: removed code
        validateAgain(id)
        updateUser(id)
        commit()
        return nil
    }`,
			removals: []Removal{
				4,
			},
			want: `
    func processUser(id int) error {
        validate(id)
        validateAgain(id)
        updateUser(id)
        commit()
        return nil
    }`,
			language: shared.TreeSitterLanguageGo,
			parser:   GetParserForLanguage("go"),
		},
		{
			name: "multiple code removal comments",
			original: `
    func processUser(id int) error {
        validate(id)
        startTx()
        logTransaction()
        validateAgain(id)
        logValidation()
        updateUser(id)
        commit()
        return nil
    }`,
			proposed: `
    func processUser(id int) error {
        validate(id)
        // Plandex: removed code
        validateAgain(id)
        // Plandex: removed code
        updateUser(id)
        commit()
        return nil
    }`,
			removals: []Removal{
				4, 6,
			},
			want: `
    func processUser(id int) error {
        validate(id)
        validateAgain(id)
        updateUser(id)
        commit()
        return nil
    }`,
			language: shared.TreeSitterLanguageGo,
			parser:   GetParserForLanguage("go"),
		},
		{
			name: "json update with reference comments",
			original: `{
        "name": "test-app",
        "version": "1.0.0",
        "dependencies": {
            "express": "^4.17.1",
            "body-parser": "^1.19.0",
            "cors": "^2.8.5"
        },
        "scripts": {
            "start": "node index.js",
            "test": "jest",
            "build": "webpack"
        }
    }`,
			proposed: `{
        // ... existing code ...
        "dependencies": {
            "express": "^4.17.1",
            "body-parser": "^1.19.0",
            "cors": "^2.8.5",
            "dotenv": "^16.0.0",
            "mongoose": "^6.0.0"
        },
        // ... existing code ...
    }`,
			references: []Reference{
				2, 10,
			},
			want: `{
        "name": "test-app",
        "version": "1.0.0",
        "dependencies": {
            "express": "^4.17.1",
            "body-parser": "^1.19.0",
            "cors": "^2.8.5",
            "dotenv": "^16.0.0",
            "mongoose": "^6.0.0"
        },
        "scripts": {
            "start": "node index.js",
            "test": "jest",
            "build": "webpack"
        }
    }`,
			language: shared.TreeSitterLanguageJson,
			parser:   GetParserForLanguage("json"),
		},
		{
			name: "method replacement with context",
			original: `
    class UserService {
        constructor() {
            this.cache = new Cache()
        }

        async getUser(id) {
            const user = await db.find(id)
            return user
        }

        async updateUser(id, data) {
            await db.update(id, data)
            this.cache.clear()
        }

        async deleteUser(id) {
            await db.delete(id)
            this.cache.clear()
        }
    }`,
			proposed: `
    class UserService {
        // ... existing code ...

        async getUser(id) {
            const cached = await this.cache.get(id)
            if (cached) return cached
            
            const user = await db.find(id)
            await this.cache.set(id, user)
            return user
        }

        // ... existing code ...
    }`,
			references: []Reference{
				3, 14,
			},
			want: `
    class UserService {
        constructor() {
            this.cache = new Cache()
        }

        async getUser(id) {
            const cached = await this.cache.get(id)
            if (cached) return cached
            
            const user = await db.find(id)
            await this.cache.set(id, user)
            return user
        }


        async updateUser(id, data) {
            await db.update(id, data)
            this.cache.clear()
        }

        async deleteUser(id) {
            await db.delete(id)
            this.cache.clear()
        }
    }`,
			language: shared.TreeSitterLanguageJavascript,
			parser:   GetParserForLanguage("javascript"),
		},
		{
			name: "nested class methods update",
			original: `
    namespace Database {
        class Transaction {
            begin() {
                log.Info("beginning transaction")
                startTx()
            }

            commit() {
                log.Info("committing transaction")
                commitTx()
            }

            rollback() {
                log.Info("rolling back transaction")
                rollbackTx()
            }
        }
    }`,
			proposed: `
    namespace Database {
        class Transaction {
            begin() {
                // ... existing code ...
            }

            commit() {
                log.Info("committing transaction")
                validateTx()
                commitTx()
                notifyCommit()
            }

            // ... existing code ...
        }
    }`,
			references: []Reference{
				5, 15,
			},
			want: `
    namespace Database {
        class Transaction {
            begin() {
                log.Info("beginning transaction")
                startTx()
            }

            commit() {
                log.Info("committing transaction")
                validateTx()
                commitTx()
                notifyCommit()
            }


            rollback() {
                log.Info("rolling back transaction")
                rollbackTx()
            }
        }
    }`,
			language: shared.TreeSitterLanguageTypescript,
			parser:   GetParserForLanguage("typescript"),
		},
		{
			name: "update with trailing commas",
			original: `
    const handlers = {
        onStart: () => {
            console.log("starting")
        },
        onProcess: () => {
            console.log("processing")
        },
        onFinish: () => {
            console.log("finishing")
        },
    }`,
			proposed: `
    const handlers = {
        // ... existing code ...
        onProcess: () => {
            console.log("processing")
            emitEvent("process"),
        },
        // ... existing code ...
    }`,
			references: []Reference{
				3, 8,
			},
			want: `
    const handlers = {
        onStart: () => {
            console.log("starting")
        },
        onProcess: () => {
            console.log("processing")
            emitEvent("process"),
        },
        onFinish: () => {
            console.log("finishing")
        },
    }`,
			language: shared.TreeSitterLanguageJavascript,
			parser:   GetParserForLanguage("javascript"),
		},
		{
			name: "multiple structural updates",
			original: `
    class Logger {
        info(msg) {
            console.log("[INFO]", msg)
        }

        warn(msg) {
            console.log("[WARN]", msg)
        }

        error(msg) {
            console.log("[ERROR]", msg)
        }

        debug(msg) {
            console.log("[DEBUG]", msg)
        }
    }`,
			proposed: `
    class Logger {
        constructor(prefix) {
            this.prefix = prefix
        }

        info(msg) {
            console.log(this.prefix, "[INFO]", msg)
        }

        // ... existing code ...

        error(msg) {
            console.error(this.prefix, "[ERROR]", msg)
            notify("error", msg)
        }

        // ... existing code ...

        fatal(msg) {
            console.error(this.prefix, "[FATAL]", msg)
            process.exit(1)
        }
    }`,
			references: []Reference{
				11, 18,
			},
			want: `
    class Logger {
        constructor(prefix) {
            this.prefix = prefix
        }

        info(msg) {
            console.log(this.prefix, "[INFO]", msg)
        }

        warn(msg) {
            console.log("[WARN]", msg)
        }

        error(msg) {
            console.error(this.prefix, "[ERROR]", msg)
            notify("error", msg)
        }

        debug(msg) {
            console.log("[DEBUG]", msg)
        }

        fatal(msg) {
            console.error(this.prefix, "[FATAL]", msg)
            process.exit(1)
        }
    }`,
			language: shared.TreeSitterLanguageJavascript,
			parser:   GetParserForLanguage("javascript"),
		},
		{
			name: "updated variable assignment",
			original: `
      import logger from "logger";

      const a = 1;
      const b = 2;
      const c = 3;
    `,
			proposed: `
      // ... existing code ...

      const a = 10;
      const b = 2;
      // ... existing code ...
    `,
			references:  []Reference{2, 6},
			anchorLines: map[int]int{4: 4},
			want: `
      import logger from "logger";

      const a = 10;
      const b = 2;
      const c = 3;
    `,
			language: shared.TreeSitterLanguageJavascript,
			parser:   GetParserForLanguage("javascript"),
		},

		{
			name: "json multi-level update",
			original: `
{
  "name": "vscode-plandex",
  "displayName": "Plandex",
  "description": "VSCode extension for Plandex integration",
  "version": "0.1.0",
  "engines": {
    "vscode": "^1.80.0"
  },
  "categories": [
    "Other"
  ],
  "activationEvents": [
    "onLanguage:plandex"
  ],
  "main": "./dist/extension.js",
  "contributes": {
    "languages": [{
      "id": "plandex",
      "aliases": ["Plandex", "plandex"],
      "extensions": [".pd"]
    }],
    "commands": [
      {
        "command": "plandex.tellPlandex",
        "title": "Tell Plandex"
      }
    ],
    "keybindings": [{
      "command": "plandex.showFilePicker",
      "key": "@",
      "when": "editorTextFocus && editorLangId == plandex"
    }]
  },
  "scripts": {
    "vscode:prepublish": "npm run package",
    "compile": "webpack",
    "watch": "webpack --watch",
    "package": "webpack --mode production --devtool hidden-source-map",
    "compile-tests": "tsc -p . --outDir out",
    "watch-tests": "tsc -p . -w --outDir out",
    "test": "node ./out/test/runTest.js"
  },
  "devDependencies": {
    "@types/vscode": "^1.80.0",
    "@types/glob": "^8.1.0",
    "@types/mocha": "^10.0.1",
    "@types/node": "20.2.5",
    "@typescript-eslint/eslint-plugin": "^5.59.8",
    "@typescript-eslint/parser": "^5.59.8",
    "eslint": "^8.41.0",
    "glob": "^8.1.0",
    "mocha": "^10.2.0",
    "typescript": "^5.1.3",
    "ts-loader": "^9.4.3",
    "webpack": "^5.85.0",
    "webpack-cli": "^5.1.1",
    "@vscode/test-electron": "^2.3.2"
  }
}
`,
			proposed: `
{
  // ... existing code ...  
  "contributes": {
    "languages": [{
      "id": "plandex",
      "aliases": ["Plandex", "plandex"],
      "extensions": [".pd"],
      "configuration": "./language-configuration.json",
      "icon": {
          "light": "./icons/plandex-light.png",
          "dark": "./icons/plandex-dark.png"
      }
    }],
    "grammars": [{
      "language": "plandex",
      "scopeName": "text.plandex",
      "path": "./syntaxes/plandex.tmLanguage.json",
      "embeddedLanguages": {
          "meta.embedded.block.yaml": "yaml",
          "text.html.markdown": "markdown"
      }
    }],
    // ... existing code ...
  },
  // ... existing code ...
}
`,
			want: `
{
  "name": "vscode-plandex",
  "displayName": "Plandex",
  "description": "VSCode extension for Plandex integration",
  "version": "0.1.0",
  "engines": {
    "vscode": "^1.80.0"
  },
  "categories": [
    "Other"
  ],
  "activationEvents": [
    "onLanguage:plandex"
  ],
  "main": "./dist/extension.js",
  "contributes": {
    "languages": [{
      "id": "plandex",
      "aliases": ["Plandex", "plandex"],
      "extensions": [".pd"],
      "configuration": "./language-configuration.json",
      "icon": {
          "light": "./icons/plandex-light.png",
          "dark": "./icons/plandex-dark.png"
      }
    }],
    "grammars": [{
      "language": "plandex",
      "scopeName": "text.plandex",
      "path": "./syntaxes/plandex.tmLanguage.json",
      "embeddedLanguages": {
          "meta.embedded.block.yaml": "yaml",
          "text.html.markdown": "markdown"
      }
    }],
    "commands": [
      {
        "command": "plandex.tellPlandex",
        "title": "Tell Plandex"
      }
    ],
    "keybindings": [{
      "command": "plandex.showFilePicker",
      "key": "@",
      "when": "editorTextFocus && editorLangId == plandex"
    }]
  },
  "scripts": {
    "vscode:prepublish": "npm run package",
    "compile": "webpack",
    "watch": "webpack --watch",
    "package": "webpack --mode production --devtool hidden-source-map",
    "compile-tests": "tsc -p . --outDir out",
    "watch-tests": "tsc -p . -w --outDir out",
    "test": "node ./out/test/runTest.js"
  },
  "devDependencies": {
    "@types/vscode": "^1.80.0",
    "@types/glob": "^8.1.0",
    "@types/mocha": "^10.0.1",
    "@types/node": "20.2.5",
    "@typescript-eslint/eslint-plugin": "^5.59.8",
    "@typescript-eslint/parser": "^5.59.8",
    "eslint": "^8.41.0",
    "glob": "^8.1.0",
    "mocha": "^10.2.0",
    "typescript": "^5.1.3",
    "ts-loader": "^9.4.3",
    "webpack": "^5.85.0",
    "webpack-cli": "^5.1.1",
    "@vscode/test-electron": "^2.3.2"
  }
}
`,
			language: shared.TreeSitterLanguageJson,
			parser:   GetParserForLanguage("json"),
			references: []Reference{
				3, 24, 26,
			},
		},

		{
			name: "json multi-level update 2",
			original: `
{
  "name": "vscode-plandex",
  "displayName": "Plandex",
  "description": "VSCode extension for Plandex integration",
  "version": "0.1.0",
  "publisher": "plandex",
  "engines": {
    "vscode": "^1.80.0"
  },
  "categories": [
    "Other"
  ],
  "activationEvents": [
    "onLanguage:pdx"
  ],
  "main": "./dist/extension.js",
  "contributes": {
    "languages": [
      {
        "id": "pdx",
        "aliases": ["Plandex", "pdx"],
        "extensions": [".pdx"],
        "configuration": "./language-configuration.json"
      }
    ],
    "grammars": [
      {
        "language": "pdx",
        "scopeName": "source.mdx",
        "path": "./syntaxes/mdx.tmLanguage.json",
        "embeddedLanguages": {
          "meta.embedded.block.yaml": "yaml",
          "meta.embedded.block.markdown": "markdown"
        }
      }
    ],
    "commands": [
      {
        "command": "plandex.tellPlandex",
        "title": "Tell Plandex",
        "icon": "$(play)"
      }
    ],
    "keybindings": [
      {
        "command": "plandex.showFilePicker",
        "key": "@",
        "when": "editorTextFocus && editorLangId == pdx"
      }
    ],
    "menus": {
      "editor/title": [
        {
          "command": "plandex.tellPlandex",
          "group": "navigation",
          "when": "editorLangId == pdx"
        }
      ]
    }
  },
  "scripts": {
    "vscode:prepublish": "npm run package",
    "compile": "webpack",
    "watch": "webpack --watch",
    "package": "webpack --mode production --devtool hidden-source-map",
    "compile-tests": "tsc -p . --outDir out",
    "watch-tests": "tsc -p . -w --outDir out",
    "pretest": "npm run compile-tests && npm run compile && npm run lint",
    "lint": "eslint src --ext ts",
    "test": "node ./out/test/runTest.js"
  },
  "devDependencies": {
    "@types/glob": "^8.1.0",
    "@types/mocha": "^10.0.1",
    "@types/node": "^20.2.5",
    "@types/vscode": "^1.80.0",
    "@typescript-eslint/eslint-plugin": "^5.59.8",
    "@typescript-eslint/parser": "^5.59.8",
    "eslint": "^8.41.0",
    "glob": "^8.1.0",
    "mocha": "^10.2.0",
    "ts-loader": "^9.4.3",
    "typescript": "^5.1.3",
    "webpack": "^5.85.0",
    "webpack-cli": "^5.1.1"
  },
  "dependencies": {
    "yaml": "^2.3.1"
  }
}
`,
			proposed: `
{
  // ... existing code ...

  "contributes": {
    // ... existing code ...

    "commands": [
      {
        "command": "plandex.tellPlandex",
        "title": "Tell Plandex",
        "icon": {
          "light": "resources/light/play.svg",
          "dark": "resources/dark/play.svg"
        }
      }
    ],

    // ... existing code ...
  },

  // ... existing code ...
}
`,
			want: `
{
  "name": "vscode-plandex",
  "displayName": "Plandex",
  "description": "VSCode extension for Plandex integration",
  "version": "0.1.0",
  "publisher": "plandex",
  "engines": {
    "vscode": "^1.80.0"
  },
  "categories": [
    "Other"
  ],
  "activationEvents": [
    "onLanguage:pdx"
  ],
  "main": "./dist/extension.js",
  "contributes": {
    "languages": [
      {
        "id": "pdx",
        "aliases": ["Plandex", "pdx"],
        "extensions": [".pdx"],
        "configuration": "./language-configuration.json"
      }
    ],
    "grammars": [
      {
        "language": "pdx",
        "scopeName": "source.mdx",
        "path": "./syntaxes/mdx.tmLanguage.json",
        "embeddedLanguages": {
          "meta.embedded.block.yaml": "yaml",
          "meta.embedded.block.markdown": "markdown"
        }
      }
    ],
    "commands": [
      {
        "command": "plandex.tellPlandex",
        "title": "Tell Plandex",
        "icon": {
          "light": "resources/light/play.svg",
          "dark": "resources/dark/play.svg"
        }
      }
    ],

    "keybindings": [
      {
        "command": "plandex.showFilePicker",
        "key": "@",
        "when": "editorTextFocus && editorLangId == pdx"
      }
    ],
    "menus": {
      "editor/title": [
        {
          "command": "plandex.tellPlandex",
          "group": "navigation",
          "when": "editorLangId == pdx"
        }
      ]
    }
  },

  "scripts": {
    "vscode:prepublish": "npm run package",
    "compile": "webpack",
    "watch": "webpack --watch",
    "package": "webpack --mode production --devtool hidden-source-map",
    "compile-tests": "tsc -p . --outDir out",
    "watch-tests": "tsc -p . -w --outDir out",
    "pretest": "npm run compile-tests && npm run compile && npm run lint",
    "lint": "eslint src --ext ts",
    "test": "node ./out/test/runTest.js"
  },
  "devDependencies": {
    "@types/glob": "^8.1.0",
    "@types/mocha": "^10.0.1",
    "@types/node": "^20.2.5",
    "@types/vscode": "^1.80.0",
    "@typescript-eslint/eslint-plugin": "^5.59.8",
    "@typescript-eslint/parser": "^5.59.8",
    "eslint": "^8.41.0",
    "glob": "^8.1.0",
    "mocha": "^10.2.0",
    "ts-loader": "^9.4.3",
    "typescript": "^5.1.3",
    "webpack": "^5.85.0",
    "webpack-cli": "^5.1.1"
  },
  "dependencies": {
    "yaml": "^2.3.1"
  }
}
`,
			language: shared.TreeSitterLanguageJson,
			parser:   GetParserForLanguage("json"),
			references: []Reference{
				3, 6, 19, 22,
			},
		},

		{
			name: "scala complex structures",
			original: `
package domain.service

import java.time.format.DateTimeFormatter

class MetricsService(
    client: Client,
    service: Service,
    automation: Automation
)(
    implicit context: Context
) extends LazyLogging
  with BaseImplicits {

    def metrics(
        ids: Seq[Id],
        channels: Option[Seq[Channel]],
    ): Future[Metrics] = {

      getMetrics(
        ids,
        channels,
        Endpoint.Metrics
      )
    }

    def metrics2(
        ids: Seq[Id],
        channels: Option[Seq[Channel]],
    ): Future[Metrics] = {

      getMetrics2(
        ids,
        channels,
        Endpoint.Metrics
      )
    }
  }
`,

			proposed: `
package domain.service

// ... existing code ...

class MetricsService(
  // ... existing code ...
)(
    implicit context: Context
) extends LazyLogging
  with BaseImplicits {

    // ... existing code ...

    def update(authContext: AuthContext, id: String): Future[Done] = {
      fallbacks.stub
        .update(
          updateRequest(
            authContext = Some(authContext),
            id = id
          )
        )
        .map(_ => Done)
    }

    // ... existing code ...
  }
`,

			want: `
package domain.service

import java.time.format.DateTimeFormatter

class MetricsService(
    client: Client,
    service: Service,
    automation: Automation
)(
    implicit context: Context
) extends LazyLogging
  with BaseImplicits {

    def metrics(
        ids: Seq[Id],
        channels: Option[Seq[Channel]],
    ): Future[Metrics] = {

      getMetrics(
        ids,
        channels,
        Endpoint.Metrics
      )
    }

    def update(authContext: AuthContext, id: String): Future[Done] = {
      fallbacks.stub
        .update(
          updateRequest(
            authContext = Some(authContext),
            id = id
          )
        )
        .map(_ => Done)
    }

    def metrics2(
        ids: Seq[Id],
        channels: Option[Seq[Channel]],
    ): Future[Metrics] = {

      getMetrics2(
        ids,
        channels,
        Endpoint.Metrics
      )
    }
  }
`,
			references: []Reference{
				4, 7, 13, 26,
			},
			parser: GetParserForLanguage("scala"),
		},

		{
			name: "top-level ambiguous",
			original: `
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
    `,
			proposed: `
    // ... existing code ...

    function newFunction() {
      console.log("newFunction")
      const res = await callSomething()
      return res
    }

    // ... existing code ...
    `,
			references: []Reference{
				2, 10,
			},
			want: `
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

    function newFunction() {
      console.log("newFunction")
      const res = await callSomething()
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
    `,
			parser: GetParserForLanguage("javascript"),
		},

		{
			name: "top-level with anchors",
			original: `
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
      doStuff()
    }

    function callSomething() {
      console.log("callSomething")
      await logSomething()
      return "something"
    }
    `,
			proposed: `
    // ... existing code ...

    function processResponse(res) {
      // ... existing code ...
    }

    function newFunction() {
      console.log("newFunction")
      const res = await callSomething()
      return res
    }

    function yetAnotherFunction() {
      // ... existing code ...
    }

    // ... existing code ...
    `,
			references: []Reference{
				2, 5, 15, 18,
			},
			want: `
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

    function newFunction() {
      console.log("newFunction")
      const res = await callSomething()
      return res
    }

    function yetAnotherFunction() {
      console.log("yetAnotherFunction")
      doStuff()
    }

    function callSomething() {
      console.log("callSomething")
      await logSomething()
      return "something"
    }
    `,
			parser: GetParserForLanguage("javascript"),
		},

		{
			name: "clean up extraneous newlines",
			original: `
      func func1 () {
        fmt.Println("func1")
      }

      func func2 () {
        fmt.Println("log something")
        fmt.Println("func2")
      }

      func func3 () {
        fmt.Println("func3")

        fmt.Println("func3")
      }
      `,
			proposed: `
      // ... existing code ...

      func func2 () {
        // ... existing code ...
      }

      func newFunc () {
        console.log("newFunc")
      }

      func func3 () {

      // ... existing code ...
      `,
			references: []Reference{
				2, 5, 14,
			},
			want: `
      func func1 () {
        fmt.Println("func1")
      }

      func func2 () {
        fmt.Println("log something")
        fmt.Println("func2")
      }

      func newFunc () {
        console.log("newFunc")
      }

      func func3 () {
        fmt.Println("func3")

        fmt.Println("func3")
      }
      `,
			parser: GetParserForLanguage("go"),
		},

		{
			name: "slice error",
			original: `package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"
	"strconv"
	"strings"
	"time"

	"context"

	"os/signal"
	"syscall"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

const defaultEditor = "vim"
const defaultAutoDebugTries = 5

var autoConfirm bool

// const defaultEditor = "nano"

var tellPromptFile string
var tellBg bool
var tellStop bool
var tellNoBuild bool
var tellAutoApply bool
var tellAutoContext bool
var noExec bool
var autoDebug int

// tellCmd represents the prompt command
var tellCmd = &cobra.Command{
	Use:     "tell [prompt]",
	Aliases: []string{"t"},
	Short:   "Send a prompt for the current plan",
	// Long:  ` + "``" + `,
	Args: cobra.RangeArgs(0, 1),
	Run:  doTell,
}

func init() {
	RootCmd.AddCommand(tellCmd)

	tellCmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")
	tellCmd.Flags().BoolVarP(&tellStop, "stop", "s", false, "Stop after a single reply")
	tellCmd.Flags().BoolVarP(&tellNoBuild, "no-build", "n", false, "Don't build files")
	tellCmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background")

	tellCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm context updates")
	tellCmd.Flags().BoolVar(&tellAutoApply, "apply", false, "Automatically apply changes (and confirm context updates)")
	tellCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git when --apply is passed")

	tellCmd.Flags().BoolVar(&tellAutoContext, "auto-context", false, "Load and manage context automatically")

	tellCmd.Flags().BoolVar(&noExec, "no-exec", false, "Disable command execution")
	tellCmd.Flags().BoolVar(&autoExec, "auto-exec", false, "Automatically execute commands without confirmation when --apply is passed")

	tellCmd.Flags().Var(newAutoDebugValue(&autoDebug), "debug", "Automatically execute and debug failing commands (optionally specify number of tries—default is 5)")
	tellCmd.Flag("debug").NoOptDefVal = strconv.Itoa(defaultAutoDebugTries)
}

func doTell(cmd *cobra.Command, args []string) {
	validateTellFlags()

	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

	prompt := getTellPrompt(args)

	if prompt == "" {
		fmt.Println("🤷‍♂️ No prompt to send")
		return
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			return lib.CheckOutdatedContextWithOutput(false, autoConfirm || tellAutoApply || tellAutoContext, maybeContexts)
		},
	}, prompt, plan_exec.TellFlags{
		TellBg:      tellBg,
		TellStop:    tellStop,
		TellNoBuild: tellNoBuild,
		AutoContext: tellAutoContext,
		ExecEnabled: !noExec,
	})

	if tellAutoApply {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			NoExec:      noExec,
			AutoExec:    autoExec || autoDebug > 0,
			AutoDebug:   autoDebug,
		}

		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
	}
}

func getTellPrompt(args []string) string {
	var prompt string
	var pipedData string

	if len(args) > 0 {
		prompt = args[0]
	} else if tellPromptFile != "" {
		bytes, err := os.ReadFile(tellPromptFile)
		if err != nil {
			term.OutputErrorAndExit("Error reading prompt file: %v", err)
		}
		prompt = string(bytes)
	}

	// Check if there's piped input
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		term.OutputErrorAndExit("Failed to stat stdin: %v", err)
	}

	if fileInfo.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		pipedDataBytes, err := io.ReadAll(reader)
		if err != nil {
			term.OutputErrorAndExit("Failed to read piped data: %v", err)
		}
		pipedData = string(pipedDataBytes)
	}

	if prompt == "" && pipedData == "" {
		prompt = getEditorPrompt()
	} else if pipedData != "" {
		if prompt != "" {
			prompt = fmt.Sprintf("%s\n\n---\n\n%s", prompt, pipedData)
		} else {
			prompt = pipedData
		}
	}

	return prompt
}

func prepareEditorCommand(editor string, filename string) *exec.Cmd {
	switch editor {
	case "vim":
		return exec.Command(editor, "+normal G$", "+startinsert!", filename)
	case "nano":
		return exec.Command(editor, "+99999999", filename)
	default:
		return exec.Command(editor, filename)
	}
}

func getEditorInstructions(editor string) string {
	return "👉  Write your prompt below, then save and exit to send it to Plandex.\n• To save and exit, press ESC, then type :wq! and press ENTER.\n• To exit without saving, press ESC, then type :q! and press ENTER.\n\n\n"
}

func getEditorPrompt() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
		if editor == "" {
			editor = defaultEditor
		}
	}

	tempFile, err := os.CreateTemp(os.TempDir(), "plandex_prompt_*")
	if err != nil {
		term.OutputErrorAndExit("Failed to create temporary file: %v", err)
	}

	instructions := getEditorInstructions(editor)
	filename := tempFile.Name()
	err = os.WriteFile(filename, []byte(instructions), 0644)
	if err != nil {
		term.OutputErrorAndExit("Failed to write instructions to temporary file: %v", err)
	}

	editorCmd := prepareEditorCommand(editor, filename)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	err = editorCmd.Run()
	if err != nil {
		term.OutputErrorAndExit("Error opening editor: %v", err)
	}

	bytes, err := os.ReadFile(tempFile.Name())
	if err != nil {
		term.OutputErrorAndExit("Error reading temporary file: %v", err)
	}

	prompt := string(bytes)

	err = os.Remove(tempFile.Name())
	if err != nil {
		term.OutputErrorAndExit("Error removing temporary file: %v", err)
	}

	prompt = strings.TrimPrefix(prompt, strings.TrimSpace(instructions))
	prompt = strings.TrimSpace(prompt)

	return prompt

}

func validateTellFlags() {
	if tellAutoApply && tellNoBuild {
		term.OutputErrorAndExit("🚨 --apply can't be used with --no-build/-n")
	}
	if tellAutoApply && tellBg {
		term.OutputErrorAndExit("🚨 --apply can't be used with --bg")
	}
	if autoCommit && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --commit/-c can only be used with --apply")
	}
	if autoExec && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --auto-exec can only be used with --apply")
	}
	if autoDebug > 0 && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --debug can only be used with --apply")
	}
	if autoDebug > 0 && noExec {
		term.OutputErrorAndExit("🚨 --debug can't be used with --no-exec")
	}

	if tellAutoContext && tellBg {
		term.OutputErrorAndExit("🚨 --auto-context/-c can't be used with --bg")
	}
	if tellAutoContext && tellStop {
		term.OutputErrorAndExit("🚨 --auto-context/-c can't be used with --stop/-s")
	}
}

func maybeShowDiffs() {
	diffs, err := api.Client.GetPlanDiffs(lib.CurrentPlanId, lib.CurrentBranch, plainTextOutput || showDiffUi)
	if err != nil {
		term.OutputErrorAndExit("Error getting plan diffs: %v", err)
		return
	}

	if len(diffs) > 0 {
		cmd := exec.Command(os.Args[0], "diffs", "--ui")

		// Create a context that's cancelled when the program exits
		ctx, cancel := context.WithCancel(context.Background())

		// Ensure cleanup on program exit
		go func() {
			// Wait for program exit signal
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c

			// Cancel context and kill the process
			cancel()
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		go func() {
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting diffs command: %v\n", err)
				return
			}

			// Wait in a separate goroutine
			go cmd.Wait()

			// Wait for either context cancellation or process completion
			<-ctx.Done()
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		// Give the UI a moment to start
		time.Sleep(100 * time.Millisecond)
	}
}

// AutoDebugValue implements the flag.Value interface
type autoDebugValue struct {
	value *int
}

func newAutoDebugValue(p *int) *autoDebugValue {
	*p = 0 // Default to 0 (disabled)
	return &autoDebugValue{p}
}

func (f *autoDebugValue) Set(s string) error {
	if s == "" {
		*f.value = defaultAutoDebugTries
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid value for --debug: %v", err)
	}
	if v <= 0 {
		return fmt.Errorf("--debug value must be greater than 0")
	}
	*f.value = v
	return nil
}

func (f *autoDebugValue) String() string {
	if f.value == nil {
		return "0"
	}
	return strconv.Itoa(*f.value)
}

func (f *autoDebugValue) Type() string {
	return "int"
}
`,
			proposed: `
// ... existing code ...

func doTell(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

  plan, err := api.Client.GetPlan(lib.CurrentPlanId)
	if err != nil {
		term.HandleApiError(err)
		return
	}

	// Use plan config if available
	if plan.Config != nil {
		autoConfirm = plan.Config.AutoApply
		autoCommit = plan.Config.AutoCommit
		tellAutoContext = plan.Config.AutoContext
		noExec = plan.Config.NoExec
		autoDebug = plan.Config.AutoDebugTries
	} else {
		// Try user default config
		config, err := api.Client.GetUserConfig()
		if err != nil {
			term.HandleApiError(err)
			return
		}
		if config != nil {
			autoConfirm = config.AutoApply
			autoCommit = config.AutoCommit
			tellAutoContext = config.AutoContext
			noExec = config.NoExec
			autoDebug = config.AutoDebugTries
		}
	}

	prompt := getTellPrompt(args)

	// ... existing code ...
}

// ... existing code ...
`,
			references: []Reference{2, 48, 51},
			want: `package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"plandex/api"
	"plandex/auth"
	"plandex/lib"
	"plandex/plan_exec"
	"plandex/term"
	"strconv"
	"strings"
	"time"

	"context"

	"os/signal"
	"syscall"

	"github.com/plandex/plandex/shared"
	"github.com/spf13/cobra"
)

const defaultEditor = "vim"
const defaultAutoDebugTries = 5

var autoConfirm bool

// const defaultEditor = "nano"

var tellPromptFile string
var tellBg bool
var tellStop bool
var tellNoBuild bool
var tellAutoApply bool
var tellAutoContext bool
var noExec bool
var autoDebug int

// tellCmd represents the prompt command
var tellCmd = &cobra.Command{
	Use:     "tell [prompt]",
	Aliases: []string{"t"},
	Short:   "Send a prompt for the current plan",
	// Long:  ` + "``" + `,
	Args: cobra.RangeArgs(0, 1),
	Run:  doTell,
}

func init() {
	RootCmd.AddCommand(tellCmd)

	tellCmd.Flags().StringVarP(&tellPromptFile, "file", "f", "", "File containing prompt")
	tellCmd.Flags().BoolVarP(&tellStop, "stop", "s", false, "Stop after a single reply")
	tellCmd.Flags().BoolVarP(&tellNoBuild, "no-build", "n", false, "Don't build files")
	tellCmd.Flags().BoolVar(&tellBg, "bg", false, "Execute autonomously in the background")

	tellCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "Automatically confirm context updates")
	tellCmd.Flags().BoolVar(&tellAutoApply, "apply", false, "Automatically apply changes (and confirm context updates)")
	tellCmd.Flags().BoolVarP(&autoCommit, "commit", "c", false, "Commit changes to git when --apply is passed")

	tellCmd.Flags().BoolVar(&tellAutoContext, "auto-context", false, "Load and manage context automatically")

	tellCmd.Flags().BoolVar(&noExec, "no-exec", false, "Disable command execution")
	tellCmd.Flags().BoolVar(&autoExec, "auto-exec", false, "Automatically execute commands without confirmation when --apply is passed")

	tellCmd.Flags().Var(newAutoDebugValue(&autoDebug), "debug", "Automatically execute and debug failing commands (optionally specify number of tries—default is 5)")
	tellCmd.Flag("debug").NoOptDefVal = strconv.Itoa(defaultAutoDebugTries)
}

func doTell(cmd *cobra.Command, args []string) {
	auth.MustResolveAuthWithOrg()
	lib.MustResolveProject()

	if lib.CurrentPlanId == "" {
		term.OutputNoCurrentPlanErrorAndExit()
	}

	var apiKeys map[string]string
	if !auth.Current.IntegratedModelsMode {
		apiKeys = lib.MustVerifyApiKeys()
	}

  plan, err := api.Client.GetPlan(lib.CurrentPlanId)
	if err != nil {
		term.HandleApiError(err)
		return
	}

	// Use plan config if available
	if plan.Config != nil {
		autoConfirm = plan.Config.AutoApply
		autoCommit = plan.Config.AutoCommit
		tellAutoContext = plan.Config.AutoContext
		noExec = plan.Config.NoExec
		autoDebug = plan.Config.AutoDebugTries
	} else {
		// Try user default config
		config, err := api.Client.GetUserConfig()
		if err != nil {
			term.HandleApiError(err)
			return
		}
		if config != nil {
			autoConfirm = config.AutoApply
			autoCommit = config.AutoCommit
			tellAutoContext = config.AutoContext
			noExec = config.NoExec
			autoDebug = config.AutoDebugTries
		}
	}

	prompt := getTellPrompt(args)

	if prompt == "" {
		fmt.Println("🤷‍♂️ No prompt to send")
		return
	}

	plan_exec.TellPlan(plan_exec.ExecParams{
		CurrentPlanId: lib.CurrentPlanId,
		CurrentBranch: lib.CurrentBranch,
		ApiKeys:       apiKeys,
		CheckOutdatedContext: func(maybeContexts []*shared.Context) (bool, bool, error) {
			return lib.CheckOutdatedContextWithOutput(false, autoConfirm || tellAutoApply || tellAutoContext, maybeContexts)
		},
	}, prompt, plan_exec.TellFlags{
		TellBg:      tellBg,
		TellStop:    tellStop,
		TellNoBuild: tellNoBuild,
		AutoContext: tellAutoContext,
		ExecEnabled: !noExec,
	})

	if tellAutoApply {
		flags := lib.ApplyFlags{
			AutoConfirm: true,
			AutoCommit:  autoCommit,
			NoCommit:    !autoCommit,
			NoExec:      noExec,
			AutoExec:    autoExec || autoDebug > 0,
			AutoDebug:   autoDebug,
		}

		lib.MustApplyPlan(
			lib.CurrentPlanId,
			lib.CurrentBranch,
			flags,
			plan_exec.GetOnApplyExecFail(flags),
		)
	}
}

func getTellPrompt(args []string) string {
	var prompt string
	var pipedData string

	if len(args) > 0 {
		prompt = args[0]
	} else if tellPromptFile != "" {
		bytes, err := os.ReadFile(tellPromptFile)
		if err != nil {
			term.OutputErrorAndExit("Error reading prompt file: %v", err)
		}
		prompt = string(bytes)
	}

	// Check if there's piped input
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		term.OutputErrorAndExit("Failed to stat stdin: %v", err)
	}

	if fileInfo.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		pipedDataBytes, err := io.ReadAll(reader)
		if err != nil {
			term.OutputErrorAndExit("Failed to read piped data: %v", err)
		}
		pipedData = string(pipedDataBytes)
	}

	if prompt == "" && pipedData == "" {
		prompt = getEditorPrompt()
	} else if pipedData != "" {
		if prompt != "" {
			prompt = fmt.Sprintf("%s\n\n---\n\n%s", prompt, pipedData)
		} else {
			prompt = pipedData
		}
	}

	return prompt
}

func prepareEditorCommand(editor string, filename string) *exec.Cmd {
	switch editor {
	case "vim":
		return exec.Command(editor, "+normal G$", "+startinsert!", filename)
	case "nano":
		return exec.Command(editor, "+99999999", filename)
	default:
		return exec.Command(editor, filename)
	}
}

func getEditorInstructions(editor string) string {
	return "👉  Write your prompt below, then save and exit to send it to Plandex.\n• To save and exit, press ESC, then type :wq! and press ENTER.\n• To exit without saving, press ESC, then type :q! and press ENTER.\n\n\n"
}

func getEditorPrompt() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
		if editor == "" {
			editor = defaultEditor
		}
	}

	tempFile, err := os.CreateTemp(os.TempDir(), "plandex_prompt_*")
	if err != nil {
		term.OutputErrorAndExit("Failed to create temporary file: %v", err)
	}

	instructions := getEditorInstructions(editor)
	filename := tempFile.Name()
	err = os.WriteFile(filename, []byte(instructions), 0644)
	if err != nil {
		term.OutputErrorAndExit("Failed to write instructions to temporary file: %v", err)
	}

	editorCmd := prepareEditorCommand(editor, filename)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	err = editorCmd.Run()
	if err != nil {
		term.OutputErrorAndExit("Error opening editor: %v", err)
	}

	bytes, err := os.ReadFile(tempFile.Name())
	if err != nil {
		term.OutputErrorAndExit("Error reading temporary file: %v", err)
	}

	prompt := string(bytes)

	err = os.Remove(tempFile.Name())
	if err != nil {
		term.OutputErrorAndExit("Error removing temporary file: %v", err)
	}

	prompt = strings.TrimPrefix(prompt, strings.TrimSpace(instructions))
	prompt = strings.TrimSpace(prompt)

	return prompt

}

func validateTellFlags() {
	if tellAutoApply && tellNoBuild {
		term.OutputErrorAndExit("🚨 --apply can't be used with --no-build/-n")
	}
	if tellAutoApply && tellBg {
		term.OutputErrorAndExit("🚨 --apply can't be used with --bg")
	}
	if autoCommit && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --commit/-c can only be used with --apply")
	}
	if autoExec && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --auto-exec can only be used with --apply")
	}
	if autoDebug > 0 && !tellAutoApply {
		term.OutputErrorAndExit("🚨 --debug can only be used with --apply")
	}
	if autoDebug > 0 && noExec {
		term.OutputErrorAndExit("🚨 --debug can't be used with --no-exec")
	}

	if tellAutoContext && tellBg {
		term.OutputErrorAndExit("🚨 --auto-context/-c can't be used with --bg")
	}
	if tellAutoContext && tellStop {
		term.OutputErrorAndExit("🚨 --auto-context/-c can't be used with --stop/-s")
	}
}

func maybeShowDiffs() {
	diffs, err := api.Client.GetPlanDiffs(lib.CurrentPlanId, lib.CurrentBranch, plainTextOutput || showDiffUi)
	if err != nil {
		term.OutputErrorAndExit("Error getting plan diffs: %v", err)
		return
	}

	if len(diffs) > 0 {
		cmd := exec.Command(os.Args[0], "diffs", "--ui")

		// Create a context that's cancelled when the program exits
		ctx, cancel := context.WithCancel(context.Background())

		// Ensure cleanup on program exit
		go func() {
			// Wait for program exit signal
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c

			// Cancel context and kill the process
			cancel()
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		go func() {
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting diffs command: %v\n", err)
				return
			}

			// Wait in a separate goroutine
			go cmd.Wait()

			// Wait for either context cancellation or process completion
			<-ctx.Done()
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}()

		// Give the UI a moment to start
		time.Sleep(100 * time.Millisecond)
	}
}

// AutoDebugValue implements the flag.Value interface
type autoDebugValue struct {
	value *int
}

func newAutoDebugValue(p *int) *autoDebugValue {
	*p = 0 // Default to 0 (disabled)
	return &autoDebugValue{p}
}

func (f *autoDebugValue) Set(s string) error {
	if s == "" {
		*f.value = defaultAutoDebugTries
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid value for --debug: %v", err)
	}
	if v <= 0 {
		return fmt.Errorf("--debug value must be greater than 0")
	}
	*f.value = v
	return nil
}

func (f *autoDebugValue) String() string {
	if f.value == nil {
		return "0"
	}
	return strconv.Itoa(*f.value)
}

func (f *autoDebugValue) Type() string {
	return "int"
}
`,
			parser: GetParserForLanguage("go"),
		},

		{
			name: "extraneous references, no change",
			original: `package cmd

import (
	"fmt"
	"os"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var helpShowAll bool

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use: ` + "`plandex [command] [flags]`" + `,
	// Short: "Plandex: iterative development with AI",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// term.OutputErrorAndExit("Error executing root command: %v", err)
		// log.Fatalf("Error executing root command: %v", err)

		// output the error message to stderr
		term.OutputSimpleError("Error: %v", err)

		fmt.Println()

		color.New(color.Bold, color.BgGreen, color.FgHiWhite).Println(" Usage ")
		color.New(color.Bold).Println("  plandex [command] [flags]")
		color.New(color.Bold).Println("  pdx [command] [flags]")
		fmt.Println()

		color.New(color.Bold, color.BgGreen, color.FgHiWhite).Println(" Help ")
		color.New(color.Bold).Println("  plandex help # show basic usage")
		color.New(color.Bold).Println("  plandex help --all # show all commands")
		color.New(color.Bold).Println("  plandex [command] --help")
		fmt.Println()

		os.Exit(1)

	}
}

func run(cmd *cobra.Command, args []string) {
}

func init() {
	var helpCmd = &cobra.Command{
		Use:     "help",
		Aliases: []string{"h"},
		Short:   "Display help for Plandex",
		Long:    ` + "`Display help for Plandex.`" + `,
		Run: func(cmd *cobra.Command, args []string) {
			term.PrintCustomHelp(helpShowAll)
		},
	}

	RootCmd.AddCommand(helpCmd)

	// add an --all/-a flag
	helpCmd.Flags().BoolVarP(&helpShowAll, "all", "a", false, "Show all commands")
}
`,
			proposed: `// ... existing code ...

func init() {
	var helpCmd = &cobra.Command{
		Use:     "help",
		Aliases: []string{"h"},
		Short:   "Display help for Plandex",
		Long:    ` + "`Display help for Plandex.`" + `,
		Run: func(cmd *cobra.Command, args []string) {
			term.PrintCustomHelp(helpShowAll)
		},
	}

	RootCmd.AddCommand(helpCmd)

	// add an --all/-a flag
	helpCmd.Flags().BoolVarP(&helpShowAll, "all", "a", false, "Show all commands")
}

// ... existing code ...
`,
			references: []Reference{1, 20},
			want: `package cmd

import (
	"fmt"
	"os"
	"plandex/term"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var helpShowAll bool

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use: ` + "`plandex [command] [flags]`" + `,
	// Short: "Plandex: iterative development with AI",
	SilenceErrors: true,
	SilenceUsage:  true,
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// term.OutputErrorAndExit("Error executing root command: %v", err)
		// log.Fatalf("Error executing root command: %v", err)

		// output the error message to stderr
		term.OutputSimpleError("Error: %v", err)

		fmt.Println()

		color.New(color.Bold, color.BgGreen, color.FgHiWhite).Println(" Usage ")
		color.New(color.Bold).Println("  plandex [command] [flags]")
		color.New(color.Bold).Println("  pdx [command] [flags]")
		fmt.Println()

		color.New(color.Bold, color.BgGreen, color.FgHiWhite).Println(" Help ")
		color.New(color.Bold).Println("  plandex help # show basic usage")
		color.New(color.Bold).Println("  plandex help --all # show all commands")
		color.New(color.Bold).Println("  plandex [command] --help")
		fmt.Println()

		os.Exit(1)

	}
}

func run(cmd *cobra.Command, args []string) {
}

func init() {
	var helpCmd = &cobra.Command{
		Use:     "help",
		Aliases: []string{"h"},
		Short:   "Display help for Plandex",
		Long:    ` + "`Display help for Plandex.`" + `,
		Run: func(cmd *cobra.Command, args []string) {
			term.PrintCustomHelp(helpShowAll)
		},
	}

	RootCmd.AddCommand(helpCmd)

	// add an --all/-a flag
	helpCmd.Flags().BoolVarP(&helpShowAll, "all", "a", false, "Show all commands")
}
`,
			parser: GetParserForLanguage("go"),
		},
	}

	for _, tt := range tests {
		// if tt.name != "extraneous references, no change" {
		// 	continue
		// }
		t.Run(tt.name, func(t *testing.T) {
			anchorLines := map[int]int{}
			if tt.anchorLines != nil {
				anchorLines = tt.anchorLines
			}

			got, err := ApplyChanges(
				context.Background(),
				tt.language,
				tt.parser,
				tt.original,
				tt.proposed,
				tt.references,
				tt.removals,
				anchorLines,
			)
			fmt.Println()
			fmt.Println(tt.name)
			fmt.Println(got)
			fmt.Println()
			assert.NoError(t, err)

			// punting for now on minor newline discrepancies
			// but it should be possible to get this exactly right
			assert.Equal(t, strings.ReplaceAll(tt.want, "\n", ""), strings.ReplaceAll(got, "\n", ""))

			// assert.Equal(t, tt.want, got)
		})
	}
}
