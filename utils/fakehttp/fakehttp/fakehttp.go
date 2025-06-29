package fakehttp

import (
	"net/http"
	"pinenut_utils/fakehttp/common"
)

// ListenAndServeTLS 便于替换原版代码 和/或 gin框架的
// 支持CA证书动态签发和普通证书直接使用两种模式
// TODO: Endless足以替换掉下面的这个代码，需求可能并不需要
func ListenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	server, err := common.GetHttpServerInstance(addr, certFile, keyFile, handler)
	if err != nil {
		return err
	}
	return server.ListenAndServeTLS("", "")
}
