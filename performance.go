package main

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

type performance struct {
	connectionNum int
	reqNum        int64
	requestBytes  []byte
	target        string
	schema        string
	proxy         string
	timeout       int
	startTime     time.Time
	endTime       time.Time
	wg            sync.WaitGroup
	finishNum     int64
	reqConnList   []*ReqConn
}

func (pf *performance) Run() {
	fmt.Printf("Running %d test @ %s by %d connections\n", pf.reqNum, pf.target, pf.connectionNum)
	var err error
	pf.startTime = time.Now()
	pf.wg.Add(pf.connectionNum)
	for i := 0; i < pf.connectionNum; i++ {
		rc := &ReqConn{
			Count:      pf.reqNum,
			NowNum:     &pf.finishNum,
			timeout:    pf.timeout,
			writeBytes: pf.requestBytes,
			reqTimes:   make([]int, 0),
			remoteAddr: pf.target,
			schema:     pf.schema,
			buf:        make([]byte, 65535),
			proxy:      pf.proxy,
		}
		go func() {
			if err = rc.Start(); err != nil {
				fmt.Println(err.Error())
				os.Exit(0)
			}
			pf.wg.Done()
		}()
		pf.reqConnList = append(pf.reqConnList, rc)
	}
	pf.wg.Wait()
	pf.endTime = time.Now()
	return
}

func (pf *performance) Print() {
	readAll := 0
	writeAll := 0
	allTimes := make([]int, 0)
	allError := 0
	for _, v := range pf.reqConnList {
		readAll += v.readLen
		writeAll += v.writeLen
		allTimes = append(allTimes, v.reqTimes...)
		allError += v.ErrorTimes
	}
	second := pf.endTime.Sub(pf.startTime).Seconds()
	fmt.Printf("%d requests in %.2fs, %s read, %s write\n", pf.reqNum, second, formatFlow(float64(readAll)), formatFlow(float64(writeAll)))
	fmt.Printf("Requests/sec: %.2f\n", float64(pf.reqNum)/second)
	fmt.Printf("Transfer/sec: %s\n", formatFlow(float64(readAll+writeAll)/second))
	fmt.Printf("Error       : %d\n", allError)
	sort.Ints(allTimes)
	rates := []int{50, 65, 75, 80, 90, 95, 98, 99, 100}
	fmt.Println("Percentage of the requests served within a certain time (ms)")
	for _, v := range rates {
		fmt.Printf("   %3d%%\t\t\t\t%d\n", v, allTimes[len(allTimes)*v/100-1])
	}
}

func formatFlow(size float64) string {
	var rt float64
	var suffix string
	const (
		Byte  = 1
		KByte = Byte * 1024
		MByte = KByte * 1024
		GByte = MByte * 1024
	)

	if size > GByte {
		rt = size / GByte
		suffix = "GB"
	} else if size > MByte {
		rt = size / MByte
		suffix = "MB"
	} else if size > KByte {
		rt = size / KByte
		suffix = "KB"
	} else {
		rt = size
		suffix = "bytes"
	}

	srt := fmt.Sprintf("%.2f%v", rt, suffix)

	return srt
}
