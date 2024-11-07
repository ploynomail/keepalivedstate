package main

import (
	"github.com/ploynomail/keepalivedstate/collector"
	"github.com/ploynomail/keepalivedstate/host"
)

func GetKeepalivedState(pid string) (*collector.KeepalivedStats, error) {
	keepalivedJSON := true
	c := host.NewKeepalivedHostCollectorHost(keepalivedJSON, pid)
	coll := collector.NewKeepalivedCollector(keepalivedJSON, "", c)
	stats, err := coll.GetKeepalivedStats()
	if err != nil {
		return nil, err
	}
	return stats, nil
}
