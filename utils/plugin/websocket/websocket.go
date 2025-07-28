// Package websocket 提供了一个与goja兼容的WebSocket客户端实现
// 这个包提供了标准的WebSocket API，可以在任何goja环境中使用
package sealws

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/gorilla/websocket"
)

// Logger 日志接口，允许用户注入自己的日志实现
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// defaultLogger 默认日志实现（无操作）
type defaultLogger struct{}

func (l *defaultLogger) Debug(_ string, _ ...interface{}) {}
func (l *defaultLogger) Info(_ string, _ ...interface{})  {}
func (l *defaultLogger) Warn(_ string, _ ...interface{})  {}
func (l *defaultLogger) Error(msg string, args ...interface{}) {
	// 默认情况下只输出错误日志到标准错误
	fmt.Printf("[WebSocket ERROR] "+msg+"\n", args...)
}

// WebSocketLogger 全局日志实例，用户可以通过SetLogger函数替换
var WebSocketLogger Logger = &defaultLogger{}

// SetLogger 设置全局日志实例
func SetLogger(logger Logger) {
	WebSocketLogger = logger
}

type (
	// WebSocketModule 是WebSocket模块的根实例
	WebSocketModule struct{}

	// WebSocket 表示WebSocket模块的一个实例
	WebSocket struct {
		rt *goja.Runtime
	}
)

// New 返回一个新的WebSocketModule实例
func New() *WebSocketModule {
	return &WebSocketModule{}
}

// NewInstance 为给定的goja运行时创建一个新的WebSocket实例
func (m *WebSocketModule) NewInstance(rt *goja.Runtime) *WebSocket {
	ws := &WebSocket{
		rt: rt,
	}
	return ws
}

// Exports 返回模块的导出对象 - 直接返回WebSocket构造函数
func (ws *WebSocket) Exports() goja.Value {
	// 使用DefineConstructor创建构造函数
	constructorFunc := func(call goja.ConstructorCall) *goja.Object {
		// 转换为FunctionCall
		funcCall := goja.FunctionCall{
			This:      call.This,
			Arguments: call.Arguments,
		}
		result := ws.NewWebSocketConnection(funcCall)
		return result.ToObject(ws.rt)
	}

	// 创建构造函数
	webSocketConstructor := ws.rt.ToValue(constructorFunc)
	constructorObj := webSocketConstructor.ToObject(ws.rt)

	// 设置静态常量到构造函数上
	_ = constructorObj.Set("CONNECTING", CONNECTING)
	_ = constructorObj.Set("OPEN", OPEN)
	_ = constructorObj.Set("CLOSING", CLOSING)
	_ = constructorObj.Set("CLOSED", CLOSED)

	// 设置prototype属性
	prototypeObj := ws.rt.NewObject()
	_ = constructorObj.Set("prototype", prototypeObj)

	return webSocketConstructor
}

// WebSocketConnection 是返回给JavaScript的WebSocket连接表示
type WebSocketConnection struct {
	rt           *goja.Runtime
	loop         *eventloop.EventLoop
	ctx          context.Context
	conn         *websocket.Conn
	scheduled    chan goja.Callable
	done         chan struct{}
	shutdownOnce sync.Once
	jsObject     *goja.Object

	// WebSocket API 属性
	url        string
	protocol   string
	readyState int

	// 事件处理器 (WebSocket标准)
	onopen    goja.Value
	onmessage goja.Value
	onclose   goja.Value
	onerror   goja.Value
}

// WebSocket readyState 常量
const (
	CONNECTING = 0
	OPEN       = 1
	CLOSING    = 2
	CLOSED     = 3
)

// WebSocket 关闭状态码常量 (RFC 6455)
const (
	CloseNormalClosure           = 1000 // 正常关闭
	CloseGoingAway               = 1001 // 端点离开
	CloseProtocolError           = 1002 // 协议错误
	CloseUnsupportedData         = 1003 // 不支持的数据类型
	CloseNoStatusRcvd            = 1005 // 没有收到状态码
	CloseAbnormalClosure         = 1006 // 异常关闭
	CloseInvalidFramePayloadData = 1007 // 无效的帧载荷数据
	ClosePolicyViolation         = 1008 // 策略违反
	CloseMessageTooBig           = 1009 // 消息过大
	CloseMandatoryExtension      = 1010 // 强制扩展
	CloseInternalServerErr       = 1011 // 内部服务器错误
	CloseTLSHandshake            = 1015 // TLS握手失败
)

type webSocketOptions struct {
	headers           http.Header
	enableCompression bool
	protocols         []string
}

const writeWait = 10 * time.Second

// NewWebSocketConnection 创建一个新的WebSocket连接 (构造函数)
// 签名: new WebSocket(url: string, protocols?: string | string[])
func (ws *WebSocket) NewWebSocketConnection(call goja.FunctionCall) goja.Value {
	rt := ws.rt
	args := call.Arguments

	if len(args) == 0 {
		panic(rt.NewTypeError("WebSocket constructor requires at least 1 argument (url)"))
	}

	url := args[0].String()
	var options *webSocketOptions

	// 第二个参数是 protocols (可选)
	if len(args) > 1 && !goja.IsUndefined(args[1]) && !goja.IsNull(args[1]) {
		options = parseWebSocketOptions(rt, args[1])
	} else {
		options = &webSocketOptions{
			headers: make(http.Header),
		}
	}

	// 强制要求事件循环
	var loop *eventloop.EventLoop
	if loopVal := rt.Get("__eventloop__"); loopVal != nil && !goja.IsUndefined(loopVal) {
		if l, ok := loopVal.Export().(*eventloop.EventLoop); ok {
			loop = l
		}
	}
	if loop == nil {
		panic(rt.NewTypeError("WebSocket requires goja_nodejs event loop. Please use eventloop.Run() and set __eventloop__ variable."))
	}

	// 创建WebSocketConnection实例
	conn := &WebSocketConnection{
		rt:         rt,
		loop:       loop,
		url:        url,
		protocol:   "", // 协议将在连接建立后设置
		readyState: CONNECTING,
		scheduled:  make(chan goja.Callable),
		done:       make(chan struct{}),
		onopen:     goja.Undefined(),
		onmessage:  goja.Undefined(),
		onclose:    goja.Undefined(),
		onerror:    goja.Undefined(),
	}

	// 创建JavaScript对象并绑定属性和方法
	conn.bindWebSocketMethods()

	// 异步建立连接
	go conn.connect(options)

	return rt.ToValue(conn.jsObject)
}

// bindWebSocketMethods 创建JavaScript对象并绑定WebSocket标准的属性和方法
func (conn *WebSocketConnection) bindWebSocketMethods() {
	rt := conn.rt
	obj := rt.NewObject()

	// 绑定WebSocket标准属性
	_ = obj.DefineAccessorProperty("readyState", rt.ToValue(func() int {
		return conn.readyState
	}), goja.Undefined(), goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("url", rt.ToValue(func() string {
		return conn.url
	}), goja.Undefined(), goja.FLAG_FALSE, goja.FLAG_TRUE)

	// WebSocket 标准中有 protocol 属性
	_ = obj.DefineAccessorProperty("protocol", rt.ToValue(func() string {
		return conn.protocol
	}), goja.Undefined(), goja.FLAG_FALSE, goja.FLAG_TRUE)

	// 绑定事件处理器属性
	_ = obj.DefineAccessorProperty("onopen", rt.ToValue(func() goja.Value {
		return conn.onopen
	}), rt.ToValue(func(val goja.Value) {
		conn.onopen = val
	}), goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("onmessage", rt.ToValue(func() goja.Value {
		return conn.onmessage
	}), rt.ToValue(func(val goja.Value) {
		conn.onmessage = val
	}), goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("onclose", rt.ToValue(func() goja.Value {
		return conn.onclose
	}), rt.ToValue(func(val goja.Value) {
		conn.onclose = val
	}), goja.FLAG_FALSE, goja.FLAG_TRUE)

	_ = obj.DefineAccessorProperty("onerror", rt.ToValue(func() goja.Value {
		return conn.onerror
	}), rt.ToValue(func(val goja.Value) {
		conn.onerror = val
	}), goja.FLAG_FALSE, goja.FLAG_TRUE)

	// 绑定WebSocket标准方法
	_ = obj.Set("send", rt.ToValue(func(message string) {
		if err := conn.Send(message); err != nil {
			conn.triggerError(err)
		}
	}))
	_ = obj.Set("close", rt.ToValue(func(args ...interface{}) {
		conn.Close(args...)
	}))

	// 注意：常量在构造函数上，不在实例上

	conn.jsObject = obj
}

// parseWebSocketOptions 解析WebSocket选项参数 TODO: 这个gojaRunTime是不是没用了
func parseWebSocketOptions(_ *goja.Runtime, protocolsVal goja.Value) *webSocketOptions {
	options := &webSocketOptions{
		headers: make(http.Header),
	}

	// 处理protocols参数，可以是字符串或字符串数组
	if !goja.IsUndefined(protocolsVal) && !goja.IsNull(protocolsVal) {
		if protocolsVal.ExportType().Kind().String() == "string" {
			// 单个协议字符串
			options.protocols = []string{protocolsVal.String()}
		} else {
			// 协议数组
			if protocolsArray := protocolsVal.Export(); protocolsArray != nil {
				if protocolSlice, ok := protocolsArray.([]interface{}); ok {
					for _, p := range protocolSlice {
						if str, ok := p.(string); ok {
							options.protocols = append(options.protocols, str)
						}
					}
				}
			}
		}
	}

	return options
}

// connect 建立WebSocket连接
func (conn *WebSocketConnection) connect(options *webSocketOptions) {
	ctx := context.Background()
	conn.ctx = ctx

	WebSocketLogger.Debug("开始建立WebSocket连接", "url", conn.url, "protocols", options.protocols)

	// 使用现有的dial逻辑
	dialer := &websocket.Dialer{
		HandshakeTimeout:  5 * time.Second, // 缩短超时时间以便测试
		EnableCompression: options.enableCompression,
		Subprotocols:      options.protocols,
		// 允许WSS连接
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// 建立连接
	wsConn, resp, err := dialer.Dial(conn.url, options.headers)
	if err != nil {
		WebSocketLogger.Error("WebSocket连接失败", "url", conn.url, "error", err.Error())
		conn.readyState = CLOSED
		// 直接触发错误事件，triggerError内部会处理事件循环
		conn.triggerError(err)
		return
	}

	WebSocketLogger.Info("WebSocket连接建立成功", "url", conn.url)

	if resp != nil {
		// 设置选择的子协议
		conn.protocol = resp.Header.Get("Sec-WebSocket-Protocol")
		if conn.protocol != "" {
			WebSocketLogger.Debug("选择的子协议", "protocol", conn.protocol)
		}
		_ = resp.Body.Close()
	}

	conn.conn = wsConn
	conn.readyState = OPEN

	// 触发onopen事件
	conn.triggerOpen()

	// 启动消息读取循环
	go conn.readPump()
}

// triggerOpen 触发onopen事件
func (conn *WebSocketConnection) triggerOpen() {
	if !goja.IsUndefined(conn.onopen) && !goja.IsNull(conn.onopen) {
		if fn, ok := goja.AssertFunction(conn.onopen); ok {
			if conn.loop != nil {
				conn.loop.RunOnLoop(func(vm *goja.Runtime) {
					// 创建Event对象
					event := vm.NewObject()
					_ = event.Set("type", "open")
					_, err := fn(goja.Undefined(), event)
					if err != nil {
						WebSocketLogger.Error("处理WebSocket打开事件失败", "url", conn.url, "error", err.Error())
					}
				})
			}
		}
	}
}

// triggerMessage 触发onmessage事件
func (conn *WebSocketConnection) triggerMessage(data interface{}) {
	if !goja.IsUndefined(conn.onmessage) && !goja.IsNull(conn.onmessage) {
		if fn, ok := goja.AssertFunction(conn.onmessage); ok {
			if conn.loop != nil {
				conn.loop.RunOnLoop(func(vm *goja.Runtime) {
					// 创建MessageEvent对象
					event := vm.NewObject()
					_ = event.Set("data", data)
					_ = event.Set("type", "message")
					_, err := fn(goja.Undefined(), event)
					if err != nil {
						WebSocketLogger.Error("处理WebSocket消息事件失败", "url", conn.url, "error", err.Error())
					}
				})
			}
		}
	}
}

// triggerClose 触发onclose事件
func (conn *WebSocketConnection) triggerClose(code int, reason string) {
	conn.readyState = CLOSED
	if !goja.IsUndefined(conn.onclose) && !goja.IsNull(conn.onclose) {
		if fn, ok := goja.AssertFunction(conn.onclose); ok {
			if conn.loop != nil {
				conn.loop.RunOnLoop(func(vm *goja.Runtime) {
					// 创建CloseEvent对象
					event := vm.NewObject()
					_ = event.Set("code", code)
					_ = event.Set("reason", reason)
					_ = event.Set("wasClean", code == CloseNormalClosure) // 正常关闭
					_ = event.Set("type", "close")
					_, err := fn(goja.Undefined(), event)
					if err != nil {
						WebSocketLogger.Error("处理WebSocket关闭事件失败", "url", conn.url, "error", err.Error())
					}
				})
			}
		}
	}
}

// triggerError 触发onerror事件
func (conn *WebSocketConnection) triggerError(err error) {
	WebSocketLogger.Error("触发WebSocket错误事件", "url", conn.url, "error", err.Error())

	if !goja.IsUndefined(conn.onerror) && !goja.IsNull(conn.onerror) {
		if fn, ok := goja.AssertFunction(conn.onerror); ok {
			if conn.loop != nil {
				conn.loop.RunOnLoop(func(vm *goja.Runtime) {
					// 创建ErrorEvent对象
					event := vm.NewObject()
					_ = event.Set("error", err.Error())
					_ = event.Set("type", "error")
					_, err = fn(goja.Undefined(), event)
					if err != nil {
						WebSocketLogger.Error("处理WebSocket错误事件失败", "url", conn.url, "error", err.Error())
					}
				})
			}
		}
	} else {
		WebSocketLogger.Warn("没有设置onerror处理器", "url", conn.url)
	}
}

// Send 向WebSocket连接发送消息
func (conn *WebSocketConnection) Send(message string) error {
	if conn.readyState != OPEN {
		err := errors.New("connection is not open")
		WebSocketLogger.Warn("尝试在非开放连接上发送消息", "readyState", conn.readyState, "url", conn.url)
		return err
	}

	WebSocketLogger.Debug("发送WebSocket消息", "url", conn.url, "messageLength", len(message))

	err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		WebSocketLogger.Error("设置WebSocket写入超时失败", "url", conn.url, "error", err.Error())
		return err
	}
	err = conn.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		WebSocketLogger.Error("发送WebSocket消息失败", "url", conn.url, "error", err.Error())
	}
	return err
}

// Close 关闭WebSocket连接
func (conn *WebSocketConnection) Close(args ...interface{}) {
	if conn.readyState == CLOSED || conn.readyState == CLOSING {
		WebSocketLogger.Debug("连接已经关闭或正在关闭", "readyState", conn.readyState, "url", conn.url)
		return
	}

	conn.readyState = CLOSING

	code := CloseNormalClosure // 正常关闭
	reason := ""

	if len(args) > 0 {
		if c, ok := args[0].(int); ok {
			code = c
		}
	}

	if len(args) > 1 {
		if r, ok := args[1].(string); ok {
			reason = r
		}
	}

	WebSocketLogger.Info("主动关闭WebSocket连接", "url", conn.url, "code", code, "reason", reason)

	if conn.conn != nil {
		err := conn.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason))
		if err != nil {
			WebSocketLogger.Warn("发送关闭消息失败", "url", conn.url, "error", err.Error())
		}
		_ = conn.conn.Close()
	}

	conn.triggerClose(code, reason)
	conn.webSocketCloseConnection()
}

// readPump 读取WebSocket消息
func (conn *WebSocketConnection) readPump() {
	defer conn.webSocketCloseConnection()

	WebSocketLogger.Debug("开始WebSocket消息读取循环", "url", conn.url)

	for {
		select {
		case <-conn.done:
			WebSocketLogger.Debug("WebSocket消息读取循环结束", "url", conn.url)
			return
		default:
			messageType, message, err := conn.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					WebSocketLogger.Error("WebSocket意外关闭错误", "url", conn.url, "error", err.Error())
					conn.triggerError(err)
				} else {
					WebSocketLogger.Info("WebSocket连接正常关闭", "url", conn.url, "error", err.Error())
				}
				conn.triggerClose(CloseAbnormalClosure, "connection lost")
				return
			}

			switch messageType {
			case websocket.TextMessage:
				WebSocketLogger.Debug("接收到文本消息", "url", conn.url, "messageLength", len(message))
				conn.triggerMessage(string(message))
			case websocket.BinaryMessage:
				WebSocketLogger.Debug("接收到二进制消息", "url", conn.url, "messageLength", len(message))
				conn.triggerMessage(message)
			default:
				WebSocketLogger.Warn("接收到未知类型消息", "url", conn.url, "messageType", messageType)
			}
		}
	}
}

// webSocketCloseConnection 关闭连接并清理资源
func (conn *WebSocketConnection) webSocketCloseConnection() {
	conn.shutdownOnce.Do(func() {
		close(conn.done)
		if conn.conn != nil {
			_ = conn.conn.Close()
		}
	})
}

// Enable 为给定的goja运行时启用WebSocket模块
// 这是一个便利函数，用于快速设置WebSocket模块
func Enable(rt *goja.Runtime) {
	module := New()
	instance := module.NewInstance(rt)
	_ = rt.Set("WebSocket", instance.Exports())
}
