/************************************************************************************************

	This package reads an access log which has the following format:

	{REMOTE_IP} - - [{DATE_TIME}] {REQUESTED_RESOURCE} "{HTTP_STATUS_CODE}" {BYTES_SENT} "{HTTP_REFERER}" "{USER_AGENT}"


*************************************************************************************************/
package main

import (      
       "bufio"
       "fmt"
       "os"
       "regexp"
       "strconv"
       "time"
       "sort"
       "encoding/json"
       "github.com/abh/geoip"
       "./lib"
)

const csvd string = "~"
const geoipdb_base = "http://geolite.maxmind.com/download/geoip/database/"
const geoipdb_basic string = "GeoIP.dat"
const geoipdb_city string = "GeoLiteCity.dat"


/*****************************************************/


type Pair struct {
     Key string
     Value int
}

type PairList []Pair

type FileElement struct {
     Name string
     Ext string
     Handle *os.File
     Size int
}

func (fe *FileElement) FullName() string{
     return fe.Name+fe.Ext
}


/*****************************************************/

func (p PairList) Swap(i, j int) { 
     p[i], p[j] = p[j], p[i] 
}

func (p PairList) Len() int { 
     return len(p) 
}

func (p PairList) Less(i, j int) bool { 
     return p[i].Value < p[j].Value 
}

func sortMapByValue(m map[string]int) PairList {
   p := make(PairList, len(m))
   i := 0
   for k, v := range m {
      p[i] = Pair{k, v}
      i++
   }
   sort.Sort(p)
   return p
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


func mapPairToCSV(colHeader string, inMap PairList, fh *os.File, geo *geoip.GeoIP) int{
    
    
    nb, err := fh.WriteString(colHeader+"\n")
    check(err)
    totalBytes := nb
    i := 0

    for _, v1 := range inMap {

    	matches := regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`).FindStringSubmatch(v1.Key)
    	if len(matches) >= 1 && geo != nil {
	    	record := geo.GetRecord(v1.Key)
		if record != nil {
        	   nb, err = fh.WriteString(v1.Key + csvd + strconv.Itoa(v1.Value)+ csvd + record.City + csvd + record.CountryName + "\n")
		}
	} else {
		nb, err = fh.WriteString(v1.Key + csvd + strconv.Itoa(v1.Value)+ "\n")
	}
	check(err)
        totalBytes += nb
        i++
    }
    fh.Sync()
    return totalBytes
}


func mapPairToJson(colHeader string, inMap PairList, fh *os.File) int{

    b, err := json.Marshal(inMap)
    if err != nil {
        fmt.Println(err)
        return 0
    }
   
    nb, err := fh.WriteString(string(b))
    check(err)
    fh.Sync()
    return nb
}


func main() {

    
     if len(os.Args) < 2 {
     	fmt.Println("\nUsage: parselog [filepath] [csv|json]\n")
	os.Exit(0)
     }

     filePath := os.Args[1]
     fmt.Println("\n-------------------------------------------")
     fmt.Println("Reading file:", filePath, "\n")

    f, err := os.Open(filePath)
    if err != nil {
        fmt.Printf("Error! Could not open file: %v\n",err)
	fmt.Println("")
	os.Exit(0)
    }


    ext := ""
    if os.Args[2] == "json" {
       ext = ".json"
    } else {
      ext = ".csv"
    }


    // Attempt to update the GeoIP DB
    //fmt.Println("Downloading "+geoipdb_base+geoipdb_city)
    
    // If not updated in a month, then attempt to update the GeoIP DB files
    if tools.FileExists(geoipdb_city) == false {

       if tools.Download(geoipdb_base+geoipdb_city) {
       	  fmt.Println("Update to " +geoipdb_city + " successful")
       } else {
	  fmt.Println("Could not update "+geoipdb_city)
       }            

    }
        

    // Now open the GeoIP database
    //var geo *geoip.GeoIP 
    //if tools.FileExists(geoipdb_city) == true {
        geo, geoErr := geoip.Open(geoipdb_city)
    	if geoErr != nil {
	       fmt.Printf("Warning, could not open GeoIP database: %s\n", err)
	}
    //} else {
    //  fmt.Println("File doesn't exist!")
    //}


    r := bufio.NewReader(f)
    s, e := Readln(r)

    lineNum := 0
    mapIP := make(map[string]int, 10)
    mapURI := make(map[string]int, 10)
    mapUA := make(map[string]int, 10)



    for e == nil {

    	// Read a line from the file
	s,e = Readln(r)

	// Attempt to extrac the IP and store it inn a map
	matches := regexp.MustCompile(`^([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`).FindStringSubmatch(s)
	if(len(matches) >= 2){		
		mapIP[matches[1]] = mapIP[matches[1]]+1
	}
	/***********************************************************/
	
	// Attempt to extract and store the URI in a map
	matches = regexp.MustCompile(`"(GET|POST)[^\\"]*(\\"[^\\"]*)*"`).FindStringSubmatch(s)
        if(len(matches) >= 1){	
		key := regexp.MustCompile("(GET|POST|\"|HTTP/1.[0-1])").ReplaceAllString(matches[0], "")
		mapURI[key] = mapURI[key]+1
	}

	/***********************************************************/

	// Attempt to extract and store the Useragent in a map
        matches = regexp.MustCompile(`"[^\\"]*(\\"[^\\"]*)*"$`).FindStringSubmatch(s)
        if(len(matches) >= 1){
                mapUA[regexp.MustCompile("\"").ReplaceAllString(matches[0], "")] = mapUA[matches[0]]+1
        }
	
	/***********************************************************/
	
	lineNum++
	matches = nil
    }




    /*********** Now sort the maps by value descending using merge sort ******/

    sMapIP := sortMapByValue(mapIP)
    mapIP = nil

    sMapURI := sortMapByValue(mapURI)
    mapURI = nil

    sMapUA := sortMapByValue(mapUA)    
    mapUA = nil
    
    /************* Now write the data to a csv file ***************/

    ts := int(time.Now().Unix())
    y,m,d := time.Now().Date()
    
    fileList := [3]FileElement{
    	     	{Name: "logfile_results_"+strconv.Itoa(y)+""+m.String()+""+strconv.Itoa(d)+"_"+strconv.Itoa(ts)+"_ip", Ext: ext, Size: 0},
		{Name: "logfile_results_"+strconv.Itoa(y)+""+m.String()+""+strconv.Itoa(d)+"_"+strconv.Itoa(ts)+"_uri", Ext: ext, Size: 0},
	     	{Name: "logfile_results_"+strconv.Itoa(y)+""+m.String()+""+strconv.Itoa(d)+"_"+strconv.Itoa(ts)+"_ua", Ext: ext, Size: 0},
	     }

    for nf := 0; nf < 3; nf++ {     
    	fh, err := os.Create(fileList[nf].FullName())
        check(err)
	fileList[nf].Handle = fh
        defer fileList[nf].Handle.Close()
    }



    // Update each file element property
    if ext == "json" {
       fileList[0].Size = mapPairToJson("IP"+csvd+"Hits"+csvd+"City"+csvd+"Country", sMapIP, fileList[0].Handle)
    	fileList[1].Size = mapPairToJson("URI"+csvd+"Hits\n", sMapURI, fileList[1].Handle)
        fileList[2].Size = mapPairToJson("UA"+csvd+"Hits\n", sMapUA, fileList[2].Handle)
    } else {
      fileList[0].Size = mapPairToCSV("IP"+csvd+"Hits"+csvd+"City"+csvd+"Country", sMapIP, fileList[0].Handle, geo)
      fileList[1].Size = mapPairToCSV("URI"+csvd+"Hits", sMapURI, fileList[1].Handle, geo)
      fileList[2].Size = mapPairToCSV("UA"+csvd+"Hits", sMapUA, fileList[2].Handle, geo)
    }

    numLines := (lineNum-1)
    numIp := len(sMapIP)
    numUri := len(sMapURI)
    numUa := len(sMapUA)

    fmt.Println("Processing details:\n-----------------------")
    fmt.Println("Total lines in log file: ", numLines)
    fmt.Println("Total unique IPs: ", numIp)     
    fmt.Println("Total unique User-Agents: ", numUa)
    fmt.Println("Total unique URIs: ", numUri, "\n")

    fmt.Println("Top 5 IPs\n----------------------------")
    for i := (numIp-1); i > 0; i-- {
    	record := geo.GetRecord(sMapIP[i].Key)
	fmt.Println("IP:", sMapIP[i].Key, "Hits:", sMapIP[i].Value, "Location:", record.City+", "+record.CountryName)
	if i == (numIp-5){
	     break	  
	}
    }
    fmt.Println("")

    fmt.Println("Top 5 URIs\n----------------------------")
    for i := (numUri-1); i > 0; i-- {
        fmt.Println("URI:", sMapURI[i].Key, "\nHits:", sMapURI[i].Value, "\n")
        if i == (numUri-5){
             break
        }
    }


    fmt.Println("Top 5 UAs\n----------------------------")
    for i := (numUa-1); i > 0; i-- {
        fmt.Println("UA:", sMapUA[i].Key, "\nHits:", sMapUA[i].Value, "\n")
        if i == (numUa-5){
             break
        }
    }
    fmt.Println("")

    fmt.Println("View the following files for full report:", "\n")
    totalBytes := 0
    for i := 0; i < 3; i++ {
    	fmt.Println("\t"+fileList[i].FullName()+" ("+strconv.Itoa(fileList[i].Size)+" bytes)")
	totalBytes += fileList[i].Size
    } 
    fmt.Println("\nTotal bytes written:", totalBytes)
    fmt.Println("\n----------------------------------\n")

}