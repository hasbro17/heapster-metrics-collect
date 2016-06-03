// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"net/http"
	gourl "net/url"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/hasbro17/boom/boomer"
	"math"
	"time"
)

const (
	headerRegexp = `^([\w-]+):\s*(.+)`
	authRegexp   = `^(.+):([^\s].+)`
)

type headerSlice []string

func (h *headerSlice) String() string {
	return fmt.Sprintf("%s", *h)
}

func (h *headerSlice) Set(value string) error {
	*h = append(*h, value)
	return nil
}

var (
	headerslice headerSlice
	m           = flag.String("m", "GET", "")
	headers     = flag.String("h", "", "")
	body        = flag.String("d", "", "")
	accept      = flag.String("A", "", "")
	contentType = flag.String("T", "text/html", "")
	authHeader  = flag.String("a", "", "")

	output = flag.String("o", "", "")
	c    = flag.Int("c", 1, "")
	n    = flag.Int("n", 10000, "")
	q    = flag.Int("q", 0, "")
	t    = flag.Int("t", 0, "")
	cpus = flag.Int("cpus", runtime.GOMAXPROCS(-1), "")
	

	disableCompression = flag.Bool("disable-compression", false, "")
	disableKeepAlives  = flag.Bool("disable-keepalive", false, "")
	proxyAddr          = flag.String("x", "", "")

	//Additional sine wave flags
	sW = flag.Bool("sW", true, "")
	sP = flag.Float64("sP", 10, "The time period of a sample in seconds. E.g. 5, 10, 60")
	aS = flag.Float64("aS", 1, "The amplitude scale factor.")
	f = flag.Float64("f", float64(1 / float64(5*60)), "The frequency of the sine wave (e.g. 1/(min*60)Hz)")
)

var usage = `Usage: boom [options...] <url>

Options:
  -sW Generate a sinusoidal wave of requests, with a peak of q Qps. Default: true
  -sP The time period of a sample in seconds. E.g. 5, 10, 60
  -aS The amplitude scaling factor. Default: 1
  -fr The frequency of the sine wave (e.g. 1/(minutes*60)Hz)

  -n  Number of requests to run.
  -c  Number of requests to run concurrently. Total number of requests cannot
      be smaller than the concurency level.
  -q  Max rate limit, in seconds (QPS). Peak of the sinusoid
  -o  Output type. If none provided, a summary is printed.
      "csv" is the only supported alternative. Dumps the response
      metrics in comma-seperated values format.

  -m  HTTP method, one of GET, POST, PUT, DELETE, HEAD, OPTIONS.
  -H  Custom HTTP header. You can specify as many as needed by repeating the flag.
      for example, -H "Accept: text/html" -H "Content-Type: application/xml" .
  -t  Timeout in ms.
  -A  HTTP Accept header.
  -d  HTTP request body.
  -T  Content-type, defaults to "text/html".
  -a  Basic authentication, username:password.
  -x  HTTP Proxy address as host:port.

  -disable-compression  Disable compression.
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                        connections between different HTTP requests.
  -cpus                 Number of used cpu cores.
                        (default for current machine is %d cores)
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}

	flag.Var(&headerslice, "H", "")

	flag.Parse()
	if flag.NArg() < 1 {
		usageAndExit("")
	}

	runtime.GOMAXPROCS(*cpus)
	num := *n
	conc := *c
	q := *q

	if num <= 0 || conc <= 0 {
		usageAndExit("n and c cannot be smaller than 1.")
	}

	url := flag.Args()[0]
	method := strings.ToUpper(*m)

	// set content-type
	header := make(http.Header)
	header.Set("Content-Type", *contentType)
	// set any other additional headers
	if *headers != "" {
		usageAndExit("flag '-h' is deprecated, please use '-H' instead.")
	}
	// set any other additional repeatable headers
	for _, h := range headerslice {
		match, err := parseInputWithRegexp(h, headerRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		header.Set(match[1], match[2])
	}

	if *accept != "" {
		header.Set("Accept", *accept)
	}

	// set basic auth if set
	var username, password string
	if *authHeader != "" {
		match, err := parseInputWithRegexp(*authHeader, authRegexp)
		if err != nil {
			usageAndExit(err.Error())
		}
		username, password = match[1], match[2]
	}

	if *output != "csv" && *output != "" {
		usageAndExit("Invalid output type; only csv is supported.")
	}

	var proxyURL *gourl.URL
	if *proxyAddr != "" {
		var err error
		proxyURL, err = gourl.Parse(*proxyAddr)
		if err != nil {
			usageAndExit(err.Error())
		}
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		usageAndExit(err.Error())
	}
	req.Header = header
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	sineWave := genSineProfile(*sP, *f, *aS * float64(q))

	(&boomer.Boomer{
		Request:            req,
		RequestBody:        *body,
		N:                  num,
		C:                  conc,
		Qps:                q,
		Timeout:            *t,
		DisableCompression: *disableCompression,
		DisableKeepAlives:  *disableKeepAlives,
		ProxyAddr:          proxyURL,
		Output:             *output,
		WaveForm: 			sineWave,
		SamplePeriod: 		time.Duration(*aS) * time.Second,
	}).Run()
}

//Returns 1 period of a Sine waveform values at the specified sampling rate and frequency
func genSineProfile(samplePeriod float64, frequency float64, ampScale float64) ([]int32) {
	var (
		samplesPerSecond	float64  = 1 / float64(samplePeriod)                  
		phase 				float64
		radiansPerSample 	float64 = float64(frequency * 2 * math.Pi / float64(samplesPerSecond))
		//Number of samples just enough to generate one period
		numberOfSamples		uint32 = uint32(2 * math.Pi / float64(radiansPerSample))
		waveform			[]int32 = make([]int32, numberOfSamples)
	)

	for sample := uint32(0); sample < numberOfSamples; sample++ {
		sampleValue := float64(ampScale) * 0.5 * float64( 1 + math.Sin(phase) + 0.1)//0.1 is needed to avoid a divide by zero in boomer
		waveform[sample] = int32(sampleValue)
		phase += radiansPerSample
	}

	return waveform
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func parseInputWithRegexp(input, regx string) ([]string, error) {
	re := regexp.MustCompile(regx)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 1 {
		return nil, fmt.Errorf("could not parse the provided input; input = %v", input)
	}
	return matches, nil
}
