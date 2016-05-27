//Program to collect metrics from the heapster service in k8s

package main

import "fmt"
import "net/http"
import "io/ioutil"
import "time"
import "strings"
import "strconv"

//import "net"

/* Metrics to collect
Cluster Metrics:
cpu/usage_rate
memory/usage

Node Metrics:
cpu/node_utilization
memory/node_utilization
memory/working_set
network/tx_rate

Pod Metrics:
cpu/usage_rate: usage in millicores
memory/usage
memory/working_set
network/tx_rate
*/

// Sends an http GET request to the specified URL and returns the body as a string
// Prints an error and returns empty string on failure
func httpGetReq(urlString string) (string) {
	
	//fmt.Printf("URL: %s\n", urlString)
	resp, err := http.Get(urlString);
	if err != nil {
		// handle error
		fmt.Printf("Http request error:\n", err)
		return "";
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
		fmt.Printf("Body read error:\n", err)
		return "";
	}

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
		if err != nil {
			// handle error
			fmt.Printf("Atoi err:\n", err)
			num = 0;
		}
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

func timeInterval(m int)(string, string) {
	minutes := -m
	endTime := time.Now()
	startTime := endTime.Add(time.Duration(minutes)*time.Minute)

	//Format to RFC3339
	start := startTime.Format(time.RFC3339)
	end := endTime.Format(time.RFC3339)

	return start, end
}


func main() {

	//Set time interval of measurment for last m minutes [now-m, now]
	start, end := timeInterval(6)

	//Heapster service URL
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


	//Get all metrics for the cluster
	fmt.Printf("\n\nCLUSTER METRICS\n")
	for _, metricType := range clusterMetricTypes {
		metricCmd := "/api/v1/model/metrics/" + metricType + "?start=" + start + "&end=" + end
		responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
		values, _ := extractValues(responseStr)
		fmt.Printf("%s: %v\n", metricType, values)
		
	}

	//Get all metrics for each node
	fmt.Printf("\n\nNODE METRICS\n")
	for _, metricType := range nodeMetricTypes {
		fmt.Printf("\nMetric Type: %s\n", metricType)
		for _, nodeName := range nodeNames {
			metricCmd := "/api/v1/model/nodes/" + nodeName + "/metrics/" + metricType + "?start=" + start + "&end=" + end
			responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
			values, _ := extractValues(responseStr)
			fmt.Printf("%s: %v\n", nodeName, values)	
		}
		
	}

	//Get all metrics for each pod
	fmt.Printf("\n\nPOD METRICS\n")
	for _, metricType := range podMetricTypes {
		fmt.Printf("\nMetric Type: %s\n", metricType)
		for _, podName := range podNames {
			metricCmd := "/api/v1/model/namespaces/default/pods/" + podName + "/metrics/" + metricType + "?start=" + start + "&end=" + end
			responseStr = httpGetReq(heapsterServiceURLPrefix + metricCmd)
			values, _ := extractValues(responseStr)
			fmt.Printf("%s: %v\n", podName, values)
		}
		
	}
	
}
