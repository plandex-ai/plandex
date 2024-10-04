pdx-1: package server
pdx-2: import (
pdx-3: 	"log"
pdx-4: 	"net/http"
pdx-5: )
pdx-6: 
pdx-7: func init() { log.Println("Server starting...") }
pdx-8: 
pdx-9: func handler(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Hello, world!")) }
pdx-10: 