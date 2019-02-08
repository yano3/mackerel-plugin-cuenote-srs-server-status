package mpcuenotesrsserverstatus

import (
	"bufio"
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	//	pp "github.com/k0kubun/pp"
	mp "github.com/mackerelio/go-mackerel-plugin"
)

// CuenoteSrsServerStatusPlugin mackerel plugin for Cuenote SR-S server status
type CuenoteSrsServerStatusPlugin struct {
	Host     string
	Username string
	Password string
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

// FetchMetrics interface for mackerelplugin
func (p CuenoteSrsServerStatusPlugin) FetchMetrics() (map[string]float64, error) {
	Url := "https://" + p.Username + ":" + p.Password + "@" + p.Host + "/api?cmd=get_server_status"
	resp, err := http.Get(Url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]float64)
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if matches := loadAverageReg.FindStringSubmatch(line); len(matches) == 5 {
			ret["loadavg5"], _ = strconv.ParseFloat(matches[3], 32)
		}
	}
	return ret, nil
}

// GraphDefinition interface for mackerelplugin
func (p CuenoteSrsServerStatusPlugin) GraphDefinition() map[string]mp.Graphs {
	var graphdef = map[string]mp.Graphs{
		"cuenote-srs.loadavg": {
			Label: "loadavg",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: "loadavg5", Label: "loadavg5", Diff: false, Stacked: false},
			},
		},
	}
	return graphdef
}

// Do the plugin
func Do() {
	optHost := flag.String("host", "", "Hostname")
	optUser := flag.String("username", "", "Username")
	optPass := flag.String("password", "", "Password")
	optPrefix := flag.String("metric-key-prefix", "cuenote-srs", "Metric key prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	var cuenoteSrsServerStatus CuenoteSrsServerStatusPlugin

	cuenoteSrsServerStatus.Host = *optHost
	cuenoteSrsServerStatus.Username = *optUser
	cuenoteSrsServerStatus.Password = *optPass
	cuenoteSrsServerStatus.Prefix = *optPrefix

	helper := mp.NewMackerelPlugin(cuenoteSrsServerStatus)
	helper.Tempfile = *optTempfile
	helper.Run()
}
