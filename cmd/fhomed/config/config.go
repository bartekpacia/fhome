package config

import "fmt"

// Config is a better, simpler representation of fhome.File.
type Config struct {
	Panels []Panel
}

func (p *Config) GetCellByID(cellID int) (*Cell, error) {
	for _, panel := range p.Panels {
		cell, err := panel.GetCellByID(cellID)
		if err != nil {
			// that's fine, maybe the cell is in another panel
			continue
		}

		return cell, nil
	}

	return nil, fmt.Errorf("no cell with id %d", cellID)
}

func (c *Config) GetPanelByID(id string) (*Panel, error) {
	for _, panel := range c.Panels {
		if panel.ID == id {
			return &panel, nil
		}
	}

	return nil, fmt.Errorf("no panel with id %s", id)
}

func (c *Config) GetPanelByName(name string) (*Panel, error) {
	for _, panel := range c.Panels {
		if panel.Name == name {
			return &panel, nil
		}
	}

	return nil, fmt.Errorf("no panel with name %s", name)
}

type Panel struct {
	ID    string
	Name  string
	Cells []Cell
}

func (p *Panel) GetCellByID(cellID int) (*Cell, error) {
	for _, cell := range p.Cells {
		if cell.ID == cellID {
			return &cell, nil
		}
	}

	return nil, fmt.Errorf("no cell with id %d", cellID)
}

type Cell struct {
	ID   int
	Icon Icon
	Name string
}
