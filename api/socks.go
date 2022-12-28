package api

import (
	"encoding/json"
	"fmt"
	"github.com/armon/go-socks5"
	"io/ioutil"
	"net"
	"net/http"
	"sealdice-core/dice"
	"strconv"
	"strings"
	"time"
)

var privateIPBlocks []*net.IPNet

func ipInit() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func getClientIp() ([]string, error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return nil, err
	}

	var lst []string
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !isPrivateIP(ipnet.IP) {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				skip := false
				for _, i := range lst {
					if i == ip {
						skip = true
					}
				}
				if !skip {
					lst = append(lst, ip)
				}
			}
		}
	}

	return lst, nil
}

type IP struct {
	Query string
}

func getip2() string {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return ""
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return ""
	}

	var ip IP
	json.Unmarshal(body, &ip)

	return ip.Query
}

func socksOpen(d *dice.Dice, port int64) (string, error) {
	ipInit()
	ttl := 20 * 60

	//ttl := flag.Int64("time", 25*60, "存活时长，单位秒，默认为25分钟")
	//user := flag.String("user", "", "用户名，默认为空")
	//password := flag.String("password", "", "密码，默认为空")
	//port := flag.Int64("port", 13325, "端口号")
	//flag.Parse()

	ip, err := getClientIp()
	if err != nil {
		return "", err
	}

	_ip := getip2()
	if _ip != "" {
		ip = append(ip, _ip)
	}

	publicIP := strings.Join(ip, ", ")

	d.Logger.Infof("onebot辅助: 将在服务器上开启一个临时socks5服务，端口%d，默认持续时长为20分钟\n", port)
	if publicIP == "" {
		d.Logger.Info("onebot辅助: 未检测到公网IP")
	} else {
		d.Logger.Info("onebot辅助: 可能的公网IP: ", publicIP)
	}
	//fmt.Println("请于服务器管理面板放行你要用的端口(一般为13326即http)，协议TCP")
	//fmt.Println("如果是Windows Server 2012R2及以上系统，请再额外关闭系统防火墙或设置规则放行")

	secure := []socks5.Authenticator{}

	//if *user != "" && *password != "" {
	//	fmt.Println("当前用户", *user)
	//	fmt.Println("当前密码", *password)
	//
	//	cred := socks5.StaticCredentials{
	//		*user: *password,
	//	}
	//	cator := socks5.UserPassAuthenticator{Credentials: cred}
	//	secure = append(secure, cator)
	//}

	// Create a SOCKS5 server
	conf := &socks5.Config{
		AuthMethods: secure,
	}
	server, err := socks5.New(conf)
	if err != nil {
		return publicIP, err
	}

	var l net.Listener
	address := "0.0.0.0:" + strconv.FormatInt(port, 10)
	l, err = net.Listen("tcp", address)
	if err != nil {
		return publicIP, err
	}

	if ttl > 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(ttl))
			d.Logger.Info("onebot辅助: 自动停止")
			l.Close()
		}()
	}
	go server.Serve(l)
	return publicIP, nil

	//if err := server.ListenAndServe("tcp", address); err != nil {
	//	panic(err)
	//}
}
