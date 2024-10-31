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
			parser: getParserForLanguage("go"),
		},
		// no longer intending to handle modified function signatures
		// {
		// 	name: "modified function signature, ref at end of function",
		// 	original: `
		// func processUser(id int) error {
		//     validate(id)
		//     validateAgain(id)
		//     prepForUpdate(id)
		//     return update(id)
		// }`,
		// 	proposed: `
		// func processUser(id int, force bool) (error, bool) {
		//     if force {
		//         log.Warn("will force update")
		//     }
		//     // ... existing code ...
		// }`,
		// 	references: []Reference{
		// 		6,
		// 	},
		// 	want: `
		// func processUser(id int, force bool) (error, bool) {
		//     if force {
		//         log.Warn("will force update")
		//     }
		//     validate(id)
		//     validateAgain(id)
		//     prepForUpdate(id)
		//     return update(id)
		// }`,
		// 	parser: getParserForLanguage("go"),
		// },
		// {
		// 	name: "modified function signature, ref at beginning of function",
		// 	original: `
		// func processUser(id int) error {
		//     validate(id)
		//     validateAgain(id)
		//     prepForUpdate(id)
		//     return update(id)
		// }`,
		// 	proposed: `
		// func processUser(id int, force bool) (error, bool) {
		//     // ... existing code ...
		//     if force {
		//         log.Warn("will force update")
		//     }
		//     return update(id)
		// }`,
		// 	references: []Reference{
		// 		3,
		// 	},
		// 	want: `
		// func processUser(id int, force bool) (error, bool) {
		//     validate(id)
		//     validateAgain(id)
		//     prepForUpdate(id)
		//     if force {
		//         log.Warn("will force update")
		//     }
		//     return update(id)
		// }`,
		// 	parser: getParserForLanguage("go"),
		// },
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
			parser: getParserForLanguage("go"),
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
		    // ... existing fields ...
		    metrics *Metrics
		}

		func (s *UserService) Process() {
		    // ... existing validation ...
				s.metrics.Record()
		    // ... existing processing ...
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
			parser: getParserForLanguage("go"),
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
			parser: getParserForLanguage("go"),
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
			parser: getParserForLanguage("go"),
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
			parser: getParserForLanguage("json"),
		},
		// {
		// 	name: "array update with context",
		// 	original: `
		// const config = {
		//     ports: [3000, 3001, 3002, 3003],
		//     hosts: ["localhost", "127.0.0.1"],
		//     timeouts: [1000, 2000, 5000]
		// }`,
		// 	proposed: `
		// const config = {
		//     ports: [3000, 3001, 8080, 8081, 3002, 3003],
		// 		hosts: ["localhost", "127.0.0.1"],
		//     // ... existing code ...
		// }`,
		// 	references: []Reference{
		// 		4,
		// 	},
		// 	want: `
		// const config = {
		//     ports: [3000, 3001, 8080, 8081, 3002, 3003],
		//     hosts: ["localhost", "127.0.0.1"],
		//     timeouts: [1000, 2000, 5000]
		// }`,
		// 	parser: getParserForLanguage("javascript"),
		// },
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
			parser: getParserForLanguage("javascript"),
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
			parser: getParserForLanguage("typescript"),
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
			parser: getParserForLanguage("javascript"),
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
			parser: getParserForLanguage("javascript"),
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
			references: []Reference{3},
			want: `
			import logger from "logger";

			const a = 10;
			const b = 2;
			const c = 3;
		`,
			parser: getParserForLanguage("javascript"),
		},
	}

	for _, tt := range tests {
		if tt.name == "array update with context" {
			t.Run(tt.name, func(t *testing.T) {
				anchorLines := map[int]int{}
				if tt.anchorLines != nil {
					anchorLines = tt.anchorLines
				}

				got, err := ApplyReferences(
					context.Background(),
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
		}
	}
}
