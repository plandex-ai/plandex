package syntax

import (
	"context"
	"fmt"
	"testing"

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
		language    string
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
			language: "go",
			parser:   getParserForLanguage("go"),
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
			language: "go",
			parser:   getParserForLanguage("go"),
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
			language: "go",
			parser:   getParserForLanguage("go"),
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
			language: "go",
			parser:   getParserForLanguage("go"),
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
			language: "go",
			parser:   getParserForLanguage("go"),
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
			language: "json",
			parser:   getParserForLanguage("json"),
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
			language: "javascript",
			parser:   getParserForLanguage("javascript"),
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
			language: "typescript",
			parser:   getParserForLanguage("typescript"),
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
			language: "javascript",
			parser:   getParserForLanguage("javascript"),
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
			language: "javascript",
			parser:   getParserForLanguage("javascript"),
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
			language: "javascript",
			parser:   getParserForLanguage("javascript"),
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
			language: "json",
			parser:   getParserForLanguage("json"),
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
			language: "json",
			parser:   getParserForLanguage("json"),
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
			parser: getParserForLanguage("scala"),
		},
	}

	for _, tt := range tests {
		// if tt.name == "json multi-level update 2" {
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
			assert.Equal(t, tt.want, got)
		})
		// }
	}
}
