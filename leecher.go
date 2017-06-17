package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"
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
	defaultURL       = "http://site.douban.com/"
	jsonPrefix       = "[{"
	jsonSuffix       = "}]"
	targetLinePrefix = "song_records"
	bufferSize       = 1024 * 8
)

// 命令行参数: 完整豆瓣小站URL或URL后缀
func main() {
	if len(os.Args) != 2 {
		fmt.Println("请输入豆瓣小站网址或后缀，如：https://site.douban.com/chinesefootball 或 chinesefootball")
		os.Exit(1)
	}

	url := getURL(os.Args[1])

	response, err := http.Get(url)
	checkError(err)
	defer response.Body.Close()

	reader := bufio.NewReaderSize(response.Body, bufferSize)
	for {
		buf := make([]byte, 0)
		buf, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
		}

		line := string(buf)

		if isSongRecords(line) {
			records := make([]Record, 0)
			strJSON := getJSONstring(line)
			err1 := json.Unmarshal([]byte(strJSON), &records)
			checkError(err1)
			err2 := download(records)
			checkError(err2)
		}
	}
}

func getURL(arg string) string {
	if strings.HasPrefix(arg, defaultURL) {
		return arg
	}
	return defaultURL + arg
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func isSongRecords(line string) bool {
	strTrimed := strings.Trim(line, " ")
	return strings.HasPrefix(strTrimed, targetLinePrefix)
}

func getJSONstring(line string) string {
	return line[strings.Index(line, jsonPrefix) : strings.LastIndex(line, jsonSuffix)+len(jsonSuffix)]
}

func download(records []Record) error {
	for _, record := range records {
		filename, err := getFileName(record)
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

func getFileName(record Record) (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	dirname := user.HomeDir + "/Downloads/"
	filename := record.Name + record.RawURL[strings.LastIndex(record.RawURL, "."):]
	return dirname + filename, nil
}
