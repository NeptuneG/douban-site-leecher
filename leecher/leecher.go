package leecher

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"errors"
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

// Handler 就是用来搞事情的函数了
func Handler(w http.ResponseWriter, r *http.Request) {
	log.Output(0, "RequestURI: "+r.RequestURI)
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
	tmpDirName := getTimeStamp()
	tmpFileName := tmpDirName + ".zip"
	err = os.Mkdir(tmpDirName, os.ModeDir)
	if err != nil {
		log.Output(0, err.Error())
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
			err = json.Unmarshal([]byte(strJSON), &records)
			if err != nil {
				log.Output(0, err.Error())
				return
			}
			err = download(tmpDirName, records)
			if err != nil {
				log.Output(0, err.Error())
				return
			}
		}
	}

	err = zipit(tmpDirName, tmpFileName)
	if err != nil {
		log.Output(0, err.Error())
		return
	}

	file, err := os.Open(tmpFileName)
	if err != nil {
		log.Output(0, err.Error())
		return
	}
	_, err = io.Copy(w, file)
	if err != nil {
		log.Output(0, err.Error())
		return
	}
	err = file.Close()
	if err != nil {
		log.Output(0, err.Error())
		return
	}

	err = removeTmpFiles(tmpFileName, tmpDirName)
	if err != nil {
		log.Output(0, err.Error())
		return
	}
}

func getURL(arg string) (string, error) {
	tokens := strings.Split(arg, "/")
	if len(tokens) != 2 {
		return "", errors.New("无法识别的URI")
	}

	return defaultURL + tokens[1], nil
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
		log.Output(0, "从"+record.RawURL+"开始下载："+record.Name)
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
