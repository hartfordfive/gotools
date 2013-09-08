package main


import(
	"fmt"
	"net/http"
	//"os"
	//"http"
	//"io/ioutil"
	//"url"
	//"runtime"
	"flag"
	"time"
	"sync"
	//"strconv"
	"runtime"
	"sort"
	"math"
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
}


type BenchTest struct{
     TimeTaken int64
     StatusCode int
}


type BenchStats struct{
     TestCount int
     TotalTime int64
     TestTime []int
     StatusCode map[string]int
     TestStart int64
     TestEnd int64
     NumFail int
     NumPass int
     BytesDownloaded int64
     ServerType string
}


/*
type Int64Slice []int64

func (a Int64Slice) Len() int { 
     return len(a) 
}

func (a Int64Slice) Swap(i, j int) { 
     a[i], a[j] = a[j], a[i] 
}

func (a Int64Slice) Less(i, j int) bool { 
     return a[i] < a[j] 
}

func SortResponseTimes(sl []int64) Int64Slice {
     sLen := len(sl)
     s := make(Int64Slice, sLen)
     for i := 0; i < sLen; i++ {
     	 s[i] = sl[i]
     }
     sort.Sort(s)
     return s
}
*/

func SortResponseTimes(sl []int) []int {
     sort.Sort(sort.IntSlice(sl))
     return sl
}

func FromNanoToMilli(ts int64) int{
     return int(ts/1000000)
}


const debug bool = false


var options = BenchOptions{
    Method: "GET",
    UserAgent: "Mozilla/5.0 GoBench",
    Timeout: 2,
    TimeBetween: 100,
    TotalTests: 30,
}

//var signalComplete chan bool


func makeRequest(urlToCall string, opt *BenchOptions, stats *BenchStats) { //w *sync.WaitGroup) {


     client := &http.Client{}     

     //req, err := http.NewRequest(opt.Method, urlToCall, bytes.NewReader(postData))

     req, _ := http.NewRequest(opt.Method, urlToCall, nil)    
     req.Header.Add("User-Agent", opt.UserAgent)     

     if len(opt.Header) >= 1 {
	for k,v := range opt.Header {
	    req.Header.Add(k, v)
	}
     }

     tStart := time.Now().UnixNano()
     resp, _ := client.Do(req)     

     


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
 
	/*
     if resp.Header.Get("Content-Encoding") != "gzip" { 
     	stats.BytesDownloaded += resp.ContentLength
     } else {
       fmt.Println("using gzip encoding for return data!")
     }
     */

     //fmt.Println(resp.Header)

     //fmt.Println("Status Code:", resp.StatusCode)
     //fmt.Println("\x0cOn", ((stats.TestCount*100)/opt.TotalTests), "%")
     //fmt.Println(((stats.TestCount*100)/opt.TotalTests), "% complete")
     

     if debug {
     	fmt.Println("\tRequest took ", (tEnd-tStart), "ns to run")
     }

}




func main() {

     var url string
     var threads,totalTests, maxCores, rampFactor, rampTime int
     var once [4]sync.Once //oncePN25, oncePN50, oncePN75, oncePN100 sync.Once


     flag.StringVar(&url, "u", "http://www.somedomain.com", "Full url to test")
     flag.IntVar(&threads, "c", 1, "Number of threads to run concurrently")
     flag.IntVar(&maxCores, "p", 1, "Number of processor cores to use")
     flag.IntVar(&totalTests, "m", 25, "Total number of tests to run")
     //flag.StringVar(&type, "t", "http", "Type of test to run (http/mc/redis/mysql)")
     flag.IntVar(&rampFactor, "rf", 1, "The number of thread to gradually ramp up by")
     flag.IntVar(&rampTime, "rt", 1, "The number of seconds to wait until the next ramp up")
     flag.Parse()


     if maxCores > runtime.NumCPU() {
     	runtime.GOMAXPROCS(runtime.NumCPU())
     } else {
        runtime.GOMAXPROCS(maxCores)
     }

     //postDataFile := flag.String("d", "postdata.txt", "The filename of the POST data to send")

     fmt.Println("\nRunning tests on "+url)
     
     
     options.TotalTests = totalTests
     options.Concurency = threads

     statusCodes := map[string]int{"2xx":0,"3xx":0,"4xx":0,"5xx":0}
     stats := BenchStats{TestStart: time.Now().UnixNano(), BytesDownloaded: 0, ServerType: "", StatusCode: statusCodes}     


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
	       makeRequest(url, &options, &stats)
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
     fmt.Println("URL Requested: " + url)
     fmt.Println("Server type: ", stats.ServerType)
     fmt.Println("Total tests run: ", stats.TestCount)
     if stats.BytesDownloaded > 0 {
          fmt.Println("Total bytes downloaded: ", stats.BytesDownloaded)
     }

     fmt.Println("Total pass: ", stats.NumPass)
     fmt.Println("Total fail: ", stats.NumFail)
     fmt.Println("Total responses in 2xx:", stats.StatusCode["2xx"])
     fmt.Println("Total responses in 3xx:", stats.StatusCode["3xx"])
     fmt.Println("Total responses in 4xx:", stats.StatusCode["4xx"])
     fmt.Println("Total responses in 5xx:", stats.StatusCode["5xx"])

     fmt.Println("Shortest time: ", stats.TestTime[0], "ms")
     fmt.Println("Longest time: ", stats.TestTime[len(stats.TestTime)-1], "ms")

     var median,avg int
     if totalTests%2==1 {
     	var index int = int(math.Ceil(float64(len(stats.TestTime)/2)))  
     	median = int( stats.TestTime[index] )
     } else {
       var index int = totalTests/2
       median = (stats.TestTime[index]+stats.TestTime[index+1])/2
     }
    

    for i := 0; i < len(stats.TestTime); i++ {
    	avg += stats.TestTime[i]
    }
    avg = int(avg/stats.TestCount)
    

     fmt.Println("Median time: ", median, "ms")
     fmt.Println("Avg. time:", avg, "ms")


     fmt.Println("")

     //fmt.Println("Test start: ", (stats.TestStart%1e6)/1e3)
     //fmt.Println("Test end: ", (stats.TestEnd%1e6)/1e3)
     //fmt.Println("Total time taken: ", ((stats.TestEnd-stats.TestStart)%1e6)/1e3)
     fmt.Println("")


     <- done

}