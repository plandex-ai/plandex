diff --git a/tests/go/shared/pre_build.go b/tests/go/valid/post_build.go
index c90eeb7..90405e7 100644
--- a/tests/go/shared/pre_build.go
+++ b/tests/go/valid/post_build.go
@@ -7,11 +7,38 @@ import (
 	"plandex/auth"
 	"plandex/lib"
 	"plandex/term"
+	"strconv"
+	"strings"
 
 	"github.com/plandex/plandex/shared"
 	"github.com/spf13/cobra"
 )
 
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
+}
+
 var contextRmCmd = &cobra.Command{
 	Use:     "rm",
 	Aliases: []string{"remove", "unload"},
