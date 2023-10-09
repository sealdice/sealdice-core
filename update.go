package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sealdice-core/dice"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var binPrefix = "https://sealdice.coding.net/p/sealdice/d/sealdice-binaries/git/raw/master"

func downloadUpdate(dm *dice.DiceManager, log *zap.SugaredLogger) (string, error) {
	var packFn string
	if dm.AppVersionOnline != nil {
		ver := dm.AppVersionOnline
		if ver.VersionLatestCode != dm.AppVersionCode {
			platform := runtime.GOOS
			arch := runtime.GOARCH
			version := ver.VersionLatest
			var ext string

			// 如果线上版本小于当前版本，那么拒绝更新
			if ver.VersionLatestCode < dm.AppVersionCode {
				return "", errors.New("获取到的线上版本旧于当前版本，停止更新")
			}

			switch platform {
			case "windows":
				ext = "zip"
			default:
				// 其他各种平台似乎都是 .tar.gz
				ext = "tar.gz"
			}

			if arch == "386" {
				arch = "i386"
			}

			fn := fmt.Sprintf("sealdice-core_%s_%s_%s.%s", version, platform, arch, ext)
			fileUrl := binPrefix + "/" + fn

			if ver.NewVersionUrlPrefix != "" {
				fileUrl = ver.NewVersionUrlPrefix + "/" + fn
			}

			log.Infof("准备下载更新: %s", fn)
			err := os.RemoveAll("./update")
			if err != nil {
				return "", errors.New("更新: 删除缓存目录(update)失败")
			}

			_ = os.MkdirAll("./update", 0755)
			fn2 := "./update/update." + ext
			err = DownloadFile(fn2, fileUrl)
			if err != nil {
				return "", errors.New("更新: 下载更新文件失败")
			}
			log.Infof("更新下载完成，保存于: %s", fn2)
			packFn = fn2
		}
	}
	return packFn, nil
}

func RebootRequestListen(dm *dice.DiceManager) {
	<-dm.RebootRequestChan
	doReboot(dm)
}

func UpdateCheckRequestListen(dm *dice.DiceManager) {
	for {
		<-dm.UpdateCheckRequestChan
		CheckVersion(dm)
	}
}

func UpdateRequestListen(dm *dice.DiceManager) {
	curDice := <-dm.UpdateRequestChan
	log := curDice.Logger
	updatePackFn, err := downloadUpdate(dm, log)
	if err == nil {
		dm.UpdateDownloadedChan <- ""
		time.Sleep(2 * time.Second)
		log.Info("进行升级准备工作")

		dm.UpdateSealdiceByFile(updatePackFn, log)
		// 旧版本行为: 将新升级包里的主程序复制到当前目录，命名为 auto_update.exe 或 auto_update
		// 然后重启主程序

		log.Info("开始自重启，重启后将拉起升级程序auto_update.exe")
		doReboot(dm)
	} else {
		dm.UpdateDownloadedChan <- err.Error()
	}
}

func doReboot(dm *dice.DiceManager) {
	executablePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return
	}

	binary, err := exec.LookPath(executablePath)
	if err != nil {
		logger.Errorf("Restart Error: %s", err)
		return
	}
	platform := runtime.GOOS
	if platform == "windows" {
		cleanUpCreate(dm)()

		//name, _ := filepath.Abs(binary)
		//err = exec.Command(`cmd`, `/C`, "start", name, "--delay=15").Start()
		cmd := executeWin(binary, "--delay=15")
		err := cmd.Start()
		if err != nil {
			logger.Errorf("Restart error: %s %v", binary, err)
		}
	} else {
		// 手动cleanup
		cleanUpCreate(dm)()
		// os.Args[1:]...
		execErr := syscall.Exec(binary, []string{os.Args[0], "--delay=25"}, os.Environ())
		if execErr != nil {
			logger.Errorf("Restart error: %s %v", binary, execErr)
		}
	}
	os.Exit(0)
}

func checkVersionBase(backendUrl string, dm *dice.DiceManager) *dice.VersionInfo {
	resp, err := http.Get(backendUrl + "/dice/api/version?versionCode=" + strconv.FormatInt(dm.AppVersionCode, 10) + "&v=" + strconv.FormatInt(rand.Int63(), 10))
	if err != nil {
		//logger.Errorf("获取新版本失败: %s", err.Error())
		return nil
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var ver dice.VersionInfo
	err = json.NewDecoder(resp.Body).Decode(&ver)
	if err != nil {
		return nil
	}

	dm.AppVersionOnline = &ver
	//downloadUpdate(dm)
	return &ver
}

func CheckVersion(dm *dice.DiceManager) *dice.VersionInfo {
	if runtime.GOOS == "android" {
		return nil
	}
	// 逐个尝试所有后端地址
	for _, i := range dice.BackendUrls {
		ret := checkVersionBase(i, dm)
		if ret != nil {
			return ret
		}
	}
	return nil
}

func DownloadFile(filepath string, url string) error {
	// Get the data
	//resp, err := http.Get(url)
	client := new(http.Client)
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Add("Accept-Encoding", "gzip")
	resp, err := client.Do(request)

	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	if resp.StatusCode == http.StatusOK {
		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		return err
	}

	return errors.New("http status:" + resp.Status)
}

func unzipSource(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer func(reader *zip.ReadCloser) {
		_ = reader.Close()
	}(reader)

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File, destination string) error {
	// 注: 用这个zip包的原因是解压utf-8不乱码
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer func(destinationFile *os.File) {
		_ = destinationFile.Close()
	}(destinationFile)

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer func(zippedFile io.ReadCloser) {
		_ = zippedFile.Close()
	}(zippedFile)

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

func ExtractTarGz(fn, dest string) error {
	gzipStream, err := os.Open(fn)
	if err != nil {
		fmt.Println("error", err.Error())
		return err
	}
	defer func(gzipStream *os.File) {
		_ = gzipStream.Close()
	}(gzipStream)

	log := logger
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Error("ExtractTarGz: NewReader failed")
		return err
	}
	defer func(uncompressedStream *gzip.Reader) {
		_ = uncompressedStream.Close()
	}(uncompressedStream)

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Errorf("ExtractTarGz: Next() failed: %s", err.Error())
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filepath.Join(dest, header.Name), 0755); err != nil {
				log.Errorf("ExtractTarGz: Mkdir() failed: %s", err.Error())
			}
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(filepath.Join(dest, header.Name)), 0755) // 进行一个目录的创
			outFile, err := os.Create(filepath.Join(dest, header.Name))
			if err != nil {
				log.Errorf("ExtractTarGz: Create() failed: %s", err.Error())
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				log.Errorf("ExtractTarGz: Copy() failed: %s", err.Error())
				return err
			}
			_ = outFile.Close()

		default:
			log.Error(
				"ExtractTarGz: uknown type: %s in %s",
				header.Typeflag,
				header.Name)
			return err
		}
	}
	return nil
}
