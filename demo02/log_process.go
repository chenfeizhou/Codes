package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
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

// 系统状态监控
type SystemInfo struct {
	HandleLine   int     `json:"handleLine"`   // 总处理日志行数
	Tps          float64 `json:"tps"`          // 系统吞出量
	ReadChanLen  int     `json:"readChanLen"`  // read channel 长度
	WriteChanlen int     `json:"writeChanLen"` // write channel 长度
	RunTime      string  `json:"runTime"`      // 运行总时间
	ErrNum       int     `json:"errNum"`       // 错误数
}

const (
	TypeHandeLine = 0
	TypeErrNum    = 1
)

var TypeMonitorChan = make(chan int, 200)

type Monitor struct {
	startTime time.Time
	data      SystemInfo
	tpsSlic   []int
}

func (m *Monitor) start(lp *LogProcess) {

	go func() {
		for n := range TypeMonitorChan {
			switch n {
			case TypeErrNum:
				m.data.ErrNum += 1
			case TypeHandeLine:
				m.data.HandleLine += 1
			}
		}
	}()

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for {
			<-ticker.C
			m.tpsSlic = append(m.tpsSlic, m.data.HandleLine)
			if len(m.tpsSlic) > 2 {
				m.tpsSlic = ,.tpsm.tpsSlic[1:]
			}
		}

	}()

	http.HandleFunc("/monitor", func(writer http.ResponseWriter, r *http.Request) {

		m.data.RunTime = time.Now().Sub(m.startTime).String()
		m.data.ReadChanLen = len(lp.rc)
		m.data.WriteChanlen = len(lp.wc)

		if len(m.tpsSli) > 2 {
			m.data.Tps = float64(m.tpsSli[1] - m.tpsSli[0]) / 5
		}

		ret, _ := json.MarshalIndent(m.data, "", "\t")

		io.WriteString(writer, string(ret))
	})

	http.ListenAndServe(":9193", nil)
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

		TypeMonitorChan <- TypeHandeLine

		rc <- line[:len(line)-1]
	}
}

func (w *WriterToInfluxDB) Write(wc chan *Message) {

	infSli := strings.Split(w.influxDBDsn, "@")

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     infSli[0],
		Username: infSli[1],
		Password: infSli[2],
	})

	if err != nil {
		fmt.Println("Error creating InfluxDB Client: ", err.Error())
	}

	defer c.Close()

	for v := range wc {

		bp, err := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  infSli[3],
			Precision: infSli[4],
		})

		if err != nil {
			log.Fatal(err)
		}

		// Create a point and add to batch
		tags := map[string]string{"Path": v.Path, "Method": v.Method, "Scheme": v.Scheme, "Status": v.Status}
		// Fields: UpstreamTime, RequestTime, BytesSent
		fields := map[string]interface{}{
			"UpstreamTime": v.UpStreamTime,
			"RequestTime":  v.RequestTime,
			"BytesSent":    v.BytesSent,
		}

		pt, err := client.NewPoint("nginx_log", tags, fields, v.TimeLocal)
		if err != nil {
			log.Fatal(err)
		}

		bp.AddPoint(pt)

		// 写入db
		if err := c.Write(bp); err != nil {
			log.Fatal(err)
		}

		log.Println("write success!")
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

			TypeMonitorChan <- TypeErrNum

			log.Println("FindStringSubmatch fail:", string(v))
			continue
		}

		message := &Message{}

		t, err := time.ParseInLocation("02/Jan/2006:15:04:05 +0000", maths[4], loc)

		if err != nil {
			TypeMonitorChan <- TypeErrNum

			log.Println("Parse Time fail:", err.Error(), maths[4])
			continue
		}

		message.TimeLocal = t

		byteSent, _ := strconv.Atoi(maths[8])
		message.BytesSent = byteSent

		reqSlice := strings.Split(maths[6], " ")

		if len(reqSlice) != 3 {
			TypeMonitorChan <- TypeErrNum

			log.Println("strings.split fail :", err.Error(), maths[6])
			continue
		}

		message.Method = reqSlice[0]

		u, err := url.Parse(reqSlice[1])

		if err != nil {
			log.Println("parse url fail:", err.Error())
			TypeMonitorChan <- TypeErrNum
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

	// go run log_process.go -path ./access.log -influxDsn http://127.0.0.1:8086username&password&db@s
	
	var path, influxDsn string
	flag.StringVar(&path, "path", "./access.log", "read file path")
	flag.StringVar(&influxDsn, "influxDsn", "http://127.0.0.1:8086@username&password&db@s", "read influxdb path")
	flag.Parse()

	r := &ReadFromFile{
		path: path,
	}

	w := &WriterToInfluxDB{
		influxDBDsn: influxDsn,
	}

	lp := &LogProcess{
		rc:    make(chan []byte, 200),
		wc:    make(chan *Message, 200),
		read:  r,
		write: w,
	}

	// 并发执行
	go lp.read.Read(lp.rc)

	for i := 0; i<2; i++ {
		go lp.Process()
	}
	
	for i := 0; i<4; i++ {
		go lp.write.Write(lp.wc)
	}

	time.Sleep(30 * time.Second)

	m := &Monitor{
		startTime: time.Now(),
		data:      SystemInfo{},
	}
	m.start(lp)
}
