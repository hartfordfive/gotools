package main


import(
	"fmt"
	"net/url"
	"net/http"
	"strconv"
	"os"
	"io/ioutil"
	//"runtime"
	"flag"
	"time"
	"sync"
	"runtime"
	"sort"
	"math"
	"math/rand"
	"strings"
	"bufio"
)



type BenchOptions struct{
     Method string
     UserAgent string
     Header map[string]string
     Timeout int
     PostData map[string]string
     TimeBetween int
     TotalTests int
     Concurency int
     UrlList []string
     UrlListLen int
     Url string
}


type BenchTest struct{
     TimeTaken int64
     StatusCode int
}


type BenchStats struct{
     TestCount int
     TotalTime int64
     TestTime []int
     AvgTime int
     MedianTime int
     UrlListTestCount map[string]int
     StatusCode map[string]int
     TestStart int64
     TestEnd int64
     NumFail int
     NumPass int
     BytesDownloaded int
     ServerType string
}

func SortResponseTimes(sl []int) []int {
     sort.Sort(sort.IntSlice(sl))
     return sl
}

func FromNanoToMilli(ts int64) int{
     return int(ts/1000000)
}

func Readln(r *bufio.Reader) (string, error) {
  var (isPrefix bool = true
       err error = nil
       line, ln []byte
      )
  for isPrefix && err == nil {
      line, isPrefix, err = r.ReadLine()
      ln = append(ln, line...)
  }
  return string(ln),err
}


func check(e error) {
    if e != nil {
        panic(e)
    }
}


const debug bool = false


var options = BenchOptions{
    Method: "GET",
    UserAgent: "Mozilla/5.0 GoBench",
    Timeout: 2,
    TimeBetween: 100,
    TotalTests: 30,
    UrlList: nil,
}


/*
	Should resemble:

	Num CPU cores used: 1
	Total URL variations: 12
	Server type:  nginx/1.0.14
	Total tests run:  9
	Total bytes downloaded:  132095 (128 KB)
	Total pass:  9
	Total fail:  0
	Total responses in 2xx: 9
	Total responses in 3xx: 0
	Total responses in 4xx: 0
	Total responses in 5xx: 0
	Shortest time:  26 ms
	Longest time:  541 ms
	Median time:  325 ms
	Avg. time: 307 ms
*/

func dumpToReportFile(bs *BenchStats, bo *BenchOptions, fileNamePrefix string) int{

     ts := int(time.Now().Unix())
     y,m,d := time.Now().Date()

    fh1, err1 := os.Create(fileNamePrefix + "_general_report_" + strconv.Itoa(y) + m.String() + strconv.Itoa(d) + "_" + strconv.Itoa(ts) + ".txt")
    check(err1)
    fh2, err2 := os.Create(fileNamePrefix + "_url_hit_report_" + strconv.Itoa(y) + m.String() + strconv.Itoa(d) + "_" + strconv.Itoa(ts) + ".txt")
    check(err2)
    fh3, err3 := os.Create(fileNamePrefix + "_time_report_" + strconv.Itoa(y) + m.String() + strconv.Itoa(d) + "_" + strconv.Itoa(ts) + ".txt")
    check(err3)
    defer fh1.Close()
    defer fh2.Close()
    defer fh3.Close()
    

    var out string
    out = "\nStress Testing Report\n"
    out += "Num CPU cores used: " + string(bo.Concurency) + "\n"

    if bo.UrlListLen >= 1 {
       out += "Total URL variations: " + string(bo.UrlListLen) + "\n"
    } else {
      out += "URL tested: " + bo.Url + "\n"
    }


    out += "Server Type: " + string(bs.ServerType) + "\n\n"

    out += "Total tests: " + string(bs.TestCount) + "\n"
    out += "Total bytes downloaded: " + string(bs.BytesDownloaded) + "\n"
    out += "Total passed: " + string(bs.NumPass) + "\n"
    out += "Total failed: " + string(bs.NumFail) + "\n"
    out += "\t2xx responses: " + string(bs.StatusCode["2xx"]) + "\n"
    out += "\t3xx responses: " + string(bs.StatusCode["3xx"]) + "\n"
    out += "\t4xx responses: " + string(bs.StatusCode["4xx"]) + "\n"
    out += "\t5xx responses: " + string(bs.StatusCode["5xx"]) + "\n\n"

    out += "Shortest time: " + string(bs.TestTime[0]) + "ms\n"
    out += "Longest time: " + string(bs.TestTime[len(bs.TestTime)-1]) + "ms\n"
    out += "Median time: " + string(bs.MedianTime) + "\n"
    out += "Average time: " + string(bs.AvgTime) + "\n\n"
    
    //out += "" + string() + "\n"    
    totalBytes := 0


    // --------------- Write the 1st report file
    nb, _ := fh1.WriteString(out)
    totalBytes += nb
    fh1.Sync()

    // --------------- Write the 2nd report file with the number of hits to each url
    out = "URL,Hits\n"
    for k,v := range bs.UrlListTestCount {
        out += k + "," + strconv.Itoa(v) + "\n"
    }
    nb, _ = fh2.WriteString(out)
    totalBytes += nb
    fh2.Sync()

    // --------------- Write the 3rd report file with the number of hits to each url
    out = "";
    for i := 0; i < bs.TestCount; i++ {
        out += strconv.Itoa(bs.TestTime[i])
	if i < (bs.TestCount-1) {
	   out += ","
	}
    }
    nb, _ = fh3.WriteString(out)
    totalBytes += nb
    fh3.Sync()

   

    return totalBytes
}




func loadPostData(inFile string) map[string]string {
    //var pd map[string]string
    pd := make(map[string]string)
    f, err := os.Open(inFile)
    if err == nil {     
        r := bufio.NewReader(f)
	 for s, e := Readln(r); e == nil; s, e = Readln(r)  {
             // Read a line from the file
	     if s == "" || len(s) < 2 { goto goreturn }
	     parts := strings.SplitN(s, "=", 2)	    
             if len(parts) == 2 {
		pd[strings.Trim(parts[0], " ")] = string(strings.Trim(parts[1], " "))
             }
	 }
	 goreturn:
	 return pd
    }
     return nil
}


func loadUrlList(inFile string) []string {
    var ul []string  
    f, err := os.Open(inFile)
    if err == nil {
        r := bufio.NewReader(f)
         for s, e := Readln(r); e == nil; s, e = Readln(r)  {
             // Read a line from the file
             if s == "" { continue }
             ul = append(ul, strings.Trim(s, " "))
         }
         return ul
    }
    return nil
}


func makeRequest(urlToCall string, opt *BenchOptions, stats *BenchStats) { //w *sync.WaitGroup) {

     client := &http.Client{}     

     values := make(url.Values)
      if len(opt.PostData) >= 1 {
        opt.Method = "POST"
        for k,v := range opt.PostData {
            values.Add(k, v)
        }
     }


     var req *http.Request

     if opt.Method == "POST" {
          req, _ = http.NewRequest(opt.Method, urlToCall, strings.NewReader(values.Encode()) )
      } else {
       	 req, _ = http.NewRequest(opt.Method, urlToCall, nil)
      }
     
     req.Header.Add("User-Agent", opt.UserAgent)
     if len(opt.Header) >= 1 {
     	if len(opt.PostData) >= 1 {
	    req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	}

     	req.Header.Add("Connection", "Keep-alive")
        for k,v := range opt.Header {
            req.Header.Add(k, v)
        }
     }


     tStart := time.Now().UnixNano()
     resp, _ := client.Do(req)     


     body, _ := ioutil.ReadAll(resp.Body)
     stats.BytesDownloaded += len(body)
     body = nil

      switch {
            case resp.StatusCode >= 200 && resp.StatusCode < 300:
                    stats.StatusCode["2xx"]++
            case resp.StatusCode >= 300 && resp.StatusCode < 400:
                    stats.StatusCode["3xx"]++
            case resp.StatusCode >= 400 && resp.StatusCode < 500:
                    stats.StatusCode["4xx"]++
            case resp.StatusCode >= 500:
                    stats.StatusCode["5xx"]++
        }



     if resp.StatusCode != 200 {
     	stats.TestTime = append(stats.TestTime, 0)
	stats.TestCount++
	stats.NumFail++
     	return
     }

     defer resp.Body.Close()
     tEnd := time.Now().UnixNano()
          
     stats.TestTime = append(stats.TestTime, FromNanoToMilli(tEnd-tStart)) // convert to milliseconds
     stats.TestCount++
     stats.NumPass++


	if resp.Header.Get("Server") != "" {
	   stats.ServerType = resp.Header.Get("Server")
	}
 
     //fmt.Println(resp.Header)

     //fmt.Println("Status Code:", resp.StatusCode)
     //fmt.Println("\x0cOn", ((stats.TestCount*100)/opt.TotalTests), "%")
     //fmt.Println(((stats.TestCount*100)/opt.TotalTests), "% complete")
     

     if debug {
     	fmt.Println("\tRequest took ", (tEnd-tStart), "ns to run")
     }

}




func main() {

     var url, postDataFile, urlList string
     var threads,totalTests, maxCores, rampFactor, rampTime int
     var once [4]sync.Once


     flag.StringVar(&url, "u", "http://www.somedomain.com", "Full url to test")
     flag.IntVar(&threads, "c", 1, "Number of threads to run concurrently")
     flag.IntVar(&maxCores, "p", 1, "Number of processor cores to use")
     flag.IntVar(&totalTests, "m", 25, "Total number of tests to run")
     //flag.StringVar(&type, "t", "http", "Type of test to run (http/mc/redis/mysql)")
     flag.IntVar(&rampFactor, "rf", 1, "The number of thread to gradually ramp up by")
     flag.IntVar(&rampTime, "rt", 1, "The number of seconds to wait until the next ramp up")
     flag.StringVar(&postDataFile, "pd", "", "Enable POST request and use specifid data file")
     flag.StringVar(&urlList, "l", "", "File containing the list of urls to test")

     flag.Parse()



     if maxCores > runtime.NumCPU() {
     	runtime.GOMAXPROCS(runtime.NumCPU())
     } else {
        runtime.GOMAXPROCS(maxCores)
     }

     
     if postDataFile == "" {
     	options.Method = "GET"
	options.PostData = nil

	if urlList != "" {
	   options.UrlList = loadUrlList(urlList)
	   options.UrlListLen = len(options.UrlList)
	}

     } else {
       options.Method = "POST"
       options.PostData = loadPostData(postDataFile)
       fmt.Println("Post data:", options.PostData)
       if options.PostData == nil {
       	  options.Method = "GET"
	  fmt.Println("Warning: Post data file "+postDataFile + " does not exist or has no data!")
       }
     }


     if options.UrlList != nil  {
     	fmt.Println("\nRunning tests on", options.UrlListLen, "different URLS randomly")
     } else {
       fmt.Println("\nRunning tests on "+url)
     }
     
     
     options.TotalTests = totalTests
     options.Concurency = threads
     
     statusCodes := map[string]int{"2xx":0,"3xx":0,"4xx":0,"5xx":0}
     stats := BenchStats{TestStart: time.Now().UnixNano(), BytesDownloaded: 0, ServerType: "", StatusCode: statusCodes, UrlListTestCount: map[string]int{}}     
     //if urlList != nil {
     //	 stats.UrlListTestCount = make(map[string]int)
     //}

     i := totalTests
     done := make(chan bool, totalTests)


     for {
     	 for j := 0; j < threads; j++ {	

	     progress := (stats.TestCount*100)/totalTests
                    switch {
                       case progress >= 25 && progress < 50:
                            once[0].Do(func() { fmt.Println("25% Completed..") })
                       case progress >= 50 && progress < 75:
                            once[1].Do(func() { fmt.Println("50% Completed..") })
                       case progress >= 75 && progress < 100:
                            once[2].Do(func() { fmt.Println("75% Completed..") })           
                    }

	    if i < 1 {
	      once[3].Do(func() { fmt.Println("100% Completed..") })
	      goto breakout
	    }
	    go func() { 
	       if options.UrlList != nil {
	       	  rand.Seed(time.Now().UnixNano())
	       	  url = options.UrlList[rand.Intn(options.UrlListLen)]
	       }	   
	       makeRequest(url, &options, &stats)
	       stats.UrlListTestCount[url]++
	       done <- true   
	    }()	    
     	    i--
	 } 
	 	 
	 if options.TimeBetween > 0 {
	    //time.Sleep(int64(options.TimeBetween) * int64(time.Millisecond))
	    //time.Sleep(options.TimeBetween * (1000*1000))
	    time.Sleep(1000 * time.Millisecond)
	 } 
	 

     }
     breakout:


     stats.TestEnd = time.Now().UnixNano()
     stats.TestTime = SortResponseTimes(stats.TestTime)


     fmt.Println("\n-------------- Test Statistics ---------------")
     fmt.Println("Num CPU cores used:", maxCores)

     if options.UrlList != nil {
     	fmt.Println("Total URL variations:", options.UrlListLen)
     } else {
       fmt.Println("URL Requested: " + url)
     }

     fmt.Println("Server type: ", stats.ServerType)
     fmt.Println("Total tests run: ", stats.TestCount)
     if stats.BytesDownloaded > 0 {
          fmt.Println("Total bytes downloaded: ", stats.BytesDownloaded, "("+strconv.Itoa((stats.BytesDownloaded/1024))+" KB)")
     }

     fmt.Println("Total pass: ", stats.NumPass)
     fmt.Println("Total fail: ", stats.NumFail)
     fmt.Println("Total responses in 2xx:", stats.StatusCode["2xx"])
     fmt.Println("Total responses in 3xx:", stats.StatusCode["3xx"])
     fmt.Println("Total responses in 4xx:", stats.StatusCode["4xx"])
     fmt.Println("Total responses in 5xx:", stats.StatusCode["5xx"])

     fmt.Println("Shortest time: ", stats.TestTime[0], "ms")
     fmt.Println("Longest time: ", stats.TestTime[len(stats.TestTime)-1], "ms")

     
     if totalTests%2==1 {
     	var index int = int(math.Ceil(float64(len(stats.TestTime)/2)))  
     	stats.MedianTime = int( stats.TestTime[index] )
     } else {
       var index int = totalTests/2
       stats.MedianTime = (stats.TestTime[index]+stats.TestTime[index+1])/2
     }
    

    for i := 0; i < len(stats.TestTime); i++ {
    	stats.AvgTime += stats.TestTime[i]
    }
    stats.AvgTime = int(stats.AvgTime/stats.TestCount)
    

     fmt.Println("Median time: ", stats.MedianTime, "ms")
     fmt.Println("Avg. time:", stats.AvgTime, "ms")

     fmt.Println("")
     dumpToReportFile(&stats, &options, "stress_test_")
     fmt.Println("For more details, please view saved reports for more details.")
     fmt.Println("")


     <- done

}