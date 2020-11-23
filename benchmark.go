// Copyright 2020 The benchmark. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// benchmark is used to manager connection and deal with the result
type benchmark struct {
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

// Start benchmark with the param has setting
func (pf *benchmark) Run() {
	fmt.Printf("Running %d test @ %s by %d connections\n", pf.reqNum, pf.target, pf.connectionNum)
	fmt.Printf("Request as following format:\n\n%s\n", string(pf.requestBytes))
	dialer, err := NewProxyConn(pf.proxy)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	pf.startTime = time.Now()
	pf.wg.Add(pf.connectionNum)
	successCount := int32(0)
	for i := 0; i < pf.connectionNum; i++ {
		rc := &ReqConn{
			Count:      pf.reqNum,
			NowNum:     &pf.finishNum,
			timeout:    pf.timeout,
			reqTimes:   make([]int, 0),
			remoteAddr: pf.target,
			schema:     pf.schema,
			dialer:     dialer,
			readWriter: NewHttpReadWriter(pf.requestBytes),
		}
		go func(idx int, rc *ReqConn) {
			if err := rc.Start(); err != nil {
				fmt.Println("Failed to start connection", idx, ":", err.Error())
				if !*ignoreErr {
					fmt.Printf("Try increasing the timeout using flag `-t`, or use `-ignore-err` to bypass.\n\n")
					os.Exit(0)
				}
			} else {
				atomic.AddInt32(&successCount, 1)
			}
			pf.wg.Done()
		}(i, rc)
		pf.reqConnList = append(pf.reqConnList, rc)
	}
	pf.wg.Wait()
	pf.endTime = time.Now()

	if successCount < int32(pf.connectionNum) {
		fmt.Printf("\nOnly %d successful connections, with %d failure. Try increasing the timeout using flag `-t`.\n\n", successCount, int32(pf.connectionNum)-successCount)
	}
	return
}

// Print the result of benchmark on console
func (pf *benchmark) Print() {
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
	runSecond := pf.endTime.Sub(pf.startTime).Seconds()
	fmt.Printf("%d requests in %.2fs, %s read, %s write\n", pf.reqNum, runSecond, formatFlow(float64(readAll)), formatFlow(float64(writeAll)))
	fmt.Printf("Requests/sec: %.2f\n", float64(pf.reqNum)/runSecond)
	fmt.Printf("Transfer/sec: %s\n", formatFlow(float64(readAll+writeAll)/runSecond))
	fmt.Printf("Error(s)    : %d\n", allError)
	sort.Ints(allTimes)
	rates := []int{50, 65, 75, 80, 90, 95, 98, 99, 100}
	fmt.Println("Percentage of the requests served within a certain time (ms)")
	for _, v := range rates {
		fmt.Printf("   %3d%%\t\t\t\t%d\n", v, allTimes[len(allTimes)*v/100-1])
	}
}

// Format the flow data
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
	return fmt.Sprintf("%.2f%v", rt, suffix)
}
