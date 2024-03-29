package api

import "fmt"

// Config merges [UserConfig] and [TouchesResponse].
type Config struct {
	Panels []Panel
}

func (c *Config) GetPanelByID(id string) (*Panel, error) {
	for i := range c.Panels {
		panel := &c.Panels[i]
		if panel.ID == id {
			return panel, nil
		}
	}

	return nil, fmt.Errorf("no panel with id %s", id)
}

func (c *Config) GetPanelByName(name string) (*Panel, error) {
	for i := range c.Panels {
		panel := &c.Panels[i]
		if panel.Name == name {
			return panel, nil
		}
	}

	return nil, fmt.Errorf("no panel with name %s", name)
}

func (c *Config) Cells() []Cell {
	cells := make([]Cell, 0)
	for _, panel := range c.Panels {
		cells = append(cells, panel.Cells...)
	}
	return cells
}

func (c *Config) GetCellByID(cellID int) (*Cell, error) {
	for i := range c.Panels {
		panel := &c.Panels[i]
		cell, err := panel.GetCellByID(cellID)
		if err != nil {
			// that's fine, maybe the cell is in another panel
			continue
		}

		return cell, nil
	}

	return nil, fmt.Errorf("no cell with id %d", cellID)
}

type Panel struct {
	ID    string
	Name  string
	Cells []Cell
}

func (p *Panel) GetCellByID(cellID int) (*Cell, error) {
	for i := range p.Cells {
		cell := &p.Cells[i]
		if cell.ID == cellID {
			return cell, nil
		}
	}

	return nil, fmt.Errorf("no cell with id %d", cellID)
}

type Cell struct {
	ID   int
	Icon Icon
	Name string // Set in client apps
	Desc string // Set in the configurator app

	Value       string
	TypeNumber  string
	DisplayType string
	Preset      string
	Style       string
	MinValue    string
	MaxValue    string
}
