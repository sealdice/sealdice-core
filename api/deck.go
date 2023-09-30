package api

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sealdice-core/dice"
	"strings"

	"github.com/labstack/echo/v4"
)

func deckList(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	return c.JSON(http.StatusOK, myDice.DeckList)
}

func deckReload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	dice.DeckReload(myDice)
	return c.JSON(http.StatusOK, true)
}

func deckUpload(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	//-----------
	// Read file
	//-----------

	// Source
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func(src multipart.File) {
		_ = src.Close()
	}(src)

	// Destination
	//fmt.Println("????", filepath.Join("./data/decks", file.Filename))
	file.Filename = strings.ReplaceAll(file.Filename, "/", "_")
	file.Filename = strings.ReplaceAll(file.Filename, "\\", "_")
	dst, err := os.Create(filepath.Join("./data/decks", file.Filename))
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		_ = dst.Close()
	}(dst)

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, nil)
}

func deckEnable(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}

	v := struct {
		Index  int  `json:"index"`
		Enable bool `json:"enable"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		if v.Index >= 0 && v.Index < len(myDice.DeckList) {
			myDice.DeckList[v.Index].Enable = v.Enable
			myDice.MarkModified()
		}
	}

	return c.JSON(http.StatusOK, myDice.BanList)
}

func deckDelete(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return c.JSON(200, map[string]interface{}{
			"testMode": true,
		})
	}

	v := struct {
		Index int `json:"index"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		if v.Index >= 0 && v.Index < len(myDice.DeckList) {
			dice.DeckDelete(myDice, myDice.DeckList[v.Index])
			myDice.MarkModified()
		}
	}

	return c.JSON(http.StatusOK, nil)
}

func deckCheckUpdate(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	v := struct {
		Index int `json:"index"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		if v.Index >= 0 && v.Index < len(myDice.DeckList) {
			deck := myDice.DeckList[v.Index]
			oldDeck, newDeck, tempFileName, err := myDice.DeckCheckUpdate(deck)
			if err != nil {
				return Error(&c, err.Error(), Response{})
			}
			return Success(&c, Response{
				"old":          oldDeck,
				"new":          newDeck,
				"format":       deck.FileFormat,
				"tempFileName": tempFileName,
			})
		}
	}
	return Success(&c, Response{})
}

func deckUpdate(c echo.Context) error {
	if !doAuth(c) {
		return c.JSON(http.StatusForbidden, nil)
	}
	if dm.JustForTest {
		return Error(&c, "展示模式不支持该操作", Response{"testMode": true})
	}
	v := struct {
		Index        int    `json:"index"`
		TempFileName string `json:"tempFileName"`
	}{}
	err := c.Bind(&v)

	if err == nil {
		if v.Index >= 0 && v.Index < len(myDice.DeckList) {
			err := myDice.DeckUpdate(myDice.DeckList[v.Index], v.TempFileName)
			if err != nil {
				return Error(&c, err.Error(), Response{})
			}
			myDice.MarkModified()
		}
	}
	return Success(&c, Response{})
}
