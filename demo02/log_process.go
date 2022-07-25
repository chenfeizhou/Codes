package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Reader interface {
	Read(rc chan []byte)
}
type Writer interface {
	Write(wc chan *Message)
}

type LogProcess struct {
	rc    chan []byte
	wc    chan *Message
	path  string // 读取文件的路径
	read  Reader
	write Writer
}

type ReadFromFile struct {
	path string // 读取文件的路径
}

type WriterToInfluxDB struct {
	influxDBDsn string // influx data source
}

type Message struct {
	TimeLocal                    time.Time
	BytesSent                    int
	Path, Method, Scheme, Status string
	UpStreamTime, RequestTime    float64
}

func (r *ReadFromFile) Read(rc chan []byte) {

	// 读取模块
	// 打开文件
	f, err := os.Open(r.path)
	if err != nil {
		panic(err.Error())
	}

	// 从文件末尾开始促行读取
	f.Seek(0, 2)
	rd := bufio.NewReader(f)

	for {

		line, err := rd.ReadBytes('\n')

		if err == io.EOF {
			time.Sleep(500 * time.Microsecond)
			continue
		} else if err != nil {
			panic(fmt.Sprintf("ReadBytes error: %s", err.Error()))
		}

		rc <- line[:len(line)-1]
	}
}

func (w *WriterToInfluxDB) Write(wc chan *Message) {
	// 写入模块
	for v := range wc {
		fmt.Println(v)
	}
}

func (l *LogProcess) Process() {

	/**
	log example:
	172.0.0.12 - - [04/Mar/2018:13:49:52 +0000] http "GET /foo?query=t HTTP/1.0" 200 2133 "-" "KeepAliveClient" "-" 1.005 1.854
	*/

	r := regexp.MustCompile(`([\d\.])\s+(^\[]+)\s+(^\[]+)\s+\[([^\]+)\]\s+([a-z]+)\s+"([^"]+)\"\s+
	(\d{3})\s+(\d+)\s+\"([^"]+)\"\s+\"(.*?)\"\s+\"([\d\.-]+)\"\s+([\d\.-]+)\s+([\d\.-]+)`)

	loc, _ := time.LoadLocation("Asia/Shanghai")

	// 解析模块
	for v := range l.rc {

		maths := r.FindStringSubmatch(string(v))

		if len(maths) != 14 {
			log.Println("FindStringSubmatch fail:", string(v))
			continue
		}

		message := &Message{}

		t, err := time.ParseInLocation("02/Jan/2006:15:04:05 +0000", maths[4], loc)

		if err != nil {
			log.Println("Parse Time fail:", err.Error(), maths[4])
		}

		message.TimeLocal = t

		byteSent, _ := strconv.Atoi(maths[8])
		message.BytesSent = byteSent

		reqSlice := strings.Split(maths[6], " ")

		if len(reqSlice) != 3 {
			log.Println("strings.split fail :", err.Error(), maths[6])
			continue
		}

		message.Method = reqSlice[0]

		u, err := url.Parse(reqSlice[1])

		if err != nil {
			log.Println("parse url fail:", err.Error())
			continue
		}

		message.Path = u.Path
		message.Scheme = maths[5]
		message.Status = maths[7]

		upstreamTime, _ := strconv.ParseFloat(maths[12], 64)
		requestTime, _ := strconv.ParseFloat(maths[13], 64)

		message.UpStreamTime = upstreamTime
		message.RequestTime = requestTime

		l.wc <- message
	}
}

func main() {
	r := &ReadFromFile{
		path: "./access.log",
	}

	w := &WriterToInfluxDB{
		influxDBDsn: "username&password",
	}

	lp := &LogProcess{
		rc:    make(chan []byte),
		wc:    make(chan *Message),
		read:  r,
		write: w,
	}

	// 并发执行
	go lp.read.Read(lp.rc)
	go lp.Process()
	go lp.write.Write(lp.wc)

	time.Sleep(30 * time.Second)
}
