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

	resp, err := http.Get(urlString);
	check(err)
	
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	check(err)

	//Convert body byte array to string
	str := string(body[:])
    return str
}

//Removes whitespace and newline characters from string
func removeWhitespace(str string)(string){
	toks := strings.Split(str, "\n")
	str = strings.Join(toks, "")
	toks = strings.Fields(str);
	str = strings.Join(toks, "")
	return str
}

//Parses the (timestamp, value) response string and extracts an array of values and timestamps from it
func extractValues(str string, allTS []string) ([]int, []string, []int) {

	//Remove all endline and spaces
	str = removeWhitespace(str)

	i := strings.Index(str, "[")
	j := strings.Index(str, "]")
	str = str[i+1:j]

	toks := strings.Split(str, "}")
	toks = toks[:len(toks)-1] //remove the last empty token

	//Slices of values and timestamps
	values := make([]int, len(toks))
	timestamps := make([]string, len(toks))

	for i, tok := range toks {
		//Get value
		tmp := strings.Split(tok, "\"value\":")
		v := tmp[1];
		//Convert value to int
		num, err := strconv.Atoi(v)
		check(err)
		values[i] = num

		//Get time stamp
		tmp = strings.Split(tmp[0], "{\"timestamp\":")
		t := tmp[1];
		t = t[1:len(t)-2]
		timestamps[i] = t
	}

	//Construct a slice of values for all timestamps in the range of allTS
	//Useful for plotting purposes to see missing (value,timestamp) pairs
	allValues := make([]int, len(allTS))
	for i, _ := range allValues {
		allValues[i] = -100; //default value for missing data point
	}

	tsShort := shortenTSArray(timestamps)

	var l int = 0
	//For every value
	for k, val := range values {
		//Find the index of its matching timestamp, and place the value there.
		for allTS[l] != tsShort[k] {
			l++
		}
		allValues[l] = val
	}

	return values, timestamps, allValues
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
		}
		return names
	}
	
}

//Shorten an RFC399 timestamp to just minute:seconds string
func shortenTimeStamp(ts string)(string) {
	toks := strings.Split(ts, "T")
	str := toks[1]
	str = str[3:8]
	return str
}

//Shorten an entire array of timestamps to minute:seconds format
func shortenTSArray(sliceTS []string)([]string){
	for i, ts := range sliceTS {
		sliceTS[i] = shortenTimeStamp(ts)
	}
	return sliceTS
}

//Return RFC3999 interval [start, end] of m minutes
//Where end = current time
//resoultion in seconds
func timeInterval(m int, resolution int, urlPrefix string)(string, string, []string) {

	//Construct start and end times
	minutes := -m
	endTime := time.Now()
	intervalDuration := time.Duration(minutes)*time.Minute
	startTime := endTime.Add(intervalDuration)

	//Format to RFC3339
	startTimeStr := startTime.Format(time.RFC3339)
	endTimeStr := endTime.Format(time.RFC3339)

	//Get actual latest timestamp from cluster
	metricCmd := "/api/v1/model/metrics/cpu/usage_rate" + "?start=" + startTimeStr + "&end=" + endTimeStr
	str := httpGetReq(urlPrefix + metricCmd)

	//Hardcoded way to parse and extract the latest timestamp from response string 
	//TODO:Fix later
	str = removeWhitespace(str)
	str = str[len(str)-8:len(str)-3]

	//Update the end time value for difference in min:seconds from latest timestamp
	newEndStr := endTimeStr[:len(endTimeStr)-11] + str + endTimeStr[len(endTimeStr)-6:]
	//Parse it to update end time
	newEndTime, err := time.Parse(time.RFC3339, newEndStr)
	check(err)

	//Construct slice of expected timestamps
	res := time.Duration(resolution) * time.Second
	steps := int(-intervalDuration/res)

	//timestampsMap := make(map[string]bool)
	timestampsSlice := make([]string, steps+1)
	ts := newEndTime.Add(intervalDuration)
	for i := 0; i <= steps; i++ {
		t := ts.Format(time.RFC3339)
		//timestampsMap[t] = true
		timestampsSlice[i] = t
		ts = ts.Add(res)
	}

	return startTimeStr, endTimeStr, timestampsSlice
}

//Prepare a 3d slice for appending integer data
func make3DsliceInt(innerSliceLen int) [][][]int {

	arr3 := make([][][]int, innerSliceLen)
	for j := 0; j < innerSliceLen; j++ {
		arr2 := make([][]int, 0)//0 so we can append rows into it
		arr3[j] = arr2
	}
	return arr3
}

//Prepare a 3d slice for appending string data
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
	start, end, sliceTS := timeInterval(minutes, resolution, heapsterServiceURLPrefix)

	sliceTS = shortenTSArray(sliceTS)

	//Cluster metrics
	clusterMetricTypes := [] string{"cpu/usage_rate"}//, "memory/usage"}
	//Node metrics
	nodeMetricTypes := [] string{}//"cpu/node_utilization", "memory/node_utilization", "memory/working_set", "network/tx_rate"}
	//Pod metrics
	podMetricTypes := [] string{"cpu/usage_rate"}//, "memory/usage", "memory/working_set", "network/tx_rate"}


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
			podMetrics[i] = append(podMetrics[i], values)
			podTS[i] = append(podTS[i], timestamps)
			podMetricsE[i] = append(podMetricsE[i], eValues)
		}
	}


	///////// Generate chart files from the matrices ////////////

	//CLUSTER CHARTS
	//Filename prefix
	fnamePrefix := "Cluster-"
	//Make a line chart file for each type of cluster metric
	for i, metricType := range clusterMetricTypes {
		xAxisData := sliceTS
		//Values
		yAxisData := make([][]int, 0)
		yAxisLineNames := make([]string, 0)

		yAxisText := metricType
		yAxisData = append(yAxisData, clusterMetricsE[i])
		yAxisLineNames = append(yAxisLineNames, "k8s-cluster")

		//Create chart file
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
