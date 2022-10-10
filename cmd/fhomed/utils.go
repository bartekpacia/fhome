package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/bartekpacia/fhome/api"
)

func dumpConfig(cfg *api.Config) error {
	file, err := os.Create("config.json")
	if err != nil {
		return fmt.Errorf("create config.json: %v", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %v", err)
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("write config: %v", err)
	}

	return nil
}

func richPrint(cellValue *api.CellValue, cfg *api.Config) error {
	cell, err := cfg.GetCellByID(cellValue.IntID())
	if err != nil {
		return fmt.Errorf("failed to get cell with ID %d: %v", cellValue.IntID(), err)
	}

	log.Printf(",%d, %s, %s, %s, %s, %s\n", cell.ID, cell.Name, cell.Desc, cellValue.DisplayType, cellValue.Value, cellValue.ValueStr)
	return nil
}
