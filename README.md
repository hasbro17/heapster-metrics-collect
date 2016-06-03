# heapster-metrics-collect

CoreOS Intern Project(05/23/16 -- 06/03/16)

metrics-collect is a program that extracts resource usage metrics from Heapster running in a kubernetes cluster via the kubernetes API through the kubectl proxy.
It also generates chart files for to be plotted via the [gochart](https://github.com/zieckey/gochart) utility.

To run:
```
./metrics-collect <heapster-resolution> <time-interval-in-minutes> <chart-type>
```
where `heapster-resolution` is the time period at which Heapster collect metrics and `time-interval-in-minutes` is the duration over which the metrics need to be extracted.
The interval will be set as `[currentTime - m, currentTime]` where m is the interval duration.

To view the chart files as plots follow the instructions on [gochart](https://github.com/zieckey/gochart)

sine-boom is the original [boom](https://github.com/rakyll/boom) program slightly modified to generate a sinusoidal load.
The sampling period and frequency of the sinusoid are passed in as additional flags to the program.
