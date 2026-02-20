package api

import (
	"fmt"
	"strings"
)

type UserConfig struct {
	Cells  []UserCell  `json:"cells"`
	Panels []UserPanel `json:"panels"`
	Server struct {
		ProjectVersion string `json:"projectVersion"`
	} `json:"server"`
}

func (f *UserConfig) GetCellsByPanelID(id string) []UserCell {
	cells := make([]UserCell, 0)
	for _, cell := range f.Cells {
		for _, pos := range cell.PositionInPanel {
			if pos.PanelID == id {
				cells = append(cells, cell)
			}
		}
	}

	return cells
}

type UserCell struct {
	ObjectID        int               `json:"objectId"`
	Icon            string            `json:"icon"`
	Name            string            `json:"name"`
	PositionInPanel []PositionInPanel `json:"positionInPanel"`
}

func (cell *UserCell) IconName() string {
	iconName := cell.Icon
	iconName = strings.TrimPrefix(iconName, "icon_cell_")
	iconName = strings.TrimSuffix(iconName, "_white")
	return iconName
}

type PositionInPanel struct {
	Orientation string `json:"orientation"`
	PanelID     string `json:"panelId"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

func (p PositionInPanel) String() string {
	return fmt.Sprintf("X: %d, Y: %d", p.X, p.Y)
}

type UserPanel struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	X                    int    `json:"x"`
	Y                    int    `json:"y"`
	Icon                 any    `json:"icon"`
	ColumnCountPortrait  int    `json:"columnCountPortrait"`
	ColumnCountLandscape int    `json:"columnCountLandscape"`
}
