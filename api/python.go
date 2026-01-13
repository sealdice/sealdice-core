package api

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"sealdice-core/dice"
)

// Python扩展相关API
// 司马go-python功能有限，暂时只能实现基本的扩展管理功能

// pythonList 列出所有Python扩展
func pythonList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	exts := myDice.ListPythonExtensions()
	var result []map[string]interface{}

	for _, ext := range exts {
		result = append(result, map[string]interface{}{
			"name":    ext.Name,
			"version": ext.Version,
			"author":  ext.Author,
			"brief":   ext.Brief,
			"loaded":  ext.IsLoaded,
		})
	}

	return c.JSON(200, map[string]interface{}{
		"extensions": result,
	})
}

// pythonUpload 上传
func pythonUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return Error(&c, "上传失败: "+err.Error(), Response{})
	}

	if file == nil {
		return Error(&c, "未找到文件", Response{})
	}

	// 检查文件扩展名
	if !strings.HasSuffix(file.Filename, ".py") {
		return Error(&c, "只允许上传.py文件", Response{})
	}

	// 保存文件到extensions/python目录
	uploadDir := filepath.Join("./extensions", "python")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return Error(&c, "创建目录失败: "+err.Error(), Response{})
	}

	dst := filepath.Join(uploadDir, file.Filename)
	src, err := file.Open()
	if err != nil {
		return Error(&c, "打开上传文件失败: "+err.Error(), Response{})
	}
	defer src.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return Error(&c, "创建目标文件失败: "+err.Error(), Response{})
	}
	defer dstFile.Close()

	// 复制文件内容
	if _, err := dstFile.ReadFrom(src); err != nil {
		return Error(&c, "保存文件失败: "+err.Error(), Response{})
	}

	return Success(&c, Response{
		"filename": file.Filename,
		"message":  "Python扩展上传成功",
	})
}

// pythonLoad 加载扩展
func pythonLoad(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var req struct {
		Filename string `json:"filename"`
	}

	if err := c.Bind(&req); err != nil {
		return Error(&c, "参数错误", Response{})
	}

	if req.Filename == "" {
		return Error(&c, "文件名不能为空", Response{})
	}

	// 构建完整路径
	filePath := filepath.Join("./extensions", "python", req.Filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return Error(&c, "文件不存在", Response{})
	}

	// 加载Python扩展
	err := myDice.LoadPythonExtension(filePath)
	if err != nil {
		return Error(&c, "加载失败: "+err.Error(), Response{})
	}

	return Success(&c, Response{
		"message": "Python扩展加载成功",
	})
}

// pythonUnload 卸载Python扩展
func pythonUnload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var req struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&req); err != nil {
		return Error(&c, "参数错误", Response{})
	}

	if req.Name == "" {
		return Error(&c, "扩展名不能为空", Response{})
	}

	// 卸载Python扩展
	err := myDice.UnloadPythonExtension(req.Name)
	if err != nil {
		return Error(&c, "卸载失败: "+err.Error(), Response{})
	}

	return Success(&c, Response{
		"message": "Python扩展卸载成功",
	})
}

// pythonExecute 执行Python代码
func pythonExecute(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := c.Bind(&req); err != nil {
		return Error(&c, "参数错误", Response{})
	}

	if req.Code == "" {
		return Error(&c, "代码不能为空", Response{})
	}

	// 执行 Python 代码（安全超时与输出捕获）
	pyExec := "python3"
	if dice.GlobalPythonManager != nil && dice.GlobalPythonManager.PythonExecutable() != "" {
		pyExec = dice.GlobalPythonManager.PythonExecutable()
	}

	// 限制代码大小，避免过大负载
	if len(req.Code) > 100*1024 { // 100KB
		return Error(&c, "代码过长，最大限制为100KB", Response{})
	}

	// 超时上下文
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, pyExec, "-c", req.Code)
	// 统一输出缓冲
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// 环境变量（禁用缓冲防止卡住）
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if runErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Error(&c, "执行超时", Response{"durationMs": duration.Milliseconds()})
		}
		// 解析进程退出码
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}

	result := map[string]interface{}{
		"stdout":     stdout.String(),
		"stderr":     stderr.String(),
		"exitCode":   exitCode,
		"durationMs": duration.Milliseconds(),
	}

	// 成功或失败统一返回结构
	return Success(&c, Response{
		"result": result,
	})
}

// pythonGetConfig 获取扩展配置
func pythonGetConfig(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	name := c.QueryParam("name")
	if name == "" {
		return Error(&c, "扩展名不能为空", Response{})
	}

	ext := myDice.GetPythonExtension(name)
	if ext == nil {
		return Error(&c, "扩展不存在", Response{})
	}

	// 这里应该返回扩展的配置信息，如果没有，我也不知道返回什么了
	config := map[string]interface{}{
		"name":    ext.Name,
		"version": ext.Version,
		"author":  ext.Author,
		"brief":   ext.Brief,
	}

	return Success(&c, Response{
		"config": config,
	})
}

// pythonSetConfig 扩展配置
func pythonSetConfig(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var req struct {
		Name   string                 `json:"name"`
		Config map[string]interface{} `json:"config"`
	}

	if err := c.Bind(&req); err != nil {
		return Error(&c, "参数错误", Response{})
	}

	if req.Name == "" {
		return Error(&c, "扩展名不能为空", Response{})
	}

	ext := myDice.GetPythonExtension(req.Name)
	if ext == nil {
		return Error(&c, "扩展不存在", Response{})
	}

	// 保存配置到扩展的Storage
	err := myDice.SavePythonExtensionConfig(req.Name, req.Config)
	if err != nil {
		return Error(&c, "保存配置失败: "+err.Error(), Response{})
	}

	return Success(&c, Response{
		"message": "配置保存成功",
		"config":  req.Config,
	})
}

// pythonCallAPI 调用扩展的API
func pythonCallAPI(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	var req struct {
		Name      string                 `json:"name"`
		Method    string                 `json:"method"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := c.Bind(&req); err != nil {
		return Error(&c, "参数错误", Response{})
	}

	if req.Name == "" || req.Method == "" {
		return Error(&c, "扩展名和方法名不能为空", Response{})
	}

	ext := myDice.GetPythonExtension(req.Name)
	if ext == nil {
		return Error(&c, "扩展不存在", Response{})
	}

	// 调用Python方法
	result, err := myDice.CallPythonMethod(req.Name, req.Method, req.Arguments)
	if err != nil {
		return Error(&c, "调用方法失败: "+err.Error(), Response{})
	}

	return Success(&c, Response{
		"result": result,
	})
}