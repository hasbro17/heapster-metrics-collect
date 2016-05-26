//Program to collect metrics from the heapster service in k8s

package main

import "fmt"
import "net/http"
import "io/ioutil"
import "time"
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




func main() {


	//Define an interval of n minutes before now
	minutes := -5
	endTime := time.Now()
	startTime := endTime.Add(time.Duration(minutes)*time.Minute)

	//Format to RFC3339
	start := startTime.Format(time.RFC3339)
	end := endTime.Format(time.RFC3339)
	fmt.Println("Normal time: " + endTime.String())
	fmt.Println("RFC3339 time: " + end)

	



	//urlString := "http://aa81625c3220211e6b36e0603f64956e-1793044283.us-west-1.elb.amazonaws.com"
	//urlString := "http://54.153.53.100:80/"
	//Heapster service URL
	heapsterServiceAddr := "http://localhost:8080/api/v1/proxy/namespaces/kube-system/services/heapster"
	metricName := "cpu/limit"
	metricCmd := "/api/v1/model/metrics/" + metricName + "?start=" + start + "&end=" + end
	urlString := heapsterServiceAddr + metricCmd

	//Send GET request to Heapster service
	resp, err := http.Get(urlString);
	if err != nil {
		// handle error
		fmt.Printf("Http request error:\n", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	//Convert body byte array to string
	str := string(body[:])

    fmt.Printf("Response body:\n", str)
}
