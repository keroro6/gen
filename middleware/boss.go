package middleware

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"net"
	"time"
	"xgit.xiaoniangao.cn/recsys/traffic_dist_recall/conf"
)

var LocalIp string

func init() {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Errorf("Oops, InterfaceAddrs err:\n%v", err)
	}
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				LocalIp = ipNet.IP.String()
				return
			}
		}
	}
}

/*
{
    trace: {
        from:  // trace来源，如某个业务的某个模块
        step:  // trace深度，请求到第几层了
        id:    // trace id，每次请求都有的一个唯一的id，比如可以采用每次生成不重复的UUID
    },
    addr: {
        rmt:  // 对端的ip和端口，比如10.2.3.4:8000
        loc:  // 本地的ip和端口，比如10.2.3.5:8020
    },
    time: // 毫秒 201809221830100 「global」
    params:{} // 请求参数
    elapse: // 耗时，毫秒
    result: { // 返回结果
        ret: 1,
        data: {}
    },
    level: //INFO、ERROR、WARNING、DEBUG 「global」
    path: “filename.go:300:/favor/add", // 打点位置  「global」
    ext: {} // 自定义字段。不超过5个
}
*/

type addr struct {
	RemoteIP string
	LocalIP  string
}

func (a *addr) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("rmt", a.RemoteIP)
	enc.AddString("loc", a.LocalIP)
	return nil
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func Boss() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		URLPath := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if URLPath == "/health" {
			c.Next()
			return
		}
		if raw != "" {
			URLPath = URLPath + "?" + raw
		}
		clientIP := c.ClientIP()

		reqBuf, _ := ioutil.ReadAll(c.Request.Body)
		reqBuf1 := ioutil.NopCloser(bytes.NewBuffer(reqBuf))
		reqBuf2 := ioutil.NopCloser(bytes.NewBuffer(reqBuf))
		c.Request.Body = reqBuf2
		reqBody, _ := ioutil.ReadAll(reqBuf1)

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		_addr := &addr{
			RemoteIP: clientIP,
			LocalIP:  LocalIp,
		}

		m := map[string]interface{}{
			"addr":     _addr,
			"elapse":   latency.Nanoseconds() / int64(time.Millisecond),
			"url":      URLPath,
			"status":   c.Writer.Status(),
			"body":     string(reqBody),
			"response": blw.body.String(),
		}
		conf.TraceLogger.InfoW("trace", m)
	}
}
