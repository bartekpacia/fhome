package main

import (
	"fmt"
	"log"

	"github.com/bartekpacia/fhome/api"
)

func richPrint(cellValue *api.CellValue, cfg *api.Config) error {
	cell, err := cfg.GetCellByID(cellValue.IntID())
	if err != nil {
		return fmt.Errorf("failed to get cell with ID %d: %v", cellValue.IntID(), err)
	}

	log.Printf(",%d, %s, %s, %s, %s, %s\n", cell.ID, cell.Name, cell.Desc, cellValue.DisplayType, cellValue.Value, cellValue.ValueStr)
	return nil
}
