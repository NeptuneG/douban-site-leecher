package leecher

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Record 是对应song_records json字符串的结构体
type Record struct {
	Name   string
	URL    string
	Cover  string
	isDemo bool
	RawURL string
	ID     string
}

const (
	defaultURL       = "https://site.douban.com/"
	jsonPrefix       = "[{"
	jsonSuffix       = "}]"
	targetLinePrefix = "song_records"
	bufferSize       = 1024 * 8
)

// Handler 就是用来搞事情的函数了
func Handler(w http.ResponseWriter, r *http.Request) {
	url, err := getURL(r.RequestURI)
	if err != nil {
		log.Output(0, err.Error())
		return
	}

	response, err := http.Get(url)
	if err != nil {
		log.Output(0, err.Error())
		return
	}
	defer response.Body.Close()

	// 创建临时文件夹用存放下载文件以供打包
	tmpDirName := getTempDirName()
	err1 := os.Mkdir(tmpDirName, os.ModeDir)
	if err1 != nil {
		log.Output(0, err1.Error())
		return
	}

	reader := bufio.NewReaderSize(response.Body, bufferSize)
	for {
		buf := make([]byte, 0)
		buf, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Output(0, err.Error())
		}

		line := string(buf)

		if isSongRecords(line) {
			records := make([]Record, 0)
			strJSON := getJSONstring(line)
			err1 := json.Unmarshal([]byte(strJSON), &records)
			if err1 != nil {
				log.Output(0, err1.Error())
				return
			}
			err2 := download(tmpDirName, records)
			if err2 != nil {
				log.Output(0, err2.Error())
				return
			}
		}
	}
}

func getURL(arg string) (string, error) {
	tokens := strings.Split(arg, "/")
	if len(tokens) != 3 {
		return "", errors.New("无法识别的URI")
	}

	return defaultURL + tokens[2], nil
}

func isSongRecords(line string) bool {
	strTrimed := strings.Trim(line, " ")
	return strings.HasPrefix(strTrimed, targetLinePrefix)
}

func getJSONstring(line string) string {
	return line[strings.Index(line, jsonPrefix) : strings.LastIndex(line, jsonSuffix)+len(jsonSuffix)]
}

func download(targetDir string, records []Record) error {
	for _, record := range records {
		filename, err := getFileName(targetDir, record)
		if err != nil {
			return err
		}
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		response, err := http.Get(record.RawURL)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		log.Output(0, "开始下载："+record.Name)
		_, err1 := io.Copy(file, response.Body)
		if err1 != nil {
			return err
		}
		log.Output(0, "完成下载："+record.Name)
	}

	return nil
}

func getFileName(dirname string, record Record) (string, error) {
	filename := dirname + "/" + record.Name + record.RawURL[strings.LastIndex(record.RawURL, "."):]
	return filename, nil
}

func getTempDirName() string {
	return time.Now().Format("20060102150405")
}
