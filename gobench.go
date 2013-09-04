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
	"strconv"
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



const debug bool = false


var options = BenchOptions{
    Method: "GET",
    UserAgent: "Mozilla/5.0 GoBench",
    Timeout: 2,
    TimeBetween: 100,
    TotalTests: 30,
}

//var signalComplete chan bool


func makeRequest(urlToCall string, opt *BenchOptions, stats *BenchStats, w *sync.WaitGroup) {


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

     if resp.StatusCode != 200 {
     	stats.TestTime = append(stats.TestTime, 0)
	stats.TestCount++
	stats.NumFail++
     	return
     }

     defer resp.Body.Close()
     tEnd := time.Now().UnixNano()
     
     stats.TestTime = append(stats.TestTime, int((tEnd-tStart)/1000)) // convert to milliseconds
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

     w.Done()


     //signalComplete <- true    

}




func main() {

     var url string
     var threads,totalTests, maxCores int


     flag.StringVar(&url, "u", "http://www.somedomain.com", "Full url to test")
     flag.IntVar(&threads, "c", 1, "Number of threads to run concurrently")
     flag.IntVar(&maxCores, "p", 1, "Number of processor cores to use")
     flag.IntVar(&totalTests, "m", 25, "Total number of tests to run")
     //flag.StringVar(&type, "t", "http", "Type of test to run (http/mc/redis/mysql)")
     flag.Parse()


     if maxCores > runtime.NumCPU() {
      	fmt.Println("Using "+strconv.Itoa(runtime.NumCPU())+" cores")
     	runtime.GOMAXPROCS(runtime.NumCPU())
     } else {
        fmt.Println("Using "+strconv.Itoa(maxCores)+" cores")
        runtime.GOMAXPROCS(maxCores)
     }

     //postDataFile := flag.String("d", "postdata.txt", "The filename of the POST data to send")

     fmt.Println("-----------------------------------------------------")
     fmt.Println("Requesting " + url + " a max of " + strconv.Itoa(totalTests)+" times")


     
     options.TotalTests = totalTests
     options.Concurency = threads

     stats := BenchStats{TestStart: time.Now().UnixNano(), BytesDownloaded: 0, ServerType: ""}     

     
     //signalComplete := make(chan bool)

     var w sync.WaitGroup
     //w.Add(totalTests)


     i := totalTests
     w.Add(totalTests)
     for i > 0 {
     	 for j := 0; j < threads; j++ {	
	    if i < 1 {
	       break
	    }
	    go makeRequest(url, &options, &stats, &w)	    
     	    i--
	 } 
	 	 
	 fmt.Println(stats.TestCount, "Tests completed")

	 if options.TimeBetween > 0 {
	    //time.Sleep(int64(options.TimeBetween) * int64(time.Millisecond))
	    //time.Sleep(options.TimeBetween * (1000*1000))
	    time.Sleep(1000 * time.Millisecond)
	 } 


     }
     stats.TestEnd = time.Now().UnixNano()

     stats.TestTime = SortResponseTimes(stats.TestTime)

     w.Wait()
     //<-signalComplete

     // Clear screen
     //fmt.Println("\x0c\n")

     //time.Sleep(12 * 1e9)
     fmt.Println("-------------- Test Statistics ---------------")
     fmt.Println("Server type: ", stats.ServerType)
     fmt.Println("Total tests run: ", stats.TestCount)
     if stats.BytesDownloaded > 0 {
          fmt.Println("Total bytes downloaded: ", stats.BytesDownloaded)
     }

     fmt.Println("Total pass: ", stats.NumPass)
     fmt.Println("Total fail: ", stats.NumFail)
     fmt.Println("Shortest time: ", 1)
     fmt.Println("Longest time: ", 1000)

     var median int
     if totalTests%2==1 {
     	var index int = int(math.Ceil(float64(len(stats.TestTime)/2)))  
     	median = int( stats.TestTime[index] )
     } else {
       var index int = totalTests/2
       median = (stats.TestTime[index]+stats.TestTime[index+1])/2
     }
    
     fmt.Println("Median time: ", median, "ms")

     fmt.Println("")
     fmt.Println(stats.TestTime)


     //fmt.Println("Test start: ", (stats.TestStart%1e6)/1e3)
     //fmt.Println("Test end: ", (stats.TestEnd%1e6)/1e3)
     //fmt.Println("Total time taken: ", ((stats.TestEnd-stats.TestStart)%1e6)/1e3)


     fmt.Println()


}