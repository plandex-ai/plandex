 ### Subtask 1: Parse the range of indices from the command-line arguments.  
                                                               
                                                        ```                      
    package cmd                                                               
                                                                              
    import (                                                                  
    	"fmt"                                                                     
    	"path/filepath"                                                           
    	"plandex/api"                                                             
    	"plandex/auth"                                                            
    	"plandex/lib"                                                             
    	"plandex/term"                                                            
    	"strconv"                                                                 
    	"strings"                                                                 
                                                                              
    	"github.com/plandex/plandex/shared"                                       
    	"github.com/spf13/cobra"                                                  
    )                                                                         
                                                                              
    func parseRange(arg string) ([]int, error) {                              
    	var indices []int                                                         
    	parts := strings.Split(arg, "-")                                          
    	if len(parts) == 2 {                                                      
    		start, err := strconv.Atoi(parts[0])                                      
    		if err != nil {                                                           
    			return nil, err                                                           
    		}                                                                         
    		end, err := strconv.Atoi(parts[1])                                        
    		if err != nil {                                                           
    			return nil, err                                                           
    		}                                                                         
    		for i := start; i <= end; i++ {                                           
    			indices = append(indices, i)                                              
    		}                                                                         
    	} else {                                                                  
    		index, err := strconv.Atoi(arg)                                           
    		if err != nil {                                                           
    			return nil, err                                                           
    		}                                                                         
    		indices = append(indices, index)                                          
    	}                                                                         
    	return indices, nil                                                       
    }
	```