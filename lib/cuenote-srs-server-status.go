package mpcuenotesrsserverstatus

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

// CuenoteSrsServerStatusPlugin mackerel plugin for Cuenote SR-S server status
type CuenoteSrsServerStatusPlugin struct {
	URI      string
	Tempfile string
	Prefix   string
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p CuenoteSrsServerStatusPlugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		p.Prefix = "cuenote-srs"
	}
	return p.Prefix
}

var loadAverageReg = regexp.MustCompile(`^(LoadAverage)\t(.+)\t(.+)\t(.+)`)

var memoryReg = regexp.MustCompile(`^Memory\t(.+)\t(.+)`)
var memoryItems = map[string]string{
	"MemTotal":           "total",
	"MemUsedPercentage":  "used",
	"SwapTotal":          "swap_total",
	"SwapUsedPercentage": "swap_used",
}

var diskReg = regexp.MustCompile(`^Disk\t(.+)\t(.+)\t(.+)`)
var diskItems = map[string]string{
	"/":               "disk_root",
	"/mnt/srslogdisk": "disk_srslogdisk",
}

// FetchMetrics interface for mackerelplugin
func (p CuenoteSrsServerStatusPlugin) FetchMetrics() (map[string]float64, error) {
	resp, err := http.Get(p.URI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]float64)
	memoryInfo := make(map[string]float64)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := loadAverageReg.FindStringSubmatch(line); len(matches) == 5 {
			ret["loadavg5"], _ = strconv.ParseFloat(matches[3], 32)
		}
		if matches := memoryReg.FindStringSubmatch(line); len(matches) == 3 {
			k, ok := memoryItems[matches[1]]
			if !ok {
				continue
			}
			value, _ := strconv.ParseFloat(matches[2], 32)
			memoryInfo[k] = value
		}
		if matches := diskReg.FindStringSubmatch(line); len(matches) == 4 {
			k, ok := diskItems[matches[1]]
			if !ok {
				continue
			}
			size, _ := strconv.ParseFloat(matches[2], 32)
			used, _ := strconv.ParseFloat(matches[3], 32)
			ret[k+"_size"] = size
			ret[k+"_used"] = size * used / 100
		}
	}

	if len(memoryInfo) == 4 {
		ret["mem_total"] = memoryInfo["total"]
		ret["mem_used"] = memoryInfo["total"] * memoryInfo["used"] / 100
		ret["mem_swap_total"] = memoryInfo["swap_total"]
		ret["mem_swap_used"] = memoryInfo["swap_total"] * memoryInfo["swap_used"] / 100
	}

	return ret, nil
}

// GraphDefinition interface for mackerelplugin
func (p CuenoteSrsServerStatusPlugin) GraphDefinition() map[string]mp.Graphs {
	var graphdef = map[string]mp.Graphs{
		"cuenote-srs.loadavg": {
			Label: "Cuenote SR-S Load Average",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: "loadavg5", Label: "loadavg5", Diff: false, Stacked: false},
			},
		},
		"cuenote-srs.memory": {
			Label: "Cuenote SR-S Memory",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "mem_total", Label: "total", Diff: false, Stacked: false, Scale: 1000},
				{Name: "mem_used", Label: "used", Diff: false, Stacked: true, Scale: 1000},
				{Name: "mem_swap_total", Label: "swap total", Diff: false, Stacked: false, Scale: 1000},
				{Name: "mem_swap_used", Label: "swap used", Diff: false, Stacked: false, Scale: 1000},
			},
		},
		"cuenote-srs.disk": {
			Label: "Cuenote SR-S Disk",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "disk_root_size", Label: "/ size", Diff: false, Stacked: false, Scale: 1000},
				{Name: "disk_root_used", Label: "/ used", Diff: false, Stacked: false, Scale: 1000},
				{Name: "disk_srslogdisk_size", Label: "/mnt/srslogdisk size", Diff: false, Stacked: false, Scale: 1000},
				{Name: "disk_srslogdisk_used", Label: "/mnt/srslogdisk used", Diff: false, Stacked: false, Scale: 1000},
			},
		},
	}
	return graphdef
}

// Do the plugin
func Do() {
	optURI := flag.String("uri", "", "URI")
	optHost := flag.String("host", "", "Hostname")
	optPath := flag.String("path", "/api?cmd=get_server_status", "Path")
	optUser := flag.String("username", "", "Username")
	optPass := flag.String("password", "", "Password")
	optPrefix := flag.String("metric-key-prefix", "cuenote-srs", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	var cuenoteSrsServerStatus CuenoteSrsServerStatusPlugin
	if *optURI != "" {
		cuenoteSrsServerStatus.URI = *optURI
	} else {
		cuenoteSrsServerStatus.URI = fmt.Sprintf("https://%s:%s@%s%s", *optUser, *optPass, *optHost, *optPath)
	}

	cuenoteSrsServerStatus.Prefix = *optPrefix

	helper := mp.NewMackerelPlugin(cuenoteSrsServerStatus)
	helper.Tempfile = *optTempfile
	helper.Run()
}
