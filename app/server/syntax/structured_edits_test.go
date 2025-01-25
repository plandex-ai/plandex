package syntax

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructuredReplacements(t *testing.T) {
	tests := []struct {
		only        bool
		name        string
		original    string
		proposed    string
		want        string
		ext         string
		isInsert    bool
		laxNewlines bool
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

			want: `
    func processUser(id int) error {
        validate(id)
        startTx()
        updateUser(id)
        commit()
        log.Info("processing user")
        return nil
    }`,
			ext: "go",
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
			ext: "go",
		},
		{
			name:        "multiple refs in class/nested structures",
			laxNewlines: true,
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
			ext: "go",
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
			want: `
    func processUser(id int) error {
        validate(id)
        validateAgain(id)
        updateUser(id)
        commit()
        return nil
    }`,
			ext: "go",
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
			want: `
    func processUser(id int) error {
        validate(id)
        validateAgain(id)
        updateUser(id)
        commit()
        return nil
    }`,
			ext: "go",
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
			ext: "json",
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
			ext: "js",
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
			ext: "ts",
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
			ext: "js",
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
			ext: "js",
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
			ext: "json",
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
			ext: "json",
		},

		{
			name:        "scala complex structures",
			laxNewlines: true,
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

			ext: "scala",
		},

		{
			name:        "top-level ambiguous",
			laxNewlines: true,
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
			ext: "js",
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
			ext: "js",
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
			ext: "go",
		},
		{
			name: "insert between non-adjacent anchors",
			original: `func main() {
  fmt.Println("start")
  doSomething()
  fmt.Println("middle")
  someOtherThing()
  fmt.Println("end")
}`,
			proposed: `func main() {
  fmt.Println("start")
  doSomething()
  log.Info("new log")
  fmt.Println("end")
}`,
			want: `func main() {
  fmt.Println("start")
  doSomething()
  log.Info("new log")
  fmt.Println("middle")
  someOtherThing()
  fmt.Println("end")
}`,
			isInsert: true,
		},
		{
			name: "insert with reference and non-adjacent anchors",
			original: `func processRequest(req *Request) error {
  validateRequest(req)
  startTransaction()

  err := updateData(req)
  if err != nil {
      return err
  }

  notifyUpdate()
  commitTransaction()
  return nil
}`,
			proposed: `func processRequest(req *Request) error {
  // ... existing code ...
  startTransaction()

  log.Info("processing request", req.ID)

  commitTransaction()
  return nil
}`,
			want: `func processRequest(req *Request) error {
  validateRequest(req)
  startTransaction()

  log.Info("processing request", req.ID)

  err := updateData(req)
  if err != nil {
      return err
  }

  notifyUpdate()
  commitTransaction()
  return nil
}`,
			isInsert: true,
		},
	}

	onlyTests := map[int]bool{}

	for i, tt := range tests {
		if tt.only {
			onlyTests[i] = true
		}
	}

	for i, tt := range tests {
		if len(onlyTests) > 0 {
			if _, ok := onlyTests[i]; !ok {
				continue
			}
		}

		t.Run(tt.name, func(t *testing.T) {
			// got, err := ApplyChanges(
			// 	context.Background(),
			// 	tt.language,
			// 	tt.parser,
			// 	tt.original,
			// 	tt.proposed,
			// 	tt.references,
			// 	tt.removals,
			// 	anchorLines,
			// )

			// originalLines := strings.Split(tt.original, "\n")
			// proposedLines := strings.Split(tt.proposed, "\n")

			desc := ""

			if tt.isInsert {
				desc = "Type: add"
			}

			parser, lang, _, _ := GetParserForExt("." + tt.ext)

			res := ApplyChanges(
				tt.original,
				tt.proposed,
				desc,
				false,
				parser,
				lang,
				context.Background(),
			)

			fmt.Println()
			fmt.Println("NAME:", tt.name)
			fmt.Println(res.NewFile)
			fmt.Println()

			// assert.Empty(t, res.NeedsVerifyReasons)

			if tt.laxNewlines {
				assert.Equal(t, strings.ReplaceAll(tt.want, "\n", ""), strings.ReplaceAll(res.NewFile, "\n", ""))
			} else {
				assert.Equal(t, tt.want, res.NewFile)
			}
		})
	}
}
