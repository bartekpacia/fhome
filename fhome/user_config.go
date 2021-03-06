package fhome

import "fmt"

type UserConfig struct {
	Cells  []Cell  `json:"cells"`
	Panels []Panel `json:"panels"`
	Server Server  `json:"server"`
}

func (f *UserConfig) GetCellsByPanelID(id string) []Cell {
	cells := make([]Cell, 0)
	for _, cell := range f.Cells {
		for _, pos := range cell.PositionInPanel {
			if pos.PanelID == id {
				cells = append(cells, cell)
			}
		}
	}

	return cells
}

type Server struct {
	ProjectVersion string `json:"projectVersion"`
}

type Cell struct {
	ObjectID        int               `json:"objectId"`
	Icon            string            `json:"icon"`
	Name            string            `json:"name"`
	PositionInPanel []PositionInPanel `json:"positionInPanel"`
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

type Panel struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	X                    int         `json:"x"`
	Y                    int         `json:"y"`
	Icon                 interface{} `json:"icon"`
	ColumnCountPortrait  int         `json:"columnCountPortrait"`
	ColumnCountLandscape int         `json:"columnCountLandscape"`
}
