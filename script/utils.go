package script

import (
	"bytes"
	"math/rand"
	"net"
	"time"
)

func GetIpAddress() string {
	addrs, _ := net.InterfaceAddrs()
	for _, value := range addrs {
		if ipnet, ok := value.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

const char = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandChar(size int) string {
	source := rand.NewSource(time.Now().UnixNano()) // 产生随机种子
	var s bytes.Buffer
	for i := 0; i < size; i++ {
		s.WriteByte(char[source.Int63()%int64(len(char))])
	}
	return s.String()
}

type Timestamp int64

const time_layout string = "2006-01-02 15:04:05"

// 日期字符串转为时间戳 工具函数
func getTimeStamp(str_date string) Timestamp {
	if len(str_date) == 10 {
		str_date = str_date + " 00:00:00"
	}
	// 获取时区
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return 0
	}
	// 转换为时间戳
	the_time, err := time.ParseInLocation(time_layout, str_date, loc)
	if err != nil {
		return 0
	}
	// 返回时间戳
	return Timestamp(the_time.Unix())
}

//获取1到2个日期字符串中更大的日期
func getMaxDate(str_date1 string, str_date2 string) string {
	var max_date string
	if str_date1 != "" && str_date2 == "" {
		// 只定义了第一个账户，取第一个账户的日期为准
		max_date = str_date1
	} else if str_date1 == "" && str_date2 != "" {
		// 只定义了第二个账户，取第二个账户的日期为准
		max_date = str_date2
	} else if str_date1 != "" && str_date2 != "" {
		// 重复定义的账户，取最晚的时间为准
		t1 := getTimeStamp(str_date1)
		t2 := getTimeStamp(str_date2)
		if t1 > t2 {
			max_date = str_date1
		} else {
			max_date = str_date2
		}
	} else if str_date1 == "" && str_date2 == "" {
		// 没有定义账户，取当前日期为准
		max_date = time.Now().Format(time_layout)
	}
	return max_date
}

// 获取1-2个日期字符串中最小的日期值
// 如果双参数均为空，则返回账簿开始记账日期
func getMinDate(str_date1 string, str_date2 string) string {
	//time_layout := "2006-01-02 15:04:05"
	var min_date string
	if str_date1 != "" && str_date2 == "" {
		// 只定义了第一个账户，取第一个账户的日期为准
		min_date = str_date1
	} else if str_date1 == "" && str_date2 != "" {
		// 只定义了第二个账户，取第二个账户的日期为准
		min_date = str_date2
	} else if str_date1 != "" && str_date2 != "" {
		// 重复定义的账户，取最早的时间
		t1 := getTimeStamp(str_date1)
		t2 := getTimeStamp(str_date2)
		if t1 < t2 {
			min_date = str_date1
		} else {
			min_date = str_date2
		}
	} else if str_date1 == "" && str_date2 == "" {
		// 没有定义账户，取固定日期"1970-01-01"
		min_date = "1970-01-01"
	}
	return min_date
}
