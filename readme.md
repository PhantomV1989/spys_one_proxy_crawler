# spys_one_proxy_crawler
A crawler that uses socks5 proxy servers from https://spys.one/en/socks-proxy-list/


# Design
![alt text](https://github.com/PhantomV1989/spys_one_proxy_crawler/raw/master/design/img.png)

(see test example TestProcessesIntegration)


## 1
Crawler first retrieves a list of socks5 proxy server addresses from https://spys.one/en/socks-proxy-list/
via **InitializeProxyClients** with a set of filters

## 2
Once initialized, you can send your request to the **ProxyAddressesCrawler** class instance where it then send the request to its own lists of proxy server addresses in a roundrobin fashion
