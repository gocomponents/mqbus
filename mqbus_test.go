package mqbus

import (
	"testing"
	"time"
)

type log struct {
	Message    interface{} `json:"message"`
	Msec       int         `json:"msec"`
	Userip     string      `json:"userip"`
	Exception  interface{} `json:"exception"`
	Timespan   string      `json:"timespan"`
	Level      string      `json:"level"`
	Logger     string      `json:"logger"`
	CreateTime string      `json:"createTime"`
	LogType    string      `json:"logType"`
	TraceId    string      `json:"traceId"`
}

func (*log) ConnectionString() string {
	return "host=localhost:5672;username=guest;password=123456"
}
func (*log) Exchange() string {
	return "Core.Logs.EventLog:Core"
}
func (*log) MessageType() string {
	return "Core.Logs.EventLog:Core"
}
func (*log) RoutingKey() string {
	return "#"
}

func TestMqBus_Post(t *testing.T) {
	var log = &log{
		Message:    "hello",
		Exception:  "exception",
		Level:      "DEBUG",
		LogType:    "MonitorApi",
		Logger:     "MonitorApi",
		CreateTime: time.Now().Format("2006-01-02 15:04:05") + "+08:00",
	}

	err := Default.Post(log)

	if err != nil {
		t.Fatalf(err.Error())
	}
}


func BenchmarkMqBus_Post(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var log = &log{
			Message:    "hello",
			Exception:  "exception",
			Level:      "DEBUG",
			LogType:    "MonitorApi",
			Logger:     "MonitorApi",
			CreateTime: time.Now().Format("2006-01-02 15:04:05") + "+08:00",
		}

		err := Default.Post(log)

		if err != nil {
			b.Fatalf(err.Error())
		}
	}
}

func subIndexStrT(str string, indexStr string) string {

	if str == "" || indexStr == "" {
		return ""
	}
	rs := []rune(str)

	result := string(rs[len(indexStr):len(str)])

	return result
}

func TestSubIndexStr(t *testing.T) {
	str := "host=localhost:5672"
	host := subIndexStrT(str, "host=")
	if host != "rabbitmq.dev.svc.cluster.local:5672" {
		t.Error("host error actualValue : ", host, " hopeValue : localhost:5672")
	}
	str2 := "password=123456"
	pwd := subIndexStrT(str2, "password=")
	if pwd != "123456" {
		t.Error("host error actualValue : ", pwd, " hopeValue : 123456")
	}
}
