//Program to collect metrics from the heapster service in k8s

package main

import "fmt"
import "net/http"
import "io/ioutil"
import "time"
import "strings"
import "strconv"
import "os"

//Error check helper
func check(e error) {
    if e != nil {
        panic(e)
    }
}

// Sends an http GET request to the specified URL and returns the body content as a string
// Prints an error and returns empty string on failure
func httpGetReq(urlString string) (string) {
	
	//fmt.Printf("URL: %s\n", urlString)
	resp, err := http.Get(urlString);
	check(err)
	
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	check(err)

	//Convert body byte array to string
	str := string(body[:])
	//fmt.Printf("Body String:\n %s\n\n", str);
    return str
}

//Parses the (timestamp, value) response string and extracts an array of values and timestamps from it
func extractValues(str string) ([] int, [] string) {
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

		values[i] = num
		timestamps[i] = t

	}
	return values, timestamps
}

//Extracts an array of names from the (node/pod name) response string
func extractNames(str string) ([] string) {
	//Remove all endline and spaces
	tok := strings.Split(str, "\n")
	str = strings.Join(tok, "")
	tok = strings.Fields(str);
	str = strings.Join(tok, "")

	str = str[1:len(str)-1]
	tok = strings.Split(str, ",")
	//fmt.Printf("\nAfter split:\n %s", str)

	//Make a slice of names
	names := make([]string, len(tok))
	for i, t := range tok {
		t = t[1:len(t)-1]
		names[i] = t
		//fmt.Printf("\nIndex: %d\n value:\n %s\n\n", i, t)
	}
	return names
}

//Return RFC3999 interval [start, end] of m minutes
//Where end = current time
func timeInterval(m int)(string, string) {
	minutes := -m
	endTime := time.Now()
	startTime := endTime.Add(time.Duration(minutes)*time.Minute)

	//Format to RFC3339
	start := startTime.Format(time.RFC3339)
	end := endTime.Format(time.RFC3339)
	//fmt.Printf("start=%s\n end=%s\n", start, end);

	return start, end
}

//Prepare a 3d slice for appending data
func make3Dslice(x int) [][][]int {
	arr3 := make([][][]int, x)
	for j := 0; j < x; j++ {
		arr2 := make([][]int, 0)//0 so we can append rows into it
		arr3[j] = arr2
	}
	return arr3
}

// Create a time series chart file to be drawn by gochart (https://github.com/zieckey/gochart)
// yAxisData[line number][values]
func createTimeSeriesChartFile(fileName string, chartType string, xAxisData []int, yAxisData [][]int, yAxisLineNames []string, yAxisText string){
	str := "ChartType = " + chartType +"\n" + 
	"Title = " + fileName + "\n"+
	"SubTitle = \n"+
	"\nXAxisNumbers = "
	
	//Append X axis numbers
	for i, x := range xAxisData {
		str += strconv.Itoa(x)
		if i != len(xAxisData) - 1 {
			str += ", "
		} else {
			str += "\n"	
		}
	}

	str += "\nYAxisText = " + yAxisText + "\n\n"

	//Append each row/line of the y-axis data
	for i, yRow := range yAxisData {
		str += "Data|" + yAxisLineNames[i] + " = "
		for j, y := range yRow {
			str += strconv.Itoa(y)
			if j != len(yAxisData[i]) - 1 {
				str += ", "
			} else {
				str += "\n"	
			}
		}
	}

	//Write out this string to the chart file
	path, err1 := os.Getwd()
	check(err1)
	//fmt.Printf("Path: %s\n", path)

	toks := strings.Split(fileName, "/")
	fileName = strings.Join(toks, "-")
	
	fh, err2 := os.Create(path + "/" + fileName + ".chart")
	check(err2)

	defer fh.Close()

	_, err3 := fh.WriteString(str)
    check(err3)
}

//Prepares data for calling createTimeSeriesChartFile()
//Used for nodes and pods which are 3d matrices. Not for cluster which is a 2d matrix
func generateCharts(fnamePrefix string, chartType string, metricTypes []string, names []string, metrics [][][]int) {
	//Make a line chart file for each type of cluster metric
	for i, metricType := range metricTypes {
		//Prepare xAxisData (Can be later changed to reflect actual timestamps)
		xAxisData := make([]int, 0)
		for j := 0; j < len(metrics[i][0]); j++ {
			xAxisData = append(xAxisData, j+1)
		}
		//Values
		yAxisData := make([][]int, 0)
		yAxisLineNames := make([]string, 0)

		yAxisText := metricType
		for k, name := range names {
			yAxisData = append(yAxisData, metrics[i][k])
			yAxisLineNames = append(yAxisLineNames, name)
		}
		
		//Create chart file
		createTimeSeriesChartFile(fnamePrefix + metricType, chartType, xAxisData, yAxisData, yAxisLineNames, yAxisText)
	}

}

//Check for correct arguments of minutes and chartype
func checkArgs(args []string) (int, string) {
	if len(args) != 2 {
		fmt.Printf("Usage: ./metrics_collect <interval-minutes> <chart-type>\n")
		os.Exit(1)
	}

	chartTypes := map[string]bool { "spline": true, "line": true, "bar": true, "column": true, "area": true }

	minutes, err := strconv.Atoi(args[0])
	check(err)
	chartType := args[1]

	if !chartTypes[chartType] {
		fmt.Printf("Valid Chart types: spline/line/bar/column/area\n")
		os.Exit(1)
	}

	return minutes, chartType
}



func main() {

	args := os.Args[1:]
	minutes, chartType := checkArgs(args)

	//Set time interval of measurment for last m minutes [now-m, now]
	//Heapster only has 15 minutes of data
	start, end := timeInterval(minutes)

	//Heapster service URL
	//Needs kubectl proxy running
	heapsterServiceURLPrefix := "http://localhost:8080/api/v1/proxy/namespaces/kube-system/services/heapster"

	//Cluster metrics
	clusterMetricTypes := [] string{"cpu/usage_rate", "memory/usage"}
	//Node metrics
	nodeMetricTypes := [] string{"cpu/node_utilization", "memory/node_utilization", "memory/working_set", "network/tx_rate"}
	//Pod metrics
	podMetricTypes := [] string{"cpu/usage_rate", "memory/usage", "memory/working_set", "network/tx_rate"}


	//Get list of node names
	responseStr := httpGetReq(heapsterServiceURLPrefix + "/api/v1/model/nodes/")
	nodeNames := extractNames(responseStr)

	//Get list of pod names
	responseStr = httpGetReq(heapsterServiceURLPrefix + "/api/v1/model/namespaces/default/pods/")
	podNames := extractNames(responseStr)

	//Matrix variables to store final metrics
	clusterMetrics := make([][]int, 0) // 2D[metric type][values]
	nodeMetrics := make3Dslice(len(nodeMetricTypes)) //3D[metric type][node name][values]
	podMetrics := make3Dslice(len(podMetricTypes)) //3D[metric type][pod name][values]


	//Get all metrics for the cluster
	//fmt.Printf("\n\nCLUSTER METRICS\n")
	for _, metricType := range clusterMetricTypes {
		metricCmd := "/api/v1/model/metrics/" + metricType + "?start=" + start + "&end=" + end
		responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
		values, _ := extractValues(responseStr)
		//fmt.Printf("%s: %v\n", metricType, values)
		clusterMetrics = append(clusterMetrics, values)
	}

	//Get all metrics for each node
	//fmt.Printf("\n\nNODE METRICS\n")
	for i, metricType := range nodeMetricTypes {
		//fmt.Printf("\nMetric Type: %s\n", metricType)
		for _, nodeName := range nodeNames {
			metricCmd := "/api/v1/model/nodes/" + nodeName + "/metrics/" + metricType + "?start=" + start + "&end=" + end
			responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
			values, _ := extractValues(responseStr)
			//fmt.Printf("%s: %v\n", nodeName, values)
			nodeMetrics[i] = append(nodeMetrics[i], values)
		}
		
	}


	//Get all metrics for each pod
	//fmt.Printf("\n\nPOD METRICS\n")
	for i, metricType := range podMetricTypes {
		//fmt.Printf("\nMetric Type: %s\n", metricType)
		for _, podName := range podNames {
			metricCmd := "/api/v1/model/namespaces/default/pods/" + podName + "/metrics/" + metricType + "?start=" + start + "&end=" + end
			responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
			values, _ := extractValues(responseStr)
			//fmt.Printf("%s: %v\n", podName, values)
			podMetrics[i] = append(podMetrics[i], values)
		}
		
	}


	///////// Generate chart files from matrices ////////////

	//CLUSTER CHARTS
	//Filename prefix
	fnamePrefix := "Cluster-"
	//Make a line chart file for each type of cluster metric
	for i, metricType := range clusterMetricTypes {
		//Prepare xAxisData (Can be later changed to reflect actual timestamps)
		xAxisData := make([]int, 0)
		for j := 0; j < len(clusterMetrics[i]); j++ {
			xAxisData = append(xAxisData, j+1)
		}
		//fmt.Printf("len(clusterMetrics[%d])=%d\nx-axis: %v", i, len(clusterMetrics[i]), xAxisData)
		//Values
		yAxisData := make([][]int, 0)
		yAxisLineNames := make([]string, 0)

		yAxisText := metricType
		yAxisData = append(yAxisData, clusterMetrics[i])
		yAxisLineNames = append(yAxisLineNames, "k8s-cluster")

		//Create chart file
		createTimeSeriesChartFile(fnamePrefix + metricType, chartType, xAxisData, yAxisData, yAxisLineNames, yAxisText)
	}

	//NODE CHARTS
	//Filename prefix
	fnamePrefix = "Node-"
	generateCharts(fnamePrefix, chartType, nodeMetricTypes, nodeNames, nodeMetrics)
	

	//POD CHARTs
	//Filename prefix
	fnamePrefix = "Pod-"
	generateCharts(fnamePrefix, chartType, podMetricTypes, podNames, podMetrics)
	
}
