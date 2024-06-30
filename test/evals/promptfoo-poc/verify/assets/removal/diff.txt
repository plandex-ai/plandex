diff --git a/pre_build.go b/tests/removal/post_build.go
index c90eeb7..e9da642 100644
--- a/pre_build.go
+++ b/tests/removal/post_build.go
@@ -7,18 +7,36 @@ import (
 	"plandex/auth"
 	"plandex/lib"
 	"plandex/term"
+	"strconv"
+	"strings"
 
 	"github.com/plandex/plandex/shared"
 	"github.com/spf13/cobra"
 )
 
-var contextRmCmd = &cobra.Command{
-	Use:     "rm",
-	Aliases: []string{"remove", "unload"},
-	Short:   "Remove context",
-	Long:    `Remove context by index, name, or glob.`,
-	Args:    cobra.MinimumNArgs(1),
-	Run:     contextRm,
+func parseRange(arg string) ([]int, error) {
+	var indices []int
+	parts := strings.Split(arg, "-")
+	if len(parts) == 2 {
+		start, err := strconv.Atoi(parts[0])
+		if err != nil {
+			return nil, err
+		}
+		end, err := strconv.Atoi(parts[1])
+		if err != nil {
+			return nil, err
+		}
+		for i := start; i <= end; i++ {
+			indices = append(indices, i)
+		}
+	} else {
+		index, err := strconv.Atoi(arg)
+		if err != nil {
+			return nil, err
+		}
+		indices = append(indices, index)
+	}
+	return indices, nil
 }
 
 func contextRm(cmd *cobra.Command, args []string) {
@@ -39,6 +57,20 @@ func contextRm(cmd *cobra.Command, args []string) {
 
 	deleteIds := map[string]bool{}
 
+	for _, arg := range args {
+		indices, err := parseRange(arg)
+		if err != nil {
+			term.OutputErrorAndExit("Error parsing range: %v", err)
+		}
+
+		for _, index := range indices {
+			if index > 0 && index <= len(contexts) {
+				context := contexts[index-1]
+				deleteIds[context.Id] = true
+			}
+		}
+	}
+
 	for i, context := range contexts {
 		for _, id := range args {
 			if fmt.Sprintf("%d", i+1) == id || context.Name == id || context.FilePath == id || context.Url == id {
@@ -64,7 +96,6 @@ func contextRm(cmd *cobra.Command, args []string) {
 					}
 					parentDir = filepath.Dir(parentDir) // Move up one directory
 				}
-
 			}
 		}
 	}
