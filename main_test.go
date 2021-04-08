package spys_one_proxy_crawler

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/proxy"
)

func readSampleText(name string) string {
	dir, _ := os.Getwd()
	x, _ := ioutil.ReadFile(filepath.Join(dir, "test_data", name))
	return string(x)
}

func TestGetRequest(t *testing.T) {
	// socks5 proxy here
	dialSocksProxy, err := proxy.SOCKS5("tcp", "70.166.167.36:4145", nil, proxy.Direct)
	if err != nil {
		fmt.Println("Error connecting to proxy:", err)
	}
	tr := &http.Transport{
		Dial:            dialSocksProxy.Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Create client
	myClient := &http.Client{
		Transport: tr,
	}

	myClient.Transport = tr
	resp, err := myClient.Get("http://neverssl.com")
	if err != nil {
		panic(err)
	}
	dd, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	print(string(dd))
}

func TestProcesses(t *testing.T) {
	crawler := ProxyAddressesCrawler{}
	sample1Body := readSampleText("sample1")
	tkns := crawler.getTokens(sample1Body)
	if tkns["a1d4"] != 1492 {
		t.Error("wrong token value", tkns["a1d4"])
	}

	rows := crawler.extractTableRows(sample1Body)
	if len(rows) != 30 {
		t.Error("wrong table row count", len(rows))
	}

	for _, row := range rows {
		crawler.processRow(row)
	}

	proxyAddressInfos := crawler.getAllAddressInfosFromBody(sample1Body)
	if proxyAddressInfos[0].ip != "185.117.244.136" {
		t.Error("ip error", proxyAddressInfos[0].ip)
	}
	if proxyAddressInfos[0].port != "9050" {
		t.Error("port error", proxyAddressInfos[0].port)
	}

	proxyAddressInfosF := crawler.filterAddressInfos(proxyAddressInfos, Countries.US, 10, 50, 3)
	if len(proxyAddressInfosF) != 11 {
		t.Error("wrong result count", len(proxyAddressInfosF))
	}
	for _, v := range proxyAddressInfosF {
		if v.country != Countries.US {
			t.Error("wrong country", v.country)
		}
		if v.uptimePercent < 50 {
			t.Error("wrong uptime", v.uptimePercent)
		}
		if v.latency > 10 {
			t.Error("wrong latency", v.latency)
		}
		if v.lastCheckHour > 3 {
			t.Error("wrong last check hour", v.lastCheckHour)
		}
	}

	proxyAddressInfosF2 := crawler.filterAddressInfos(proxyAddressInfos, Countries.empty, 10, 50, 3)
	if len(proxyAddressInfosF2) != 23 {
		t.Error("wrong result count", len(proxyAddressInfosF2))
	}
}

func TestProcessesIntegration(t *testing.T) { // requires online
	/*
		curl -k --socks5 72.206.181.103:4145 https://duckduckgo.com/
		http://google.com
	*/
	crawler := ProxyAddressesCrawler{}
	addressInfos := crawler.InitializeProxyClients(Countries.empty, 5, 80, 2)
	if len(addressInfos) == 0 {
		t.Error("No results found", 0)
	}

	req, err := http.NewRequest("GET", "https://www.facebook.com/", strings.NewReader(""))
	if err != nil {
		panic(err)
	}
	res := crawler.ProcessRequestsRoundRobin(req)
	dd, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	print(string(dd))

	res = crawler.ProcessRequestsRoundRobin(req)
	dd, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	print(string(dd))
}
