package main

import (
	"../include/fluent-bit-go/output"
)
import (
	"C"
	// "os"
	// "strings"
	"sync"
	"fmt"
	"unsafe"
	"time"
	"net/http"
)

// ToString converts an interface into a string
func ToString(s interface{}) string {
	switch t := s.(type) {
	case []byte:
		// prevent encoding to base64
		return string(t)
	default:
		return ""
	}
}


// DataItem represents the object corresponding to the json that is sent by fluentbit tail plugin
type DataItem struct {
	LogEntry              string `json:"LogEntry"`
	LogEntrySource        string `json:"LogEntrySource"`
	LogEntryTimeStamp     string `json:"LogEntryTimeStamp"`
	LogEntryTimeOfCommand string `json:"TimeOfCommand"`
	ID                    string `json:"Id"`
	Image                 string `json:"Image"`
	Name                  string `json:"Name"`
	SourceSystem          string `json:"SourceSystem"`
	Computer              string `json:"Computer"`
}

var (
	// DataUpdateMutex read and write mutex access to the container id set
	DataUpdateMutex = &sync.Mutex{}
	// FlushedRecordsSize indicates the size of the flushed records in the current period
	FlushedRecordsSize float64
	// HTTPClient for making POST requests to OMSEndpoint
	HTTPClient http.Client
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "oms", "OMS GO!")
}

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
	// Log("Initializing out_oms go plugin for fluentbit")
	// agentVersion := os.Getenv("AGENT_VERSION")
	// if strings.Compare(strings.ToLower(os.Getenv("CONTROLLER_TYPE")), "replicaset") == 0 {
	// 	Log("Using %s for plugin config \n", ReplicaSetContainerLogPluginConfFilePath)
	// 	InitializePlugin(ReplicaSetContainerLogPluginConfFilePath, agentVersion)
	// } else {
	// 	Log("Using %s for plugin config \n", DaemonSetContainerLogPluginConfFilePath)
	// 	InitializePlugin(DaemonSetContainerLogPluginConfFilePath, agentVersion)
	// }
	// enableTelemetry := output.FLBPluginConfigKey(ctx, "EnableTelemetry")
	// if strings.Compare(strings.ToLower(enableTelemetry), "true") == 0 {
	// 	telemetryPushInterval := output.FLBPluginConfigKey(ctx, "TelemetryPushIntervalSeconds")
	// 	go SendContainerLogPluginMetrics(telemetryPushInterval)
	// } else {
	// 	Log("Telemetry is not enabled for the plugin %s \n", output.FLBPluginConfigKey(ctx, "Name"))
	// 	return output.FLB_OK
	// }
	return output.FLB_OK
}

// PostDataHelper sends data to the OMS endpoint
func PostDataHelper(tailPluginRecords []map[interface{}]interface{}) int {
	start := time.Now()
	var dataItems []DataItem

	var maxLatency float64
	var maxLatencyContainer string

	// imageIDMap := make(map[string]string)
	// nameIDMap := make(map[string]string)

	DataUpdateMutex.Lock()

	// for k, v := range ImageIDMap {
	// 	imageIDMap[k] = v
	// }
	// for k, v := range NameIDMap {
	// 	nameIDMap[k] = v
	// }
	DataUpdateMutex.Unlock()

	for _, record := range tailPluginRecords {
		containerID := "388b1a08956c78beab8cbaea5168d89a1e76c019d9ce53994bea50dff3457b0a"
		k8sNamespace := "kaveeshnamespace"
		_ := "kaveeshpod"
		logEntrySource := ToString(record["stream"])

		// if strings.EqualFold(logEntrySource, "stdout") {
		// 	if containerID == "" || containsKey(StdoutIgnoreNsSet, k8sNamespace) {
		// 		continue
		// 	}
		// } else if strings.EqualFold(logEntrySource, "stderr") {
		// 	if containerID == "" || containsKey(StderrIgnoreNsSet, k8sNamespace) {
		// 		continue
		// 	}
		// }

		stringMap := make(map[string]string)

		stringMap["LogEntry"] = ToString(record["log"])
		stringMap["LogEntrySource"] = logEntrySource
		stringMap["LogEntryTimeStamp"] = ToString(record["time"])
		stringMap["SourceSystem"] = "Containers"
		stringMap["Id"] = containerID
		stringMap["Image"] = "d49a338d9831fce14af5ea7e329373f599a4908446449f3bac9da9a75bfb06a6"
		stringMap["Name"] = "bragi92/python"

		// if val, ok := imageIDMap[containerID]; ok {
		// 	stringMap["Image"] = val
		// }

		// if val, ok := nameIDMap[containerID]; ok {
		// 	stringMap["Name"] = val
		// }

		var dataItem DataItem
		if enrichContainerLogs == true {
			dataItem = DataItem{
				ID:                    stringMap["Id"],
				LogEntry:              stringMap["LogEntry"],
				LogEntrySource:        stringMap["LogEntrySource"],
				LogEntryTimeStamp:     stringMap["LogEntryTimeStamp"],
				LogEntryTimeOfCommand: start.Format(time.RFC3339),
				SourceSystem:          stringMap["SourceSystem"],
				Computer:              "kaveeshDesktop",
				Image:                 stringMap["Image"],
				Name:                  stringMap["Name"],
			}
		} else { // dont collect timeofcommand field as its part of container log enrichment [But currently we dont know the ux behavior , so waiting for ux fix (LA ux)]
			dataItem = DataItem{
				ID:                    stringMap["Id"],
				LogEntry:              stringMap["LogEntry"],
				LogEntrySource:        stringMap["LogEntrySource"],
				LogEntryTimeStamp:     stringMap["LogEntryTimeStamp"],
				LogEntryTimeOfCommand: start.Format(time.RFC3339),
				SourceSystem:          stringMap["SourceSystem"],
				Computer:              "kaveeshDesktop",
				Image:                 stringMap["Image"],
				Name:                  stringMap["Name"],
			}
		}

		FlushedRecordsSize += float64(len(stringMap["LogEntry"]))

		dataItems = append(dataItems, dataItem)
		if dataItem.LogEntryTimeStamp != "" {
			loggedTime, e := time.Parse(time.RFC3339, dataItem.LogEntryTimeStamp)
			if e != nil {
				message := fmt.Sprintf("Error while converting LogEntryTimeStamp for telemetry purposes: %s", e.Error())
				// Log(message)
				// SendException(message)
			} else {
				ltncy := float64(start.Sub(loggedTime) / time.Millisecond)
				if ltncy >= maxLatency {
					maxLatency = ltncy
					maxLatencyContainer = dataItem.Name + "=" + dataItem.ID
				}
			}
		}
	}

	var OMSEndpoint string;
	OMSEndpoint := "https://5e0e87ea-67ac-4779-b6f7-30173b69112a.ods.opinsights.azure.com/OperationalData.svc/PostJsonDataItemsAZURE_RESOURCE_ID=";

	if len(dataItems) > 0 {
		logEntry := ContainerLogBlob{
			DataType:  ContainerLogDataType,
			IPName:    IPName,
			DataItems: dataItems}

		marshalled, err := json.Marshal(logEntry)
		if err != nil {
			message := fmt.Sprintf("Error while Marshalling log Entry: %s", err.Error())
			// Log(message)
			// SendException(message)
			return output.FLB_OK
		}

		req, _ := http.NewRequest("POST", OMSEndpoint, bytes.NewBuffer(marshalled))
		req.Header.Set("Content-Type", "application/json")
		//expensive to do string len for every request, so use a flag
		// if ResourceCentric == true {
		// 	req.Header.Set("x-ms-AzureResourceId", ResourceID)
		// }

		resp, err := HTTPClient.Do(req)
		elapsed := time.Since(start)

		if err != nil {
			message := fmt.Sprintf("Error when sending request %s \n", err.Error())
			// Log(message)
			// Commenting this out for now. TODO - Add better telemetry for ods errors using aggregation
			//SendException(message)
			// Log("Failed to flush %d records after %s", len(dataItems), elapsed)

			return output.FLB_RETRY
		}

		if resp == nil || resp.StatusCode != 200 {
			// if resp != nil {
			// 	Log("Status %s Status Code %d", resp.Status, resp.StatusCode)
			// }
			return output.FLB_RETRY
		}

		defer resp.Body.Close()
		numRecords := len(dataItems)
		// Log("PostDataHelper::Info::Successfully flushed %d records in %s", numRecords, elapsed)
		ContainerLogTelemetryMutex.Lock()
		FlushedRecordsCount += float64(numRecords)
		FlushedRecordsTimeTaken += float64(elapsed / time.Millisecond)

		if maxLatency >= AgentLogProcessingMaxLatencyMs {
			AgentLogProcessingMaxLatencyMs = maxLatency
			AgentLogProcessingMaxLatencyMsContainer = maxLatencyContainer
		}

		ContainerLogTelemetryMutex.Unlock()
	}

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var ret int
	var record map[interface{}]interface{}
	var records []map[interface{}]interface{}

	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))

	// Iterate Records
	for {
		// Extract Record
		ret, _, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}
		records = append(records, record)
	}

	// incomingTag := strings.ToLower(C.GoString(tag))
	// if strings.Contains(incomingTag, "oms.container.log.flbplugin") {
	// 	// This will also include populating cache to be sent as for config events
	// 	return PushToAppInsightsTraces(records, appinsights.Information, incomingTag)
	// } else if strings.Contains(incomingTag, "oms.container.perf.telegraf") {
	// 	return PostTelegrafMetricsToLA(records)
	// }

	return PostDataHelper(records)
}

// FLBPluginExit exits the plugin
func FLBPluginExit() int {
	// ContainerLogTelemetryTicker.Stop()
	// ContainerImageNameRefreshTicker.Stop()
	return output.FLB_OK
}

func main() {
}



// package main


// import (
// 	"C"
// 	"unsafe"
// )

// import "../include/fluent-bit-go/output"


// // //export FLBPluginRegister
// func FLBPluginRegister(ctx unsafe.Pointer) int {
// 	return output.FLBPluginRegister(ctx, "oms", "OMS GO!")
// }

// //export FLBPluginInit
// func FLBPluginInit(plugin unsafe.Pointer) int {
//     // Gets called only once for each instance you have configured.
//     return output.FLB_OK
// }

// //export FLBPluginFlushCtx
// func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
//     // Gets called with a batch of records to be written to an instance.
//     return output.FLB_OK
// }

// //export FLBPluginExit
// func FLBPluginExit() int {
// 	return output.FLB_OK
// }

// func main() {
// }