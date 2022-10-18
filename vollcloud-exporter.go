package main

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	pflag.String("address", ":9109", "The address on which to expose the web interface and generated Prometheus metrics.")
	pflag.String("configfile", "./config/vollcloud-exporter.yaml", "exporter config file")
}

type Exporter struct {
	NodeOnline prometheus.GaugeVec
}

func NewExporter() *Exporter {
	return &Exporter{
		NodeOnline: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "node_suspended",
				Help: "server run status value, Running=0 / Suspended=1",
			}, []string{"ip_address", "node_ip", "hostname", "vm_type", "node_location", "os"}),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.NodeOnline.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.NodeOnline.WithLabelValues("")

	e.NodeOnline.Collect(ch)
}

func reloadConfig(w http.ResponseWriter, _ *http.Request) {
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		fmt.Println(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	fmt.Println(fmt.Sprintf("reload config file: %s", viper.ConfigFileUsed()))
	io.WriteString(w, fmt.Sprintf("rereload config file: %s", viper.ConfigFileUsed()))
}

func open(uri string) error {
	var commands = map[string]string{
		"windows": "start",
		"darwin":  "open",
		"linux":   "xdg-open",
	}
	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("%s platform ？？？", runtime.GOOS)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "start ", uri)
		//cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	} else {
		cmd = exec.Command(run, uri)
	}
	return cmd.Start()
}

func main() {
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		fmt.Println("Fatal error BindPFlags: %w", err.Error())
	}
	fmt.Println("load config file ", viper.GetString("configfile"))
	viper.SetConfigType("yaml")
	viper.SetConfigFile(viper.GetString("configfile"))
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	prometheus.MustRegister(NewExporter())
	// http server
	fmt.Printf("http server start, address %s/metrics\n", viper.GetString("address"))
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/reload", reloadConfig)
	if err := open(fmt.Sprintf("http://127.0.0.1:9109/metrics")); err != nil {
		fmt.Println(err.Error())
	}
	if err := http.ListenAndServe(viper.GetString("exporter.address"), nil); err != nil {
		fmt.Println("Fatal error http: %w", err)
	}
}
