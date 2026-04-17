package storylog

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"sealdice-core/model"
	"sealdice-core/utils"
)

// FormatLogTxtLine formats one exported log entry using the shared TXT layout.
func FormatLogTxtLine(nickname, imUserID string, unixTime int64, message string) string {
	timeTxt := time.Unix(unixTime, 0).Format("2006-01-02 15:04:05")
	return fmt.Sprintf("%s(%s) %s\n%s\n\n", nickname, imUserID, timeTxt, message)
}

// WriteLogTXT writes exported log lines using the shared TXT layout.
func WriteLogTXT(w io.Writer, lines []*model.LogOneItem) error {
	for _, line := range lines {
		if _, err := io.WriteString(w, FormatLogTxtLine(line.Nickname, line.IMUserID, line.Time, line.Message)); err != nil {
			return err
		}
	}
	return nil
}

func writeLogTXTParquet(w io.Writer, lines []model.LogOneItemParquet) error {
	for _, line := range lines {
		if _, err := io.WriteString(w, FormatLogTxtLine(line.Nickname, line.IMUserID, line.Time, line.Message)); err != nil {
			return err
		}
	}
	return nil
}

func writeStoryV1JSON(w io.Writer, lines []*model.LogOneItem) error {
	if _, err := io.WriteString(w, `{"items":[`); err != nil {
		return err
	}
	for i, line := range lines {
		if i > 0 {
			if _, err := io.WriteString(w, ","); err != nil {
				return err
			}
		}
		itemJSON, err := json.Marshal(line)
		if err != nil {
			return err
		}
		if _, err := w.Write(itemJSON); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, `],"version":%d}`, StoryVersionV1)
	return err
}

func writeReadmeToZip(writer *zip.Writer) error {
	readmeWriter, err := writer.Create(ExportReadmeFilename)
	if err != nil {
		return err
	}
	_, err = io.WriteString(readmeWriter, ExportReadmeContent)
	return err
}

func copyFileToZip(writer *zip.Writer, zipName string, sourcePath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	targetWriter, err := writer.Create(zipName)
	if err != nil {
		return err
	}
	_, err = io.Copy(targetWriter, sourceFile)
	return err
}

func exportTempFilePattern(prefix string, fallback string, ext string) string {
	return utils.TempFilePattern(prefix, fallback, ext, 80)
}

func exportZipFilename(groupID string, logName string, now time.Time) string {
	groupPart := utils.FilenameSafeReadable(groupID, "group", 32)
	logPart := utils.FilenameSafeReadable(logName, "log", 64)
	return fmt.Sprintf("%s_%s.%s.zip", groupPart, logPart, now.Format("060102150405"))
}
