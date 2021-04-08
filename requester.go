package spys_one_proxy_crawler

type Requester struct {
	country          string
	maxLatency       float32 //<10
	minUptimePercent float32 // >0.5
	maxLastCheckHour int     //<=2
}

func retrieveResult() {

}
