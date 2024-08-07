pdx-1: package database
pdx-2: 
pdx-3: import "database/sql"
pdx-4: 
pdx-5: var db *sql.DB
pdx-6: 
pdx-7: func init() {
pdx-8:   // Initialize database connection
pdx-9: }
pdx-10: 
pdx-11: func GetConnection() *sql.DB {
pdx-12:   return db
pdx-13: }
