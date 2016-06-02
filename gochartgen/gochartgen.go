//Script to generate chart files for the gochart plotter

package gochartgen

import "strings"
import "strconv"
import "os"

//Error check helper
func check(e error) {
    if e != nil {
        panic(e)
    }
}

// Create a time series chart file to be drawn by gochart (https://github.com/zieckey/gochart)
// yAxisData[line number][values]
func CreateTimeSeriesChartFile(fileName string, chartType string, xAxisData []int, yAxisData [][]int, yAxisLineNames []string, yAxisText string){
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



// Create a time series chart file to be drawn by gochart (https://github.com/zieckey/gochart)
// yAxisData[line number][values]
func CreateTimeSeriesChartFileTS(fileName string, chartType string, xAxisTS []string, yAxisData [][]int, yAxisLineNames []string, yAxisText string){
	str := "ChartType = " + chartType +"\n" + 
	"Title = " + fileName + "\n"+
	"SubTitle = \n"+
	"\nXAxisNumbers = "

	for i, xTS := range xAxisTS {
		str += xTS[len(xTS)-2:]
		if i != len(xAxisTS) - 1 {
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






//Float version
func CreateTimeSeriesChartFileFloat(fileName string, chartType string, xAxisData []float64, yAxisData [][]float64, yAxisLineNames []string, yAxisText string){
	str := "ChartType = " + chartType +"\n" + 
	"Title = " + fileName + "\n"+
	"SubTitle = \n"+
	"\nXAxisNumbers = "
	
	//Append X axis numbers
	for i, x := range xAxisData {
		str += strconv.FormatFloat(x, 'f', 3, 64)
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
			str += strconv.FormatFloat(y, 'f', 3, 64)
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