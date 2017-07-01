package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	timeformat       = "20060102150405"
)

// Handler 负责处理请求
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, err := template.ParseFiles("server.gtpl")
		if isFailed(err) {
			return
		}
		err = t.Execute(w, nil)
		if isFailed(err) {
			return
		}
	} else {
		url, err := getURL(r.FormValue("URI"))
		if isFailed(err) {
			return
		}

		response, err := http.Get(url)
		if isFailed(err) {
			return
		}
		defer response.Body.Close()

		// 创建临时文件夹用存放下载文件以供打包
		tmpDirName := getTimeStamp()
		tmpFileName := tmpDirName + ".zip"
		err = os.Mkdir(tmpDirName, os.ModeDir)
		if isFailed(err) {
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
			}

			line := string(buf)

			if isSongRecords(line) {
				records := make([]Record, 0)
				strJSON := getJSONstring(line)
				err = json.Unmarshal([]byte(strJSON), &records)
				if isFailed(err) {
					return
				}
				err = download(tmpDirName, records)
				if isFailed(err) {
					return
				}
			}
		}

		err = zipit(tmpDirName, tmpFileName)
		if isFailed(err) {
			return
		}

		file, err := os.Open(tmpFileName)
		if isFailed(err) {
			return
		}
		_, err = io.Copy(w, file)
		if isFailed(err) {
			return
		}
		err = file.Close()
		if isFailed(err) {
			return
		}

		err = removeTmpFiles(tmpFileName, tmpDirName)
		if isFailed(err) {
			return
		}
	}
}

func getURL(arg string) (string, error) {
	return defaultURL + arg, nil
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
		log.Println("从<" + record.RawURL + ">开始下载: " + record.Name)
		_, err1 := io.Copy(file, response.Body)
		if err1 != nil {
			return err
		}
		log.Println("完成下载：" + record.Name)
	}

	return nil
}

func getFileName(dirname string, record Record) (string, error) {
	filename := dirname + "/" + record.Name + record.RawURL[strings.LastIndex(record.RawURL, "."):]
	return filename, nil
}

func getTimeStamp() string {
	return time.Now().Format(timeformat)
}

func zipit(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func removeTmpFiles(tmpFileName, tmpDirName string) error {
	err := os.Remove(tmpFileName)
	if err != nil {
		return err
	}
	err = os.RemoveAll(tmpDirName)
	if err != nil {
		return err
	}
	return nil
}

func isFailed(err error) bool {
	if err != nil {
		log.Println(err.Error())
		return true
	}
	return false
}
