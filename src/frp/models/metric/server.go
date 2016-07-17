// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"encoding/json"
	"sync"
	"time"

	"frp/models/consts"
)

var (
	DailyDataKeepDays   int = 7
	ServerMetricInfoMap map[string]*ServerMetric
	smMutex             sync.RWMutex
)

type ServerMetric struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	BindAddr      string   `json:"bind_addr"`
	ListenPort    int64    `json:"listen_port"`
	CustomDomains []string `json:"custom_domains"`
	Status        string   `json:"status"`
	UseEncryption bool     `json:"use_encryption"`
	UseGzip       bool     `json:"use_gzip"`
	PrivilegeMode bool     `json:"privilege_mode"`

	// statistics
	CurrentConns int64               `json:"current_conns"`
	Daily        []*DailyServerStats `json:"daily"`
	mutex        sync.RWMutex
}

type DailyServerStats struct {
	Time             string `json:"time"`
	FlowIn           int64  `json:"flow_in"`
	FlowOut          int64  `json:"flow_out"`
	TotalAcceptConns int64  `json:"total_accept_conns"`
}

func init() {
	ServerMetricInfoMap = make(map[string]*ServerMetric)
}

func GetAllProxyMetrics() map[string]*ServerMetric {
	result := make(map[string]*ServerMetric)
	smMutex.RLock()
	defer smMutex.RUnlock()
	for proxyName, metric := range ServerMetricInfoMap {
		metric.mutex.RLock()
		byteBuf, _ := json.Marshal(metric)
		metric.mutex.RUnlock()
		tmpMetric := &ServerMetric{}
		json.Unmarshal(byteBuf, &tmpMetric)
		result[proxyName] = tmpMetric
	}
	return result
}

// if proxyName isn't exist, return nil
func GetProxyMetrics(proxyName string) *ServerMetric {
	smMutex.RLock()
	defer smMutex.RUnlock()
	metric, ok := ServerMetricInfoMap[proxyName]
	if ok {
		byteBuf, _ := json.Marshal(metric)
		tmpMetric := &ServerMetric{}
		json.Unmarshal(byteBuf, &tmpMetric)
		return tmpMetric
	} else {
		return nil
	}
}

func SetProxyInfo(proxyName string, proxyType, bindAddr string,
	useEncryption, useGzip, privilegeMode bool, customDomains []string,
	listenPort int64) {
	smMutex.Lock()
	info, ok := ServerMetricInfoMap[proxyName]
	if !ok {
		info = &ServerMetric{}
		info.Daily = make([]*DailyServerStats, 0)
	}
	info.Name = proxyName
	info.Type = proxyType
	info.UseEncryption = useEncryption
	info.UseGzip = useGzip
	info.PrivilegeMode = privilegeMode
	info.BindAddr = bindAddr
	info.ListenPort = listenPort
	info.CustomDomains = customDomains
	ServerMetricInfoMap[proxyName] = info
	smMutex.Unlock()
}

func SetStatus(proxyName string, status int64) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.Status = consts.StatusStr[status]
		metric.mutex.Unlock()
	}
}

type DealFuncType func(*DailyServerStats)

func DealDailyData(dailyData []*DailyServerStats, fn DealFuncType) (newDailyData []*DailyServerStats) {
	now := time.Now().Format("20060102")
	dailyLen := len(dailyData)
	if dailyLen == 0 {
		daily := &DailyServerStats{}
		daily.Time = now
		fn(daily)
		dailyData = append(dailyData, daily)
	} else {
		daily := dailyData[dailyLen-1]
		if daily.Time == now {
			fn(daily)
		} else {
			newDaily := &DailyServerStats{}
			newDaily.Time = now
			fn(newDaily)
			if dailyLen == DailyDataKeepDays {
				for i := 0; i < dailyLen-1; i++ {
					dailyData[i] = dailyData[i+1]
				}
				dailyData[dailyLen-1] = newDaily
			} else {
				dailyData = append(dailyData, newDaily)
			}
		}
	}
	return dailyData
}

func OpenConnection(proxyName string) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.CurrentConns++
		metric.Daily = DealDailyData(metric.Daily, func(stats *DailyServerStats) {
			stats.TotalAcceptConns++
		})
		metric.mutex.Unlock()
	}
}

func CloseConnection(proxyName string) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.CurrentConns--
		metric.mutex.Unlock()
	}
}

func AddFlowIn(proxyName string, value int64) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.Daily = DealDailyData(metric.Daily, func(stats *DailyServerStats) {
			stats.FlowIn += value
		})
		metric.mutex.Unlock()
	}
}

func AddFlowOut(proxyName string, value int64) {
	smMutex.RLock()
	metric, ok := ServerMetricInfoMap[proxyName]
	smMutex.RUnlock()
	if ok {
		metric.mutex.Lock()
		metric.Daily = DealDailyData(metric.Daily, func(stats *DailyServerStats) {
			stats.FlowOut += value
		})
		metric.mutex.Unlock()
	}
}