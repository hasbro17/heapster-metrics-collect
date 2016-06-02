//Script to collect metrics from the heapster service in k8s,
//and generate chart files to be plotted by gochart (https://github.com/zieckey/gochart)

package main

import "fmt"
import "net/http"
import "io/ioutil"
import "time"
import "strings"
import "strconv"
import "os"
//import "math"
//import "sort"
import "./gochartgen"

//Error check helper
func check(e error) {
    if e != nil {
        panic(e)
    }
}

// Sends an http GET request to the specified URL and returns the body content as a string
// Prints an error and returns empty string on failure
func httpGetReq(urlString string) (string) {

	//fmt.Printf("Request:\n %s\n\n", urlString)
	resp, err := http.Get(urlString);
	check(err)
	
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	check(err)

	//Convert body byte array to string
	str := string(body[:])
	//fmt.Printf("Response:\n %s\n\n", str);
    return str
}

//Parses the (timestamp, value) response string and extracts an array of values and timestamps from it
func extractValues(str string, sliceTS []string) ([]int, []string, []int) {

	//Remove all endline and spaces
	toks := strings.Split(str, "\n")
	str = strings.Join(toks, "")
	toks = strings.Fields(str);
	str = strings.Join(toks, "")
	//fmt.Printf("\nBefore split:\n %s", str)

	i := strings.Index(str, "[")
	j := strings.Index(str, "]")
	str = str[i+1:j]
	//fmt.Printf("\nAfter split:\n %s", str)
	toks = strings.Split(str, "}")
	toks = toks[:len(toks)-1] //remove the last empty token

	//Make a slice of values
	values := make([]int, len(toks))
	timestamps := make([]string, len(toks))

	for i, tok := range toks {
		//Get value
		tmp := strings.Split(tok, "\"value\":")
		v := tmp[1];

		//Get time stamp
		//fmt.Printf("\nTimeStamp: %s\n", tmp[0])
		tmp = strings.Split(tmp[0], "{\"timestamp\":")
		t := tmp[1];
		t = t[1:len(t)-2]
		//fmt.Printf("\nTimeStamp: %s\n", t)
		
		//Convert value to int
		num, err := strconv.Atoi(v)
		check(err)

		//Check if
		values[i] = num
		timestamps[i] = t

	}

	//Extended values for all timestamps, for plotting purposes
	extendedValues := make([]int, len(sliceTS))
	for i, _ := range extendedValues {
		extendedValues[i] = -1; //default missing value
	}

	//sliceTSS = shortenTSArray(sliceTS)
	timestampsS := shortenTSArray(timestamps)

	//fmt.Printf("SliceTS len(%d): %v\n\n", len(sliceTS), sliceTS)
	//fmt.Printf("Timestamps len(%d): %v\n\n", len(timestampsS), timestampsS)


	var l int = 0
	for k, val := range values {
		//Find the right index for this value, returned timestamps must be in order for this to work
		for sliceTS[l] != timestampsS[k] {
			l++
		}
		extendedValues[l] = val
	}


	return values, timestamps, extendedValues
}

//Extracts an array of names from the (node/pod name) response string
func extractNames(str string) ([]string) {

	//Remove all endline and spaces
	tok := strings.Split(str, "\n")
	str = strings.Join(tok, "")
	tok = strings.Fields(str);
	str = strings.Join(tok, "")

	str = str[1:len(str)-1]
	tok = strings.Split(str, ",")
	//fmt.Printf("\nAfter split:\n %s", str)
	//if()

	//Make a slice of names
	if(len(tok[0]) == 0) { //If no names present, return empty slice
		names := make([]string, 0)
		fmt.Printf("")
		return names
	} else {
		names := make([]string, len(tok))
		for i, t := range tok {
			t = t[1:len(t)-1]
			names[i] = t
			//fmt.Printf("\nIndex: %d\n value:\n %s\n\n", i, t)
		}
		return names
	}
	
}

//Shorten the RFC399 timestamp to just minute:seconds
func shortenTimeStamp(ts string)(string) {
	toks := strings.Split(ts, "T")
	//toks = strings.Split(toks[1], "-")
	str := toks[1]
	str = str[3:8]
	return str
}

func shortenTSArray(sliceTS []string)([]string){
	for i, ts := range sliceTS {
		sliceTS[i] = shortenTimeStamp(ts)
	}
	return sliceTS
}

//Return RFC3999 interval [start, end] of m minutes
//Where end = current time
//resoultion in seconds
func timeInterval(m int, resolution int, urlPrefix string)(string, string, map[string]bool, []string) {

	minutes := -m
	endTime := time.Now()
	intervalDuration := time.Duration(minutes)*time.Minute
	startTime := endTime.Add(intervalDuration)

	//Format to RFC3339
	start := startTime.Format(time.RFC3339)
	end := endTime.Format(time.RFC3339)
	//fmt.Printf("start=%s\n end=%s\n", start, end);

	//Get actual latest timestamp from cluster
	metricCmd := "/api/v1/model/metrics/cpu/usage_rate" + "?start=" + start + "&end=" + end
	str := httpGetReq(urlPrefix + metricCmd)

	//Remove all endline and spaces
	toks := strings.Split(str, "\n")
	str = strings.Join(toks, "")
	toks = strings.Fields(str);
	str = strings.Join(toks, "")

	//Hardcoded way to get latest timestamp TODO:Fix later
	str = str[len(str)-8:len(str)-3]
	//update the end time value
	newEndStr := end[:len(end)-11] + str + end[len(end)-6:]
	//Parse it to update end time


	//fmt.Printf("Previous end: %v\n\n", end)
	//fmt.Printf("Updated end: %v\n\n", updatedEnd)

	newEnd, err := time.Parse(time.RFC3339, newEndStr)
	check(err)

	//fmt.Printf("New end: %v\n\n", newEnd)

	//Construct map of expected timestamps
	res := time.Duration(resolution) * time.Second
	steps := int(-intervalDuration/res)

	//Map of timestamps present
	timestampsMap := make(map[string]bool)
	timestampsSlice := make([]string, steps+1)
	ts := newEnd.Add(intervalDuration)
	for i := 0; i <= steps; i++ {
		t := ts.Format(time.RFC3339)
		timestampsMap[t] = true
		timestampsSlice[i] = t
		ts = ts.Add(res)
	}

	//fmt.Printf("TS Map: %v\n\n", timestampsMap)

	//fmt.Printf("Sorted Array: %v\n\n", timestampsSlice)


	return start, end, timestampsMap, timestampsSlice
}

//Prepare a 3d slice for appending data
func make3DsliceInt(innerSliceLen int) [][][]int {

	arr3 := make([][][]int, innerSliceLen)
	for j := 0; j < innerSliceLen; j++ {
		arr2 := make([][]int, 0)//0 so we can append rows into it
		arr3[j] = arr2
	}
	return arr3
}

func make3DsliceString(innerSliceLen int) [][][]string {

	arr3 := make([][][]string, innerSliceLen)
	for j := 0; j < innerSliceLen; j++ {
		arr2 := make([][]string, 0)//0 so we can append rows into it
		arr3[j] = arr2
	}
	return arr3
}

//Prepares data for calling createTimeSeriesChartFile()
//Used for nodes and pods which are 3d matrices. Not for cluster which is a 2d matrix
func generateCharts(fnamePrefix string, chartType string, metricTypes []string, names []string, metrics [][][]int, shortTS []string) {

	//Make a line chart file for each type of cluster metric
	for i, metricType := range metricTypes {
/*
		//Get maximum length of any data row for this metric type
		maxLen := 0
		for _, arr := range metrics[i] {
			maxLen = int(math.Max(float64(len(arr)), float64(maxLen)))
		}

		//Prepare xAxisData (Can be later changed to reflect actual timestamps)
		xAxisData := make([]int, 0)
		for j := 0; j < maxLen; j++ {
			xAxisData = append(xAxisData, j+1)
		}
*/
		xAxisData := shortTS
		//Values
		yAxisData := make([][]int, 0)
		yAxisLineNames := make([]string, 0)

		yAxisText := metricType
		for k, name := range names {
			yAxisData = append(yAxisData, metrics[i][k])
			yAxisLineNames = append(yAxisLineNames, name)
		}
		
		//Create chart file
		//gochartgen.CreateTimeSeriesChartFile(fnamePrefix + metricType, chartType, xAxisData, yAxisData, yAxisLineNames, yAxisText)
		gochartgen.CreateTimeSeriesChartFileTS(fnamePrefix + metricType, chartType, xAxisData, yAxisData, yAxisLineNames, yAxisText)
	}
}

//Check for correct arguments of minutes and chartype
func checkArgs(args []string) (int, int, string) {

	if len(args) != 3 {
		fmt.Printf("Usage: ./metrics_collect <heapster-resolution> <interval-minutes> <chart-type>\n")
		os.Exit(1)
	}

	chartTypes := map[string]bool { "spline": true, "line": true, "bar": true, "column": true, "area": true }

	fmt.Printf("Map: %v\n\n", chartTypes)

	resolution, err := strconv.Atoi(args[0])
	check(err)

	minutes, err := strconv.Atoi(args[1])
	check(err)

	chartType := args[2]

	if !chartTypes[chartType] {
		fmt.Printf("Valid Chart types: spline/line/bar/column/area\n")
		os.Exit(1)
	}

	return resolution, minutes, chartType
}



func main() {

	args := os.Args[1:]
	resolution, minutes, chartType := checkArgs(args)


	//Heapster service URL
	//Needs kubectl proxy running
	heapsterServiceURLPrefix := "http://localhost:8080/api/v1/proxy/namespaces/kube-system/services/heapster-custom"

	//Set time interval of measurment for last m minutes [now-m, now]
	//Heapster only has 15 minutes of data
	start, end, _, sliceTS := timeInterval(minutes, resolution, heapsterServiceURLPrefix)

	sliceTS = shortenTSArray(sliceTS)

	//Cluster metrics
	clusterMetricTypes := [] string{"cpu/usage_rate", "memory/usage"}
	//Node metrics
	nodeMetricTypes := [] string{}//"cpu/node_utilization", "memory/node_utilization", "memory/working_set", "network/tx_rate"}
	//Pod metrics
	podMetricTypes := [] string{"cpu/usage_rate", "memory/usage", "memory/working_set", "network/tx_rate"}


	//Get list of node names
	responseStr := httpGetReq(heapsterServiceURLPrefix + "/api/v1/model/nodes/")
	nodeNames := extractNames(responseStr)
	if len(nodeNames) == 0 {
		fmt.Printf("Error: No nodeNames returned\n")
	}
	//Get list of pod names
	responseStr = httpGetReq(heapsterServiceURLPrefix + "/api/v1/model/namespaces/default/pods/")
	podNames := extractNames(responseStr)
	if len(podNames) == 0 {
		fmt.Printf("Error: No podNames returned\n")
	}

	//Matrix variables to store final metrics ints
	clusterMetrics := make([][]int, 0) // 2D[metric type][values]
	nodeMetrics := make3DsliceInt(len(nodeMetricTypes)) //3D[metric type][node name][values]
	podMetrics := make3DsliceInt(len(podMetricTypes)) //3D[metric type][pod name][values]

	//Timestamp(RFC3999) string matrices for the above metrics
	clusterTS := make([][]string, 0) // 2D[metric type][ts]
	nodeTS := make3DsliceString(len(nodeMetricTypes)) //3D[metric type][node name][ts]
	podTS := make3DsliceString(len(podMetricTypes)) //3D[metric type][pod name][ts]

	//Extended values for plotting missing values as well
	clusterMetricsE := make([][]int, 0) // 2D[metric type][values]
	nodeMetricsE := make3DsliceInt(len(nodeMetricTypes)) //3D[metric type][node name][values]
	podMetricsE := make3DsliceInt(len(podMetricTypes)) //3D[metric type][pod name][values]

	//Get all metrics for the cluster
	fmt.Printf("\nCLUSTER METRICS\n\n")
	for _, metricType := range clusterMetricTypes {
		metricCmd := "/api/v1/model/metrics/" + metricType + "?start=" + start + "&end=" + end
		responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
		values, timestamps, eValues := extractValues(responseStr, sliceTS)
		fmt.Printf("%s: %v\n", metricType, values)
		clusterMetrics = append(clusterMetrics, values)
		clusterTS = append(clusterTS, timestamps)
		clusterMetricsE = append(clusterMetricsE, eValues)
	}

	//Get all metrics for each node
	fmt.Printf("\n\nNODE METRICS\n")
	for i, metricType := range nodeMetricTypes {
		fmt.Printf("\nMetric Type: %s\n", metricType)
		for _, nodeName := range nodeNames {
			metricCmd := "/api/v1/model/nodes/" + nodeName + "/metrics/" + metricType + "?start=" + start + "&end=" + end
			responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
			values, timestamps, eValues := extractValues(responseStr, sliceTS)
			fmt.Printf("%s: %v\n", nodeName, values)
			nodeMetrics[i] = append(nodeMetrics[i], values)
			nodeTS[i] = append(nodeTS[i], timestamps)
			nodeMetricsE[i] = append(nodeMetricsE[i], eValues)
		}
	}


	//Get all metrics for each pod
	fmt.Printf("\n\nPOD METRICS\n")
	for i, metricType := range podMetricTypes {
		fmt.Printf("\nMetric Type: %s\n", metricType)
		for _, podName := range podNames {
			metricCmd := "/api/v1/model/namespaces/default/pods/" + podName + "/metrics/" + metricType + "?start=" + start + "&end=" + end
			responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
			values, timestamps, eValues := extractValues(responseStr, sliceTS)
			fmt.Printf("%s: %v\n", podName, values)
			//fmt.Printf("%s: %v\n", timestamps, timestamps)
			podMetrics[i] = append(podMetrics[i], values)
			podTS[i] = append(podTS[i], timestamps)
			podMetricsE[i] = append(podMetricsE[i], eValues)
		}
	}


	///////// Generate chart files from the matrices ////////////

	//Shorten the timestamps first
	//shortTS := make([]string, len(sliceTS))
//	for i, ts := range sliceTS {
//		shortTS[i] = shortenTimeStamp(ts)
//	}

	//CLUSTER CHARTS
	//Filename prefix
	fnamePrefix := "Cluster-"
	//Make a line chart file for each type of cluster metric
	for i, metricType := range clusterMetricTypes {
		//Prepare xAxisData (Can be later changed to reflect actual timestamps)
		/*
		xAxisData := make([]int, 0)
		for j := 0; j < len(clusterMetrics[i]); j++ {
			xAxisData = append(xAxisData, j+1)
		}
		*/
		xAxisData := sliceTS
		//fmt.Printf("len(clusterMetrics[%d])=%d\nx-axis: %v", i, len(clusterMetrics[i]), xAxisData)
		//Values
		yAxisData := make([][]int, 0)
		yAxisLineNames := make([]string, 0)

		yAxisText := metricType
		yAxisData = append(yAxisData, clusterMetricsE[i])
		yAxisLineNames = append(yAxisLineNames, "k8s-cluster")

		//Create chart file
		//gochartgen.CreateTimeSeriesChartFile(fnamePrefix + metricType, chartType, xAxisData, yAxisData, yAxisLineNames, yAxisText)
		gochartgen.CreateTimeSeriesChartFileTS(fnamePrefix + metricType, chartType, xAxisData, yAxisData, yAxisLineNames, yAxisText)
	}

	//NODE CHARTS
	//Filename prefix
	fnamePrefix = "Node-"
	generateCharts(fnamePrefix, chartType, nodeMetricTypes, nodeNames, nodeMetricsE, sliceTS)
	

	//POD CHARTs
	//Filename prefix
	fnamePrefix = "Pod-"
	generateCharts(fnamePrefix, chartType, podMetricTypes, podNames, podMetricsE, sliceTS)
	
}
