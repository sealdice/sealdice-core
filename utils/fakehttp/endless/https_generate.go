package endless

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// generateSelfSignedCert 会生成一个自签名证书并返回证书和私钥的临时路径
func generateSelfSignedCert() (certPath, keyPath string, err error) {
	// 存储证书和私钥
	certFile, err := os.Create("cert.crt")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp certificate file: %v", err)
	}
	defer certFile.Close()

	keyFile, err := os.Create("cert.key")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp key file: %v", err)
	}
	defer keyFile.Close()

	// 生成 ECDSA 私钥
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %v", err)
	}

	// 生成随机序列号
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %v", err)
	}

	// 获取所有网络接口的IP地址
	var ipAddresses []net.IP
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			// 跳过未启用的接口
			if iface.Flags&net.FlagUp == 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
					// 添加非回环地址
					ipAddresses = append(ipAddresses, ipNet.IP)
				}
			}
		}
	}
	// 确保包含基本的localhost地址
	ipAddresses = append(ipAddresses, net.IPv4(127, 0, 0, 1), net.IPv6loopback)

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"good.sealdice.com"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 证书有效期一年
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// 添加SAN (Subject Alternative Names) 包含所有网络接口IP
		IPAddresses: ipAddresses,
		DNSNames: []string{
			"weizaima.com",
			"*.sealdice.com",
		},
	}

	// 生成自签名证书
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %v", err)
	}

	// 将证书写入临时文件
	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return "", "", fmt.Errorf("failed to encode certificate to PEM: %v", err)
	}

	// 将私钥写入临时文件
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %v", err)
	}
	err = pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	if err != nil {
		return "", "", fmt.Errorf("failed to encode private key to PEM: %v", err)
	}

	return certFile.Name(), keyFile.Name(), nil
}

// checkOrGenerateCert 检查证书和私钥文件是否存在，如果不存在则生成临时的
func CheckOrGenerateCert(certPath, keyPath string) (string, string, error) {
	// 如果文件不存在，生成临时文件
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return generateSelfSignedCert()
	}
	// 如果文件存在，直接返回路径
	return certPath, keyPath, nil
}
