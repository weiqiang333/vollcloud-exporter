package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"vollcloud-exporter/pkg/vollcloud/grab"
	vclogin "vollcloud-exporter/pkg/vollcloud/login"
)

func init() {
	pflag.String("address", ":9109", "The address on which to expose the web interface and generated Prometheus metrics.")
	pflag.String("configfile", "./config/vollcloud-exporter.yaml", "exporter config file")
}

const namespace = "vollcloud"

type Exporter struct {
	HttpClient       *http.Client
	NodeOnline       prometheus.GaugeVec
	BandwidthTotalGB prometheus.GaugeVec
	BandwidthUsedGB  prometheus.GaugeVec
	BandwidthFreeGB  prometheus.GaugeVec
	BandwidthUsage   prometheus.GaugeVec
	CostUSD          prometheus.GaugeVec
}

func NewExporter(httpClient http.Client) *Exporter {
	return &Exporter{
		HttpClient: &httpClient,
		NodeOnline: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "node_online",
				Help:      "server run status value, Disabled=0 / Online=1",
			}, []string{"ip_address", "hostname", "vm_type", "memory", "disk"}),
		BandwidthTotalGB: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "bandwidth_total_GB",
				Help:      "宽带流量当月总数 GB",
			}, []string{"ip_address", "hostname"}),
		BandwidthUsedGB: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "bandwidth_used_GB",
				Help:      "宽带流量当月使用总数 GB",
			}, []string{"ip_address", "hostname"}),
		BandwidthFreeGB: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "bandwidth_free_GB",
				Help:      "宽带流量当月剩余总数 GB",
			}, []string{"ip_address", "hostname"}),
		BandwidthUsage: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "bandwidth_usage",
				Help:      "宽带流量使用百分比 %",
			}, []string{"ip_address", "hostname"}),
		CostUSD: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cost_usb",
				Help:      "服务成本/USB",
			}, []string{"ip_address", "hostname", "date_start", "date_end", "cost_cycle"}),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.NodeOnline.Describe(ch)
	e.BandwidthTotalGB.Describe(ch)
	e.BandwidthFreeGB.Describe(ch)
	e.BandwidthUsage.Describe(ch)
	e.BandwidthUsedGB.Describe(ch)
	e.CostUSD.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.NodeOnline.Reset()
	e.BandwidthTotalGB.Reset()
	e.BandwidthUsedGB.Reset()
	e.BandwidthFreeGB.Reset()
	e.BandwidthUsage.Reset()
	e.CostUSD.Reset()

	httpClient := *e.HttpClient
	vcClientarea := grab.NewClientarea(httpClient)
	vcClientarea.Get()
	_, err := vcClientarea.IfUserLogin()
	if err != nil {
		log.Println("Failed grab in login, About to sign in again from.")
		httpClient = *loginGetClient()
		e.HttpClient = &httpClient
	}

	vsServices := grab.NewServices(httpClient)
	vsServices.Get()
	vsServices.GetProductIdUrls()
	idUrls := vsServices.IdUrls
	for _, idUrl := range idUrls {
		vsProductdetails := grab.NewProductdetails(httpClient)
		if err := vsProductdetails.Get(idUrl); err != nil {
			continue
		}
		if err := vsProductdetails.CreateStats(); err != nil {
			continue
		}
		e.NodeOnline.WithLabelValues(vsProductdetails.Stats.IpAddress, vsProductdetails.Stats.Hostname, vsProductdetails.Stats.Type, vsProductdetails.Stats.Memory, vsProductdetails.Stats.Disk).Set(vsProductdetails.Stats.Status)
		e.BandwidthTotalGB.WithLabelValues(vsProductdetails.Stats.IpAddress, vsProductdetails.Stats.Hostname).Set(vsProductdetails.Stats.BandwidthTotalGB)
		e.BandwidthUsedGB.WithLabelValues(vsProductdetails.Stats.IpAddress, vsProductdetails.Stats.Hostname).Set(vsProductdetails.Stats.BandwidthUsedGB)
		e.BandwidthFreeGB.WithLabelValues(vsProductdetails.Stats.IpAddress, vsProductdetails.Stats.Hostname).Set(vsProductdetails.Stats.BandwidthFreeGB)
		e.BandwidthUsage.WithLabelValues(vsProductdetails.Stats.IpAddress, vsProductdetails.Stats.Hostname).Set(vsProductdetails.Stats.BandwidthUsage)
		if err := vsProductdetails.GetProductDetails(); err != nil {
			log.Println(err.Error())
			continue
		}
		for _, cost := range vsProductdetails.SplitCostCycle() {
			e.CostUSD.WithLabelValues(vsProductdetails.Stats.IpAddress, vsProductdetails.Stats.Hostname, cost.DateStart, cost.DateEnd, cost.CostCycle).Set(cost.BlendedCostUSD)
		}
	}

	e.NodeOnline.Collect(ch)
	e.BandwidthTotalGB.Collect(ch)
	e.BandwidthUsedGB.Collect(ch)
	e.BandwidthFreeGB.Collect(ch)
	e.BandwidthUsage.Collect(ch)
	e.CostUSD.Collect(ch)
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

func loginGetClient() *http.Client {
	vcLogin := vclogin.NewLogin()
	_, err := vcLogin.Login()
	if err != nil {
		log.Println("Failed grab in login")
	}
	return vcLogin.HttpClient
}

func main() {
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		log.Fatal("Fatal error BindPFlags: %w", err.Error())
	}
	fmt.Println("load config file ", viper.GetString("configfile"))
	viper.SetConfigType("yaml")
	viper.SetConfigFile(viper.GetString("configfile"))
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	httpClient := *loginGetClient()

	prometheus.MustRegister(NewExporter(httpClient))

	// http server
	listenAddress := viper.GetString("address")
	fmt.Printf("http server start, address %s/metrics\n", listenAddress)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/reload", reloadConfig)
	go func() {
		time.Sleep(time.Second)
		if err := open(fmt.Sprintf("http://127.0.0.1:9109/metrics")); err != nil {
			log.Println(err.Error())
		}
	}()
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatal("Fatal error http: %w", err)
	}
}
