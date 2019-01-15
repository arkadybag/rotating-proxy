package rotateProxy

import (
	"github.com/sirupsen/logrus"
	"github.com/storyicon/golang-proxy/business"
	"github.com/storyicon/golang-proxy/dao"
	"github.com/storyicon/golang-proxy/model"
	"io/ioutil"
	"time"
)

type ProxyServer struct {
	ips chan string
}

func (p *ProxyServer) Init() {
	//go p.startProxyServer()
	go p.getProxyUrl()

}

func (p *ProxyServer) GetProxy() string {
	return <-p.ips
}

func (p *ProxyServer) startProxyServer() {
	logrus.SetOutput(ioutil.Discard)
	logrus.SetLevel(logrus.ErrorLevel)
	go business.StartConsumer()
	go business.StartProducer()
	go business.StartAssessor()
}

func (p *ProxyServer) getProxyUrl() {
	for {
		query := "select * from proxy order by score desc limit 50"

		proxy, err := dao.GetSQLResult("proxy", query)
		if err != nil {
			p.getProxyUrl()
		}

		proxyModels := *proxy.(*[]model.Proxy)

		for _, proxyModel := range proxyModels {
			p.ips <- proxyModel.Content
		}

		time.Sleep(time.Millisecond * 50)
	}
}

func NewProxyServer() *ProxyServer {
	return &ProxyServer{
		ips: make(chan string, 50),
	}
}
