package net_usage

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var file_pattern = "/sys/class/net/%s/statistics/"

var file_map = map[string]string{
	"rx_bytes":   "Received (Bytes)",
	"tx_bytes":   "Transmitted (Bytes)",
	"tx_errors":  "Transmission Errors",
	"rx_errors":  "Reception Errors",
	"rx_packets": "Received (Packets)",
	"tx_packets": "Transmitted (Packets)",
}

var last_metrics map[string]map[string]uint64
var rwmutex sync.RWMutex

func readFile(base_path string, metric string) (uint64, error) {
	out, err := ioutil.ReadFile(filepath.Join(base_path, metric))

	if err != nil {
		return 0, err
	}

	out_i, err := strconv.ParseUint(strings.Split(string(out), "\n")[0], 10, 64)

	return out_i, err
}

func GetMetric(params interface{}, log *syslog.Writer) interface{} {

	new_metrics := false
	device := params.(string)

	if last_metrics == nil {
		rwmutex.Lock()
		last_metrics = make(map[string]map[string]uint64)
		rwmutex.Unlock()
		new_metrics = true
	}

	if last_metrics[device] == nil {
		rwmutex.Lock()
		last_metrics[device] = make(map[string]uint64)
		rwmutex.Unlock()
		new_metrics = true
	}

	if new_metrics {
		log.Debug("New instance, sending zeroes")
	}

	metrics := make(map[string]uint64)
	difference := make(map[string]uint64)

	base_path := fmt.Sprintf(file_pattern, device)

	for fn, metric := range file_map {
		log.Debug(fmt.Sprintf("Reading file: %s", fn))
		result, err := readFile(base_path, fn)
		log.Debug(fmt.Sprintf("Got result: %s", result))
		if err == nil {
			metrics[metric] = result
		} else {
			metrics[metric] = 0
		}
	}

	for metric, value := range metrics {
		if new_metrics {
			difference[metric] = 0
			rwmutex.Lock()
			last_metrics[device][metric] = value
			rwmutex.Unlock()
		} else {
			rwmutex.RLock()
			difference[metric] = value - last_metrics[device][metric]
			rwmutex.RUnlock()
			rwmutex.Lock()
			last_metrics[device][metric] = value
			rwmutex.Unlock()
		}

	}

	return difference
}
