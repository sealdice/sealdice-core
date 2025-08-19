package sealws

// Package websocket 提供了一个与goja兼容的WebSocket客户端实现
// 这个包提供了标准的WebSocket API，可以在任何goja环境中使用

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/gorilla/websocket"

	"sealdice-core/logger"
)

// Logger 日志接口，与helper.go中的Helper方法签名一致
type Logger interface {
	Debug(a ...interface{})
	Debugf(format string, a ...interface{})
	Info(a ...interface{})
	Infof(format string, a ...interface{})
	Warn(a ...interface{})
	Warnf(format string, a ...interface{})
	Error(a ...interface{})
	Errorf(format string, a ...interface{})
}

// defaultLogger 默认日志实现（无操作）
type defaultLogger struct{}

func (l *defaultLogger) Debug(_ ...interface{})            {}
func (l *defaultLogger) Debugf(_ string, _ ...interface{}) {}
func (l *defaultLogger) Info(_ ...interface{})             {}
func (l *defaultLogger) Infof(_ string, _ ...interface{})  {}
func (l *defaultLogger) Warn(_ ...interface{})             {}
func (l *defaultLogger) Warnf(_ string, _ ...interface{})  {}
func (l *defaultLogger) Error(args ...interface{}) {
	// 默认情况下只输出错误日志到标准错误
	logger.M().Errorf("[WebSocket ERROR] %v", args...)
}
func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	// 默认情况下只输出错误日志到标准错误
	logger.M().Errorf("[WebSocket ERROR] "+format, args...)
}

// WebSocketLogger 全局日志实例，用户可以通过SetLogger函数替换
var WebSocketLogger Logger = &defaultLogger{}

// GlobalConnManager 是一个全局的WebSocket管理器 用来最后优雅销毁的
var GlobalConnManager = &WebSocketManager{}

// SetLogger 设置全局日志实例
func SetLogger(logger Logger) {
	WebSocketLogger = logger
}

type (
	// WebSocketModule 是WebSocket模块的根实例
	WebSocketModule struct{}

	// WebSocket 表示WebSocket模块的一个实例
	WebSocket struct {
		rt   *goja.Runtime
		loop *eventloop.EventLoop
	}
	WebSocketManager struct {
		connections []*WebSocketConnection
		mutex       sync.Mutex
	}
)

// Register 注册连接
func (m *WebSocketManager) Register(conn *WebSocketConnection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connections = append(m.connections, conn)
}

// CloseAll 关闭所有连接
func (m *WebSocketManager) CloseAll() {
	// 获取连接副本并清空原列表
	m.mutex.Lock()
	connsCopy := make([]*WebSocketConnection, len(m.connections))
	copy(connsCopy, m.connections)
	m.connections = m.connections[:0]
	m.mutex.Unlock()

	// 在锁外关闭连接，避免死锁
	for _, conn := range connsCopy {
		if conn != nil {
			conn.closeWithoutUnregister() // 使用新方法避免重复注销
		}
	}
}

// Unregister 移除已关闭的连接
func (m *WebSocketManager) Unregister(conn *WebSocketConnection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, c := range m.connections {
		if c == conn {
			m.connections = append(m.connections[:i], m.connections[i+1:]...)
			break
		}
	}
}

// Manager方法结束

// New 返回一个新的WebSocketModule实例
func New() *WebSocketModule {
	return &WebSocketModule{}
}

// NewInstance 为给定的goja运行时创建一个新的WebSocket实例
func (m *WebSocketModule) NewInstance(rt *goja.Runtime, loop *eventloop.EventLoop) *WebSocket {
	return &WebSocket{
		rt:   rt,
		loop: loop,
	}
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
		options = parseWebSocketOptions(args[1])
	} else {
		options = &webSocketOptions{
			headers: make(http.Header),
		}
	}

	// 使用WebSocket实例中的eventloop
	if ws.loop == nil {
		panic(rt.NewTypeError("WebSocket requires event loop. Please provide eventloop when calling Enable()."))
	}

	// 创建WebSocketConnection实例
	conn := &WebSocketConnection{
		rt:         rt,
		loop:       ws.loop,
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
	// 注册到全局管理器
	GlobalConnManager.Register(conn)

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

// parseWebSocketOptions 解析WebSocket选项参数
func parseWebSocketOptions(protocolsVal goja.Value) *webSocketOptions {
	options := &webSocketOptions{
		headers: make(http.Header),
	}

	// 处理protocols参数，可以是字符串或字符串数组
	if !goja.IsUndefined(protocolsVal) && !goja.IsNull(protocolsVal) {
		if protocolsVal.ExportType().Kind() == reflect.String {
			// 单个协议字符串
			options.protocols = []string{protocolsVal.String()}
		} else {
			// 协议数组
			if protocolsArray := protocolsVal.Export(); protocolsArray != nil {
				if protocolSlice, ok := protocolsArray.([]interface{}); ok {
					for _, p := range protocolSlice {
						if str, ok2 := p.(string); ok2 {
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

	WebSocketLogger.Debugf("开始建立WebSocket连接 url=%s protocols=%v", conn.url, options.protocols)

	// 使用现有的dial逻辑
	dialer := &websocket.Dialer{
		HandshakeTimeout:  5 * time.Second, // 缩短超时时间以便测试
		EnableCompression: options.enableCompression,
		Subprotocols:      options.protocols,
		// 允许WSS连接
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		},
	}

	// 建立连接
	wsConn, resp, err := dialer.Dial(conn.url, options.headers)
	if err != nil {
		WebSocketLogger.Errorf("WebSocket连接失败 url=%s error=%s", conn.url, err.Error())
		conn.readyState = CLOSED
		// 直接触发错误事件，triggerError内部会处理事件循环
		conn.triggerError(err)
		return
	}

	WebSocketLogger.Infof("WebSocket连接建立成功 url=%s", conn.url)

	if resp != nil {
		// 设置选择的子协议
		conn.protocol = resp.Header.Get("Sec-WebSocket-Protocol")
		if conn.protocol != "" {
			WebSocketLogger.Debugf("选择的子协议 protocol=%s", conn.protocol)
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
						WebSocketLogger.Errorf("处理WebSocket打开事件失败 url=%s error=%s", conn.url, err.Error())
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
						WebSocketLogger.Errorf("处理WebSocket消息事件失败 url=%s error=%s", conn.url, err.Error())
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
						WebSocketLogger.Errorf("处理WebSocket关闭事件失败 url=%s error=%s", conn.url, err.Error())
					}
				})
			}
		}
	}
}

// triggerError 触发onerror事件
func (conn *WebSocketConnection) triggerError(err error) {
	WebSocketLogger.Errorf("触发WebSocket错误事件 url=%s error=%s", conn.url, err.Error())

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
						WebSocketLogger.Errorf("处理WebSocket错误事件失败 url=%s error=%s", conn.url, err.Error())
					}
				})
			}
		}
	} else {
		WebSocketLogger.Warnf("没有设置onerror处理器 url=%s", conn.url)
	}
}

// Send 向WebSocket连接发送消息
func (conn *WebSocketConnection) Send(message string) error {
	if conn.readyState != OPEN {
		err := errors.New("connection is not open")
		WebSocketLogger.Warnf("尝试在非开放连接上发送消息 readyState=%d url=%s", conn.readyState, conn.url)
		return err
	}

	WebSocketLogger.Debugf("发送WebSocket消息 url=%s messageLength=%d", conn.url, len(message))

	err := conn.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		WebSocketLogger.Errorf("设置WebSocket写入超时失败 url=%s error=%s", conn.url, err.Error())
		return err
	}
	err = conn.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		WebSocketLogger.Errorf("发送WebSocket消息失败 url=%s error=%s", conn.url, err.Error())
	}
	return err
}

// Close 关闭WebSocket连接
func (conn *WebSocketConnection) Close(args ...interface{}) {
	code := CloseNormalClosure
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
	WebSocketLogger.Infof("关闭WebSocket连接 url=%s code=%d reason=%s", conn.url, code, reason)
	conn.closeInternal(true, args...)
}

// closeWithoutUnregister 关闭连接但不从管理器注销（用于CloseAll）
func (conn *WebSocketConnection) closeWithoutUnregister(args ...interface{}) {
	conn.closeInternal(false, args...)
}

// closeInternal 内部关闭方法
func (conn *WebSocketConnection) closeInternal(shouldUnregister bool, args ...interface{}) {
	if conn.readyState == CLOSED || conn.readyState == CLOSING {
		WebSocketLogger.Debugf("连接已经关闭或正在关闭 readyState=%d url=%s", conn.readyState, conn.url)
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

	WebSocketLogger.Infof("主动关闭WebSocket连接 url=%s code=%d reason=%s", conn.url, code, reason)

	if conn.conn != nil {
		err := conn.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason))
		if err != nil {
			WebSocketLogger.Warnf("发送关闭消息失败 url=%s error=%s", conn.url, err.Error())
		}
		_ = conn.conn.Close()
	}

	conn.triggerClose(code, reason)
	if shouldUnregister {
		conn.webSocketCloseConnection()
	} else {
		// 只关闭资源，不从管理器注销
		conn.shutdownOnce.Do(func() {
			close(conn.done)
			if conn.conn != nil {
				_ = conn.conn.Close()
			}
		})
	}
}

// readPump 读取WebSocket消息
func (conn *WebSocketConnection) readPump() {
	defer conn.webSocketCloseConnection()

	WebSocketLogger.Debugf("开始WebSocket消息读取循环 url=%s", conn.url)

	for {
		select {
		case <-conn.done:
			WebSocketLogger.Debugf("WebSocket消息读取循环结束 url=%s", conn.url)
			return
		default:
			messageType, message, err := conn.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					WebSocketLogger.Errorf("WebSocket意外关闭错误 url=%s error=%s", conn.url, err.Error())
					conn.triggerError(err)
				} else {
					WebSocketLogger.Infof("WebSocket连接正常关闭 url=%s error=%s", conn.url, err.Error())
				}
				conn.triggerClose(CloseAbnormalClosure, "connection lost")
				return
			}

			switch messageType {
			case websocket.TextMessage:
				WebSocketLogger.Debugf("接收到文本消息 url=%s messageLength=%d", conn.url, len(message))
				conn.triggerMessage(string(message))
			case websocket.BinaryMessage:
				WebSocketLogger.Debugf("接收到二进制消息 url=%s messageLength=%d", conn.url, len(message))
				conn.triggerMessage(message)
			default:
				WebSocketLogger.Warnf("接收到未知类型消息 url=%s messageType=%d", conn.url, messageType)
			}
		}
	}
}

// webSocketCloseConnection 关闭连接并清理资源
func (conn *WebSocketConnection) webSocketCloseConnection() {
	conn.shutdownOnce.Do(func() {
		WebSocketLogger.Debugf("清理WebSocket连接资源 url=%s", conn.url)
		// 从全局管理器中注销
		GlobalConnManager.Unregister(conn)
		close(conn.done)
		if conn.conn != nil {
			_ = conn.conn.Close()
		}
		WebSocketLogger.Debugf("WebSocket连接资源清理完成 url=%s", conn.url)
	})
}

// Enable 为给定的goja运行时启用WebSocket模块
// 这是一个便利函数，用于快速设置WebSocket模块
func Enable(rt *goja.Runtime, loop *eventloop.EventLoop) {
	module := New()
	instance := module.NewInstance(rt, loop)
	_ = rt.Set("WebSocket", instance.Exports())
}
