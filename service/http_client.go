package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"one-api/common"
	"time"

	"golang.org/x/net/proxy"
)

var httpClient *http.Client

func InitHttpClient() {
	if common.RelayTimeout == 0 {
		httpClient = &http.Client{}
	} else {
		httpClient = &http.Client{
			Timeout: time.Duration(common.RelayTimeout) * time.Second,
		}
	}
}

func GetHttpClient() *http.Client {
	return httpClient
}

// NewProxyHttpClient 创建支持代理的 HTTP 客户端
func NewProxyHttpClient(proxyURL string) (*http.Client, error) {
	return NewCustomHttpClient(proxyURL, false)
}

// NewCustomHttpClient 创建自定义 HTTP 客户端，支持代理和跳过TLS验证
func NewCustomHttpClient(proxyURL string, insecureSkipVerify bool) (*http.Client, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	if proxyURL == "" {
		return &http.Client{
			Transport: transport,
		}, nil
	}

	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	switch parsedURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(parsedURL)
		return &http.Client{
			Transport: transport,
		}, nil

	case "socks5", "socks5h":
		// 获取认证信息
		var auth *proxy.Auth
		if parsedURL.User != nil {
			auth = &proxy.Auth{
				User:     parsedURL.User.Username(),
				Password: "",
			}
			if password, ok := parsedURL.User.Password(); ok {
				auth.Password = password
			}
		}

		// 创建 SOCKS5 代理拨号器
		// proxy.SOCKS5 使用 tcp 参数，所有 TCP 连接包括 DNS 查询都将通过代理进行。行为与 socks5h 相同
		dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}

		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		
		return &http.Client{
			Transport: transport,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", parsedURL.Scheme)
	}
}
