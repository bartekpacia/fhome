package main

import (
	"fmt"
	"os"

	"github.com/bartekpacia/fhome/api"
	"golang.org/x/exp/slog"
)

var (
	jsonHandler = slog.NewJSONHandler(os.Stdout)
	logger      = slog.New(jsonHandler)
)

// printCellData prints the values of its arguments into a JSON object.
func printCellData(cellValue *api.CellValue, cfg *api.Config) error {
	cell, err := cfg.GetCellByID(cellValue.IntID())
	if err != nil {
		return fmt.Errorf("failed to get cell with ID %d: %v", cellValue.IntID(), err)
	}

	logger.Info("cell event received",
		slog.Int("id", cell.ID),
		slog.String("name", cell.Name),
		slog.String("desc", cell.Desc),
		slog.String("display_type", string(cellValue.DisplayType)),
		slog.String("value", cellValue.Value),
		slog.String("value_str", cellValue.ValueStr),
	)
	return nil
}
