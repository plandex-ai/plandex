package types

import (
	shared "plandex-shared"
)

type ChangesWithLineNums struct {
	Comments []struct {
		Txt       string `json:"txt"`
		Reference bool   `json:"reference"`
	}
	Changes []*shared.StreamedChangeWithLineNums `json:"changes"`
}
