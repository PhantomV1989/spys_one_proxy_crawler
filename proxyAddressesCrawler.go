package spys_one_proxy_crawler

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"
)

// AddressInfo result
type AddressInfo struct {
	ip            string
	port          string
	country       string
	latency       float32
	uptimePercent int
	lastCheckHour int
}

type ProxyClient struct {
	AddressInfo AddressInfo
	Client      *http.Client
}

// ProxyAddressesCrawler ...
type ProxyAddressesCrawler struct {
	Tokens     map[string]int
	Proxies    []ProxyClient
	currentPos int
	blacklist  map[int]bool
}

// InitializeProxyClients ...
func (c *ProxyAddressesCrawler) InitializeProxyClients(country string, maxLatency float32, minUptimePercent int, maxLastCheckHour int) []ProxyClient {
	body := c.getHtmlBody()
	c.getTokens(body)
	addressInfos := c.getAllAddressInfosFromBody(body)
	addressInfos = c.filterAddressInfos(addressInfos, country, maxLatency, minUptimePercent, maxLastCheckHour)
	c.Proxies = []ProxyClient{}
	c.blacklist = map[int]bool{}
	for _, v := range addressInfos {
		dialSocksProxy, err := proxy.SOCKS5("tcp", v.ip+":"+v.port, nil, proxy.Direct)
		if err != nil {
			fmt.Println("Error connecting to proxy:", err)
		} else {
			tr := &http.Transport{
				Dial:            dialSocksProxy.Dial,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := ProxyClient{
				AddressInfo: v,
				Client: &http.Client{
					Transport: tr,
				},
			}
			c.Proxies = append(c.Proxies, client)
		}
	}
	log.Println("Found", len(c.Proxies), "proxies")
	return c.Proxies
}

// ProcessRequestsRoundRobin ...
func (c *ProxyAddressesCrawler) ProcessRequestsRoundRobin(req *http.Request) *http.Response {
	for {
		c.currentPos++
		for {
			if _, e := c.blacklist[c.currentPos]; e {
				c.currentPos++
			} else {
				break
			}
		}
		if c.currentPos >= len(c.Proxies) {
			c.currentPos = 0
		}
		res, err := c.Proxies[c.currentPos].Client.Do(req)
		if err != nil {
			log.Println("Failed proxy: ", c.Proxies[c.currentPos].AddressInfo.ip+":"+c.Proxies[c.currentPos].AddressInfo.port, err)
			c.blacklist[c.currentPos] = true
		} else {
			log.Println("Successful proxy:", c.Proxies[c.currentPos].AddressInfo.ip+":"+c.Proxies[c.currentPos].AddressInfo.port)
			return res
		}
	}
}

func (c *ProxyAddressesCrawler) getHtmlBody() string {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://spys.one/en/socks-proxy-list/", strings.NewReader(""))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Host", " spys.one")
	addHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	bn, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return string(bn)
}

func (c *ProxyAddressesCrawler) filterAddressInfos(ai []AddressInfo, country string, maxLatency float32, minUptimePercent int, maxLastCheckHour int) []AddressInfo {
	r := []AddressInfo{}
	for _, v := range ai {
		if country != "" {
			if v.country != country {
				continue
			}
		}
		if v.latency > maxLatency {
			continue
		}
		if v.uptimePercent < minUptimePercent {
			continue
		}
		if v.lastCheckHour > maxLastCheckHour {
			continue
		}
		r = append(r, v)
	}
	return r
}

func (c *ProxyAddressesCrawler) getAllAddressInfosFromBody(s string) []AddressInfo {
	rows := c.extractTableRows(s)
	r := []AddressInfo{}
	for _, row := range rows {
		r = append(r, c.processRow(row))
	}
	return r
}

func (c *ProxyAddressesCrawler) extractTableRows(s string) []string {
	rows := strings.Split(s, "<tr class=spy1x")
	rows2 := []string{}
	for _, v := range rows {
		if strings.Contains(v, "onmouseover") {
			rows2 = append(rows2, v)
		}
	}
	return rows2
}

func (c *ProxyAddressesCrawler) getTokens(s string) map[string]int { //tokens used to obfuscate port number
	rows := strings.Split(s, "<script type=\"text/javascript\">")
	tokensS := strings.Split(rows[1], "</script>")[0]
	tokens2 := strings.Split(tokensS, ";")
	tokens := map[string]int{}
	for _, v := range tokens2 {
		if len(v) == 0 {
			continue
		}
		vv := strings.Split(v, "=")
		n, e := strconv.Atoi(vv[1])
		if e != nil {
			v2 := strings.Split(vv[1], "^")
			n1, _ := strconv.Atoi(v2[0])
			n2 := tokens[v2[1]]
			n = n1 ^ n2
		}
		tokens[vv[0]] = n
	}
	c.Tokens = tokens
	return tokens
}

func (c *ProxyAddressesCrawler) processRow(s string) AddressInfo {
	cols := strings.Split(s, "td colspan=1>")

	// ip addr
	ip := strings.Split(strings.TrimPrefix(cols[1], "<font class=spy14>"), "<")[0]
	// port
	portRaw := strings.Split(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSuffix(strings.Split(cols[1], "document.write(\"<font class=spy2>:<\\/font>\"+")[1], ")</script></font></td><"), "(", ""), ")", ""), "+")
	port := ""
	for _, v := range portRaw {
		vv := strings.Split(v, "^")
		n := c.Tokens[vv[0]] ^ c.Tokens[vv[1]]
		port += strconv.Itoa(n)
	}

	country := getEnclosedSubstring(cols[4], "<font class=spy14>", "</font>")
	latency, _ := strconv.ParseFloat(getEnclosedSubstring(cols[6], "<font class=spy1>", "</font></td><"), 32)

	uptime := "0"
	if strings.Contains(cols[8], "%") {
		uptime = getEnclosedSubstring(cols[8], "last check status=OK'>", "%")
		if len(uptime) > 3 {
			uptime = getEnclosedSubstring(cols[8], "last check status=OK'><font class=spy14>", "%")
		}
	}
	uptimePercent, _ := strconv.Atoi(uptime)

	lastChkHourS := getEnclosedSubstring(cols[9], " <font class=spy5>(", ")</font></font></td></tr>")
	lastChkHour := 0
	if strings.Contains(lastChkHourS, "hour") {
		lastChkHour, _ = strconv.Atoi(strings.Split(lastChkHourS, " ")[0])
	}

	return AddressInfo{
		ip:            ip,
		port:          port,
		country:       country,
		latency:       float32(latency),
		uptimePercent: uptimePercent,
		lastCheckHour: lastChkHour,
	}
}

func getEnclosedSubstring(s, a, b string) string {
	return strings.Split(strings.Split(s, a)[1], b)[0]
}

func addHeaders(req *http.Request) {
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:87.0) Gecko/20100101 Firefox/87.0")
	req.Header.Add("Accept", " text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Add("Accept-Language", " en-US,en;q=0.5")
	req.Header.Add("Connection", " keep-alive")
	req.Header.Add("Upgrade-Insecure-Requests", " 1")
}
