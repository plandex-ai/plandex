package shared

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func (m FileMapParts) CombinedMapAndSha() (string, string) {
	// Combine map parts into single body
	var combinedMap strings.Builder
	var combinedHashes bytes.Buffer
	paths := make([]string, 0, len(m))
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		part := m[path]
		fileHeading := fmt.Sprintf("### %s\n", path)
		combinedMap.WriteString(fileHeading)
		combinedMap.WriteString(part.Body)
		combinedMap.WriteString("\n")

		hash := sha256.Sum256([]byte(part.Body))
		combinedHashes.Write(hash[:])
	}

	combinedHash := sha256.Sum256(combinedHashes.Bytes())
	combinedSha := hex.EncodeToString(combinedHash[:])

	return combinedMap.String(), combinedSha
}

func (m FileMapParts) CombinedMap() string {
	var combinedMap strings.Builder
	paths := make([]string, 0, len(m))
	for path := range m {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		part := m[path]
		fileHeading := fmt.Sprintf("### %s\n", path)
		combinedMap.WriteString(fileHeading)
		combinedMap.WriteString(part.Body)
		combinedMap.WriteString("\n")
	}
	return combinedMap.String()
}
