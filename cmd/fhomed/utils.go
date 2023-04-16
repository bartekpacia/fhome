package main

import (
	"fmt"

	"github.com/bartekpacia/fhome/api"
	"golang.org/x/exp/slog"
)

// printCellData prints the values of its arguments into a JSON object.
func printCellData(cellValue *api.CellValue, cfg *api.Config) error {
	cell, err := cfg.GetCellByID(cellValue.IntID())
	if err != nil {
		return fmt.Errorf("failed to get cell with ID %d: %v", cellValue.IntID(), err)
	}

	// Find panel ID of the cell
	var panelName string
	for _, panel := range cfg.Panels {
		for _, c := range panel.Cells {
			if c.ID == cell.ID {
				panelName = panel.Name
				break
			}
		}
	}

	slog.Info("object state changed",
		slog.Int("id", cell.ID),
		slog.String("panel", panelName),
		slog.String("name", cell.Name),
		slog.String("desc", cell.Desc),
		slog.String("display_type", string(cellValue.DisplayType)),
		slog.String("value", cellValue.Value),
		slog.String("value_str", cellValue.ValueStr),
	)
	return nil
}
